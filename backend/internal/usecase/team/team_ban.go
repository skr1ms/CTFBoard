package team

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-logkit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

const maxBanRetries = 3

//nolint:funlen // transaction and post-commit steps are kept in one place
func (uc *TeamUseCase) BanTeam(ctx context.Context, teamID uuid.UUID, reason string, banMembers bool, actorID uuid.UUID) error {
	var memberIDs []uuid.UUID
	var bannedUserIDs []uuid.UUID
	var err error
	for attempt := 0; attempt < maxBanRetries; attempt++ {
		err = uc.deps.TM.Run(ctx, func(ctx context.Context) error {
			members, err := uc.deps.UserRepo.GetByTeamID(ctx, teamID)
			if err != nil {
				return fmt.Errorf("TeamUseCase - BanTeam - UserRepo.GetByTeamID: %w", err)
			}
			slices.SortFunc(members, func(a, b *domain.User) int {
				return strings.Compare(a.ID.String(), b.ID.String())
			})
			for _, m := range members {
				if err := uc.deps.UserRepo.Lock(ctx, m.ID); err != nil {
					return fmt.Errorf("TeamUseCase - BanTeam - UserRepo.Lock: %w", err)
				}
			}
			if err := uc.deps.TeamRepo.Lock(ctx, teamID); err != nil {
				return fmt.Errorf("TeamUseCase - BanTeam - TeamRepo.Lock: %w", err)
			}
			membersAfter, err := uc.deps.UserRepo.GetByTeamID(ctx, teamID)
			if err != nil {
				return fmt.Errorf("TeamUseCase - BanTeam - UserRepo.GetByTeamID (recheck): %w", err)
			}
			slices.SortFunc(membersAfter, func(a, b *domain.User) int {
				return strings.Compare(a.ID.String(), b.ID.String())
			})
			if len(membersAfter) != len(members) {
				return httperr.ErrTeamConflict
			}
			for i := range members {
				if members[i].ID != membersAfter[i].ID {
					return httperr.ErrTeamConflict
				}
			}
			team, err := uc.deps.TeamRepo.GetByID(ctx, teamID)
			if err != nil {
				return fmt.Errorf("TeamUseCase - BanTeam - TeamRepo.GetByID: %w", err)
			}
			if team.IsBanned {
				return nil
			}
			ids := make([]uuid.UUID, len(members))
			for i, m := range members {
				ids[i] = m.ID
			}
			memberIDs = ids
			challengeIDs, err := uc.getChallengeIDsForTeam(ctx, teamID)
			if err != nil {
				return fmt.Errorf("TeamUseCase - BanTeam - getChallengeIDsForTeam: %w", err)
			}
			if err := uc.deps.TeamRepo.Ban(ctx, teamID, reason); err != nil {
				return fmt.Errorf("TeamUseCase - BanTeam - TeamRepo.Ban: %w", err)
			}
			if err := uc.deps.SolveRepo.SoftBanByTeamID(ctx, teamID); err != nil {
				return fmt.Errorf("TeamUseCase - BanTeam - SolveRepo.SoftBanByTeamID: %w", err)
			}
			if err := uc.deps.SubmissionRepo.SoftBanByTeamID(ctx, teamID); err != nil {
				return fmt.Errorf("TeamUseCase - BanTeam - SubmissionRepo.SoftBanByTeamID: %w", err)
			}
			if err := uc.deps.AwardRepo.SoftBanByTeamID(ctx, teamID); err != nil {
				return fmt.Errorf("TeamUseCase - BanTeam - AwardRepo.SoftBanByTeamID: %w", err)
			}
			if uc.deps.HintRepo != nil {
				if err := uc.deps.HintRepo.SoftBanUnlocksByTeamID(ctx, teamID); err != nil {
					return fmt.Errorf("TeamUseCase - BanTeam - HintRepo.SoftBanUnlocksByTeamID: %w", err)
				}
			}
			if err := uc.adjustSolveCountsForChallenges(ctx, challengeIDs, true); err != nil {
				return fmt.Errorf("TeamUseCase - BanTeam - adjustSolveCountsForChallenges: %w", err)
			}
			var fieldValuesBackup []map[string]string
			if uc.deps.FieldValueRepo != nil {
				fvList, err := uc.deps.FieldValueRepo.GetByEntityID(ctx, teamID)
				if err != nil {
					return fmt.Errorf("TeamUseCase - BanTeam - FieldValueRepo.GetByEntityID: %w", err)
				}
				if len(fvList) > 0 {
					fieldValuesBackup = make([]map[string]string, len(fvList))
					for i, fv := range fvList {
						fieldValuesBackup[i] = map[string]string{"field_id": fv.FieldID.String(), "value": fv.Value}
					}
				}
				if err := uc.deps.FieldValueRepo.DeleteByEntityID(ctx, teamID); err != nil {
					return fmt.Errorf("TeamUseCase - BanTeam - FieldValueRepo.DeleteByEntityID: %w", err)
				}
			}
			if banMembers {
				for _, m := range members {
					if m.Role == domain.RoleAdmin {
						continue
					}
					if err := uc.deps.UserRepo.Ban(ctx, m.ID, reason); err != nil {
						return fmt.Errorf("TeamUseCase - BanTeam - UserRepo.Ban: %w", err)
					}
					bannedUserIDs = append(bannedUserIDs, m.ID)
				}
				if err := uc.deps.UserRepo.UpdateTeamIDBatch(ctx, memberIDs, nil); err != nil {
					return fmt.Errorf("TeamUseCase - BanTeam - UserRepo.UpdateTeamIDBatch: %w", err)
				}
			}
			if !banMembers {
				var nonAdminIDs []uuid.UUID
				for _, m := range members {
					if m.Role != domain.RoleAdmin {
						nonAdminIDs = append(nonAdminIDs, m.ID)
					}
				}
				if len(nonAdminIDs) > 0 {
					if err := uc.deps.UserRepo.SetWasInBannedTeamByIDs(ctx, nonAdminIDs, true); err != nil {
						return fmt.Errorf("TeamUseCase - BanTeam - UserRepo.SetWasInBannedTeamByIDs: %w", err)
					}
				}
			}
			memberIDStrings := make([]string, len(memberIDs))
			for i, id := range memberIDs {
				memberIDStrings[i] = id.String()
			}
			details := map[string]any{"reason": reason, "member_ids": memberIDStrings}
			if banMembers {
				details["ban_members"] = true
				bannedStrings := make([]string, len(bannedUserIDs))
				for i, id := range bannedUserIDs {
					bannedStrings[i] = id.String()
				}
				details["banned_user_ids"] = bannedStrings
			}
			if len(fieldValuesBackup) > 0 {
				details["field_values"] = fieldValuesBackup
			}
			auditLog := &domain.TeamAuditLog{
				TeamID:  teamID,
				UserID:  &actorID,
				Action:  domain.TeamActionBanned,
				Details: details,
			}
			if err := uc.deps.TeamRepo.CreateAuditLog(ctx, auditLog); err != nil {
				return fmt.Errorf("TeamUseCase - BanTeam - TeamRepo.CreateAuditLog: %w", err)
			}
			return nil
		})
		if err == nil {
			break
		}
		if !errors.Is(err, httperr.ErrTeamConflict) {
			break
		}
	}
	if err != nil {
		return fmt.Errorf("TeamUseCase - BanTeam - TM.Run: %w", err)
	}
	postCtx := context.WithoutCancel(ctx)
	for _, id := range memberIDs {
		uc.invalidateUserCache(postCtx, id)
	}
	if banMembers {
		for _, id := range memberIDs {
			if uc.deps.JWTRevoker != nil {
				if err := uc.deps.JWTRevoker.RevokeAllForUser(postCtx, id); err != nil && uc.deps.Logger != nil {
					uc.deps.Logger.WithError(err).Warn("TeamUseCase - BanTeam - RevokeAllForUser", logkit.UserID(id.String()))
				}
			}
		}
	}
	uc.invalidateTeamCache(postCtx, teamID)
	uc.invalidateScoreboardCacheForTeam(postCtx, teamID)
	uc.invalidateChallengeListCache(postCtx)
	return nil
}

