package team

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-logkit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/computil"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/ctxutil"
)

// unbanTeamMembersByLog unbans the members who were banned as part of the original team
// ban, leaving any independently banned members untouched. It reads the ban_members flag
// and banned_user_ids list from the audit log's Details. When banned_user_ids is present
// only those specific users are unbanned (provided they are still banned). When the list
// is absent but ban_members was true, the function falls back to a timestamp heuristic
// any member whose BannedAt is on or after the team ban timestamp is considered to have
// been banned as a side effect and is unbanned.
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
			err := uc.deps.UserRepo.Unban(ctx, id)
			if err != nil {
				return fmt.Errorf("TeamUseCase - UnbanTeam - UserRepo.Unban: %w", err)
			}
		}
	}

	return nil
}

// unbanTeamTx reverses a team ban inside a transaction. It reads the most recent ban
// audit log to reconstruct the original member list, locks those user rows in
// lexicographic order, and calls unbanTeamMembersByLog to restore individually banned
// members. Only members who are currently free (no team, not banned) are re-added
// the MaxTeamSize constraint is enforced by truncating the candidates list. If the
// original captain is not among the restored members the first available free member is
// promoted to captain. Solves, submissions, awards, and hint unlocks are restored from
// their soft-banned state, custom field values are reinstated from the audit log snapshot,
// and adjustSolveCountsForTeam recalculates scoring. If no members can be restored
// the team is set hidden rather than active. An unban audit log entry is written in all
// non-error paths
//
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

	domain.SortUUIDs(*memberIDs)

	for _, id := range *memberIDs {
		err := uc.deps.UserRepo.Lock(ctx, id)
		if err != nil {
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

	domain.SortUUIDs(freeMembers)

	comp, err := uc.deps.CompRepo.Get(ctx)
	if err != nil && !errors.Is(err, apperr.ErrCompetitionNotFound) {
		return fmt.Errorf("TeamUseCase - UnbanTeam - CompRepo.Get: %w", err)
	}

	maxSize := resolveMaxTeamSize(comp, uc.deps.DefaultMaxTeamSize)

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
		err := uc.deps.TeamRepo.SetHidden(ctx, teamID, true)
		if err != nil {
			return fmt.Errorf("TeamUseCase - UnbanTeam - TeamRepo.SetHidden: %w", err)
		}

		uc.deps.Logger.Warn("TeamUseCase - UnbanTeam - no members restored; team set hidden", logkit.Fields{"team_id": teamID.String()})

		auditLog := &domain.TeamAuditLog{
			TeamID: teamID,
			UserID: &actorID,
			Action: domain.TeamActionUnbanned,
		}

		err = uc.deps.TeamRepo.CreateAuditLog(ctx, auditLog)
		if err != nil {
			return fmt.Errorf("TeamUseCase - UnbanTeam - TeamRepo.CreateAuditLog: %w", err)
		}

		return nil
	}

	if err := uc.cascadeRestore(ctx, teamID); err != nil {
		return fmt.Errorf("TeamUseCase - UnbanTeam - cascadeRestore: %w", err)
	}

	if uc.deps.FieldValueRepo != nil && banLog != nil && banLog.Details != nil {
		fvMap := parseFieldValuesFromDetails(banLog.Details, "field_values")
		if len(fvMap) > 0 {
			err := uc.deps.FieldValueRepo.SetValues(ctx, teamID, fvMap)
			if err != nil {
				return fmt.Errorf("TeamUseCase - UnbanTeam - FieldValueRepo.SetValues: %w", err)
			}
		}
	}

	if err := uc.adjustSolveCountsForTeam(ctx, teamID); err != nil {
		return fmt.Errorf("TeamUseCase - UnbanTeam - adjustSolveCountsForTeam: %w", err)
	}

	freeMembers, err = uc.deps.UserRepo.FilterIDsByTeamIDNullAndNotBanned(ctx, *memberIDs)
	if err != nil {
		return fmt.Errorf("TeamUseCase - UnbanTeam - UserRepo.FilterIDsByTeamIDNullAndNotBanned: %w", err)
	}

	domain.SortUUIDs(freeMembers)

	if maxSize > 0 && len(freeMembers) > maxSize {
		freeMembers = freeMembers[:maxSize]
	}

	if len(freeMembers) > 0 {
		err := uc.deps.UserRepo.UpdateTeamIDBatch(ctx, freeMembers, &teamID)
		if err != nil {
			return fmt.Errorf("TeamUseCase - UnbanTeam - UserRepo.UpdateTeamIDBatch: %w", err)
		}

		captainInRestored := slices.Contains(freeMembers, team.CaptainID)

		if !captainInRestored {
			err := uc.deps.TeamRepo.UpdateCaptain(ctx, teamID, freeMembers[0])
			if err != nil {
				return fmt.Errorf("TeamUseCase - UnbanTeam - TeamRepo.UpdateCaptain: %w", err)
			}
		}
	} else if len(currentMembers) == 0 {
		err := uc.deps.TeamRepo.SetHidden(ctx, teamID, true)
		if err != nil {
			return fmt.Errorf("TeamUseCase - UnbanTeam - TeamRepo.SetHidden: %w", err)
		}

		uc.deps.Logger.Warn("TeamUseCase - UnbanTeam - no members restored; team set hidden", logkit.Fields{"team_id": teamID.String()})
	}

	if len(*memberIDs) > 0 {
		err := uc.deps.UserRepo.SetWasInBannedTeamByIDs(ctx, *memberIDs, false)
		if err != nil {
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

	postCtx, postCancel := ctxutil.PostCommitContext(ctx)
	defer postCancel()

	comp := computil.Cached(postCtx, nil, uc.deps.CompRepo)
	frozen := comp != nil && comp.IsFreezeActive()
	uc.invalidateTeamAndMembers(postCtx, teamID, memberIDs, frozen)

	return nil
}