func parseUUIDSliceFromDetails(details map[string]any, key string) []uuid.UUID {
	raw, ok := details[key]
	if !ok || raw == nil {
		return nil
	}
	strSlice := toStringSlice(raw)
	var ids []uuid.UUID
	for _, s := range strSlice {
		if id, err := uuid.Parse(s); err == nil {
			ids = append(ids, id)
		}
	}
	return ids
}

func toStringSlice(raw any) []string {
	if raw == nil {
		return nil
	}
	if s, ok := raw.([]string); ok {
		return s
	}
	if s, ok := raw.([]interface{}); ok {
		out := make([]string, 0, len(s))
		for _, v := range s {
			if str, ok := v.(string); ok {
				out = append(out, str)
			}
		}
		return out
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	var out []string
	if err := json.Unmarshal(encoded, &out); err != nil {
		return nil
	}
	return out
}

func parseFieldValuesFromDetails(details map[string]any, key string) map[string]string {
	raw, ok := details[key]
	if !ok || raw == nil {
		return nil
	}
	slice, ok := raw.([]interface{})
	if !ok {
		return nil
	}
	out := make(map[string]string)
	for _, item := range slice {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		fid, okF := m["field_id"].(string)
		val, okV := m["value"].(string)
		if okF && fid != "" {
			if okV {
				out[fid] = val
			} else {
				out[fid] = ""
			}
		}
	}
	return out
}

func (uc *TeamUseCase) unbanTeamMembersByLog(ctx context.Context, banLog *domain.TeamAuditLog, memberIDs *[]uuid.UUID) error {
	var banMembers bool
	if banLog != nil && banLog.Details != nil {
		if b, ok := banLog.Details["ban_members"].(bool); ok {
			banMembers = b
		}
	}
	if !banMembers || banLog == nil || banLog.Details == nil {
		return nil
	}
	bannedByTeamBan := parseUUIDSliceFromDetails(banLog.Details, "banned_user_ids")
	bannedSet := make(map[uuid.UUID]struct{}, len(bannedByTeamBan))
	for _, id := range bannedByTeamBan {
		bannedSet[id] = struct{}{}
	}
	if len(bannedSet) > 0 {
		for _, id := range *memberIDs {
			if _, wasBannedByTeam := bannedSet[id]; !wasBannedByTeam {
				continue
			}
			u, err := uc.deps.UserRepo.GetByID(ctx, id)
			if err != nil || u == nil || !u.IsBanned {
				continue
			}
			if err := uc.deps.UserRepo.Unban(ctx, id); err != nil {
				return fmt.Errorf("TeamUseCase - UnbanTeam - UserRepo.Unban: %w", err)
			}
		}
		return nil
	}
	teamBanAt := banLog.CreatedAt
	for _, id := range *memberIDs {
		u, err := uc.deps.UserRepo.GetByID(ctx, id)
		if err != nil || u == nil {
			continue
		}
		if !u.IsBanned || u.BannedAt == nil {
			continue
		}
		if !u.BannedAt.Before(teamBanAt) {
			if err := uc.deps.UserRepo.Unban(ctx, id); err != nil {
				return fmt.Errorf("TeamUseCase - UnbanTeam - UserRepo.Unban: %w", err)
			}
		}
	}
	return nil
}

//nolint:funlen // unban flow: restore, recalc, reassign members
func (uc *TeamUseCase) unbanTeamTx(ctx context.Context, teamID, actorID uuid.UUID, memberIDs *[]uuid.UUID) error {
	banLog, err := uc.deps.TeamRepo.GetLatestAuditLogByTeamIDAndAction(ctx, teamID, string(domain.TeamActionBanned))
	if err != nil {
		return fmt.Errorf("TeamUseCase - UnbanTeam - TeamRepo.GetLatestAuditLogByTeamIDAndAction: %w", err)
	}
	if banLog != nil && banLog.Details != nil {
		*memberIDs = parseUUIDSliceFromDetails(banLog.Details, "member_ids")
	}
	if len(*memberIDs) == 0 {
		uc.deps.Logger.Warn("TeamUseCase - UnbanTeam - no member_ids in ban audit log; team will be unbanned without restoring members", logkit.Fields{"team_id": teamID.String()})
	}
	slices.SortFunc(*memberIDs, func(a, b uuid.UUID) int {
		return strings.Compare(a.String(), b.String())
	})
	for _, id := range *memberIDs {
		if err := uc.deps.UserRepo.Lock(ctx, id); err != nil {
			return fmt.Errorf("TeamUseCase - UnbanTeam - UserRepo.Lock: %w", err)
		}
	}
	if err := uc.unbanTeamMembersByLog(ctx, banLog, memberIDs); err != nil {
		return err
	}
	freeMembers, err := uc.deps.UserRepo.FilterIDsByTeamIDNullAndNotBanned(ctx, *memberIDs)
	if err != nil {
		return fmt.Errorf("TeamUseCase - UnbanTeam - UserRepo.FilterIDsByTeamIDNullAndNotBanned: %w", err)
	}
	slices.SortFunc(freeMembers, func(a, b uuid.UUID) int {
		return strings.Compare(a.String(), b.String())
	})
	comp, err := uc.deps.CompRepo.Get(ctx)
	if err != nil && !errors.Is(err, httperr.ErrCompetitionNotFound) {
		return fmt.Errorf("TeamUseCase - UnbanTeam - CompRepo.Get: %w", err)
	}
	maxSize := uc.deps.DefaultMaxTeamSize
	if comp != nil && comp.MaxTeamSize > 0 {
		maxSize = comp.MaxTeamSize
	}
	if maxSize > 0 && len(freeMembers) > maxSize {
		freeMembers = freeMembers[:maxSize]
	}
	if err := uc.deps.TeamRepo.Lock(ctx, teamID); err != nil {
		return fmt.Errorf("TeamUseCase - UnbanTeam - TeamRepo.Lock: %w", err)
	}
	team, err := uc.deps.TeamRepo.GetByID(ctx, teamID)
	if err != nil {
		return fmt.Errorf("TeamUseCase - UnbanTeam - TeamRepo.GetByID: %w", err)
	}
	if !team.IsBanned {
		return nil
	}
	if err := uc.deps.TeamRepo.Unban(ctx, teamID); err != nil {
		return fmt.Errorf("TeamUseCase - UnbanTeam - TeamRepo.Unban: %w", err)
	}
	currentMembers, err := uc.deps.UserRepo.GetByTeamID(ctx, teamID)
	if err != nil {
		return fmt.Errorf("TeamUseCase - UnbanTeam - UserRepo.GetByTeamID: %w", err)
	}
	if len(freeMembers) == 0 && len(currentMembers) == 0 {
		if err := uc.deps.TeamRepo.SetHidden(ctx, teamID, true); err != nil {
			return fmt.Errorf("TeamUseCase - UnbanTeam - TeamRepo.SetHidden: %w", err)
		}
		uc.deps.Logger.Warn("TeamUseCase - UnbanTeam - no members restored; team set hidden", logkit.Fields{"team_id": teamID.String()})
		auditLog := &domain.TeamAuditLog{
			TeamID: teamID,
			UserID: &actorID,
			Action: domain.TeamActionUnbanned,
		}
		if err := uc.deps.TeamRepo.CreateAuditLog(ctx, auditLog); err != nil {
			return fmt.Errorf("TeamUseCase - UnbanTeam - TeamRepo.CreateAuditLog: %w", err)
		}
		return nil
	}
	if err := uc.deps.SolveRepo.RestoreByBannedTeamID(ctx, teamID); err != nil {
		return fmt.Errorf("TeamUseCase - UnbanTeam - SolveRepo.RestoreByBannedTeamID: %w", err)
	}
	if err := uc.deps.SubmissionRepo.RestoreByBannedTeamID(ctx, teamID); err != nil {
		return fmt.Errorf("TeamUseCase - UnbanTeam - SubmissionRepo.RestoreByBannedTeamID: %w", err)
	}
	if err := uc.deps.AwardRepo.RestoreByBannedTeamID(ctx, teamID); err != nil {
		return fmt.Errorf("TeamUseCase - UnbanTeam - AwardRepo.RestoreByBannedTeamID: %w", err)
	}
	if uc.deps.HintRepo != nil {
		if err := uc.deps.HintRepo.RestoreUnlocksByBannedTeamID(ctx, teamID); err != nil {
			return fmt.Errorf("TeamUseCase - UnbanTeam - HintRepo.RestoreUnlocksByBannedTeamID: %w", err)
		}
	}
	if uc.deps.FieldValueRepo != nil && banLog != nil && banLog.Details != nil {
		fvMap := parseFieldValuesFromDetails(banLog.Details, "field_values")
		if len(fvMap) > 0 {
			if err := uc.deps.FieldValueRepo.SetValues(ctx, teamID, fvMap); err != nil {
				return fmt.Errorf("TeamUseCase - UnbanTeam - FieldValueRepo.SetValues: %w", err)
			}
		}
	}
	if err := uc.adjustSolveCountsForTeam(ctx, teamID, false); err != nil {
		return fmt.Errorf("TeamUseCase - UnbanTeam - adjustSolveCountsForTeam: %w", err)
	}
	freeMembers, err = uc.deps.UserRepo.FilterIDsByTeamIDNullAndNotBanned(ctx, *memberIDs)
	if err != nil {
		return fmt.Errorf("TeamUseCase - UnbanTeam - UserRepo.FilterIDsByTeamIDNullAndNotBanned: %w", err)
	}
	slices.SortFunc(freeMembers, func(a, b uuid.UUID) int {
		return strings.Compare(a.String(), b.String())
	})
	if maxSize > 0 && len(freeMembers) > maxSize {
		freeMembers = freeMembers[:maxSize]
	}
	if len(freeMembers) > 0 {
		if err := uc.deps.UserRepo.UpdateTeamIDBatch(ctx, freeMembers, &teamID); err != nil {
			return fmt.Errorf("TeamUseCase - UnbanTeam - UserRepo.UpdateTeamIDBatch: %w", err)
		}
		captainInRestored := false
		for _, id := range freeMembers {
			if id == team.CaptainID {
				captainInRestored = true
				break
			}
		}
		if !captainInRestored {
			if err := uc.deps.TeamRepo.UpdateCaptain(ctx, teamID, freeMembers[0]); err != nil {
				return fmt.Errorf("TeamUseCase - UnbanTeam - TeamRepo.UpdateCaptain: %w", err)
			}
		}
	} else if len(currentMembers) == 0 {
		if err := uc.deps.TeamRepo.SetHidden(ctx, teamID, true); err != nil {
			return fmt.Errorf("TeamUseCase - UnbanTeam - TeamRepo.SetHidden: %w", err)
		}
		uc.deps.Logger.Warn("TeamUseCase - UnbanTeam - no members restored; team set hidden", logkit.Fields{"team_id": teamID.String()})
	}
	if len(*memberIDs) > 0 {
		if err := uc.deps.UserRepo.SetWasInBannedTeamByIDs(ctx, *memberIDs, false); err != nil {
			return fmt.Errorf("TeamUseCase - UnbanTeam - UserRepo.SetWasInBannedTeamByIDs: %w", err)
		}
	}
	auditLog := &domain.TeamAuditLog{
		TeamID: teamID,
		UserID: &actorID,
		Action: domain.TeamActionUnbanned,
	}
	if err := uc.deps.TeamRepo.CreateAuditLog(ctx, auditLog); err != nil {
		return fmt.Errorf("TeamUseCase - UnbanTeam - TeamRepo.CreateAuditLog: %w", err)
	}
	return nil
}

func (uc *TeamUseCase) UnbanTeam(ctx context.Context, teamID, actorID uuid.UUID) error {
	var memberIDs []uuid.UUID
	err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		return uc.unbanTeamTx(ctx, teamID, actorID, &memberIDs)
	})
	if err != nil {
		return fmt.Errorf("TeamUseCase - UnbanTeam - TM.Run: %w", err)
	}
	for _, id := range memberIDs {
		uc.invalidateUserCache(ctx, id)
	}
	uc.invalidateTeamCache(ctx, teamID)
	uc.invalidateScoreboardCacheForTeam(ctx, teamID)
	uc.invalidateChallengeListCache(ctx)
	return nil
}
