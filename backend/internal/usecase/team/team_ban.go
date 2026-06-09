package team

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-logkit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/txctx"
)

const maxBanRetries = 3

// BanTeam bans a team and cascades the ban across all associated data.
// The transaction is retried up to maxBanRetries times on ErrTeamConflict, which
// occurs when the member list changes between the pre-lock snapshot and the re-read
// inside the lock (a TOCTOU guard). Within the transaction: all member rows are locked
// in lexicographic UUID order to prevent deadlocks, the team is marked banned, solves
// and submissions are soft-banned (hidden from the scoreboard), awards and hint unlocks
// are soft-banned, and solve counts are recalculated via adjustSolveCountsForChallenges.
// When banMembers is true each non-admin member is individually banned and their team
// membership is cleared; when false, members are flagged with WasInBannedTeam instead.
// The complete member list and any banned user IDs are recorded in a restorable audit
// log entry. After the transaction, JWT tokens are revoked (best-effort) and all
// relevant caches are invalidated.
func (uc *TeamUseCase) BanTeam(ctx context.Context, teamID uuid.UUID, reason string, banMembers bool, actorID uuid.UUID) error {
	var (
		result teamBanTxResult
		err    error
	)

	for range maxBanRetries {
		err = uc.deps.TM.Run(ctx, func(ctx context.Context) error {
			var txErr error

			result, txErr = uc.banTeamTx(ctx, teamID, reason, banMembers, actorID)

			return txErr
		})
		if err == nil {
			break
		}

		if !errors.Is(err, apperr.ErrTeamConflict) {
			break
		}
	}

	if err != nil {
		return fmt.Errorf("TeamUseCase - BanTeam - TM.Run: %w", err)
	}

	txctx.AfterCommitOrNow(ctx, func(ctx context.Context) {
		uc.afterTeamBanCommit(ctx, []uuid.UUID{teamID}, result, banMembers)
	})

	return nil
}

type teamBanTxResult struct {
	memberIDs     []uuid.UUID
	bannedUserIDs []uuid.UUID
	changed       bool
}

//nolint:funlen // shared tx helper keeps single and bulk team ban semantics identical.
func (uc *TeamUseCase) banTeamTx(ctx context.Context, teamID uuid.UUID, reason string, banMembers bool, actorID uuid.UUID) (teamBanTxResult, error) {
	var result teamBanTxResult

	members, err := uc.lockTeamWithMembers(ctx, teamID)
	if err != nil {
		return result, fmt.Errorf("TeamUseCase - banTeamTx - lockTeamWithMembers: %w", err)
	}

	team, err := uc.deps.TeamRepo.GetByID(ctx, teamID)
	if err != nil {
		return result, fmt.Errorf("TeamUseCase - banTeamTx - TeamRepo.GetByID: %w", err)
	}

	if team.IsBanned {
		return result, nil
	}

	ids := make([]uuid.UUID, len(members))
	for i, m := range members {
		ids[i] = m.ID
	}

	result.memberIDs = ids

	challengeIDs, err := uc.getChallengeIDsForTeam(ctx, teamID)
	if err != nil {
		return result, fmt.Errorf("TeamUseCase - banTeamTx - getChallengeIDsForTeam: %w", err)
	}

	if err := uc.deps.TeamRepo.Ban(ctx, teamID, reason); err != nil {
		return result, fmt.Errorf("TeamUseCase - banTeamTx - TeamRepo.Ban: %w", err)
	}

	result.changed = true

	if err := uc.cascadeSoftBan(ctx, teamID); err != nil {
		return result, fmt.Errorf("TeamUseCase - banTeamTx - cascadeSoftBan: %w", err)
	}

	if err := uc.adjustSolveCountsForChallenges(ctx, challengeIDs); err != nil {
		return result, fmt.Errorf("TeamUseCase - banTeamTx - adjustSolveCountsForChallenges: %w", err)
	}

	var fieldValuesBackup []map[string]string

	if uc.deps.FieldValueRepo != nil {
		fvList, err := uc.deps.FieldValueRepo.GetByEntityID(ctx, teamID)
		if err != nil {
			return result, fmt.Errorf("TeamUseCase - banTeamTx - FieldValueRepo.GetByEntityID: %w", err)
		}

		if len(fvList) > 0 {
			fieldValuesBackup = make([]map[string]string, len(fvList))
			for i, fv := range fvList {
				fieldValuesBackup[i] = map[string]string{"field_id": fv.FieldID.String(), "value": fv.Value}
			}
		}

		if err := uc.deps.FieldValueRepo.DeleteByEntityID(ctx, teamID); err != nil {
			return result, fmt.Errorf("TeamUseCase - banTeamTx - FieldValueRepo.DeleteByEntityID: %w", err)
		}
	}

	if banMembers {
		for _, m := range members {
			if m.Role == domain.RoleAdmin || m.IsBanned {
				continue
			}

			err := uc.deps.UserRepo.Ban(ctx, m.ID, reason)
			if err != nil {
				return result, fmt.Errorf("TeamUseCase - banTeamTx - UserRepo.Ban: %w", err)
			}

			result.bannedUserIDs = append(result.bannedUserIDs, m.ID)
		}

		err := uc.deps.UserRepo.UpdateTeamIDBatch(ctx, result.memberIDs, nil)
		if err != nil {
			return result, fmt.Errorf("TeamUseCase - banTeamTx - UserRepo.UpdateTeamIDBatch: %w", err)
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
			err := uc.deps.UserRepo.SetWasInBannedTeamByIDs(ctx, nonAdminIDs, true)
			if err != nil {
				return result, fmt.Errorf("TeamUseCase - banTeamTx - UserRepo.SetWasInBannedTeamByIDs: %w", err)
			}
		}
	}

	memberIDStrings := make([]string, len(result.memberIDs))
	for i, id := range result.memberIDs {
		memberIDStrings[i] = id.String()
	}

	details := map[string]any{domain.TeamAuditDetailReason: reason, "member_ids": memberIDStrings}

	if banMembers {
		details["ban_members"] = true

		bannedStrings := make([]string, len(result.bannedUserIDs))
		for i, id := range result.bannedUserIDs {
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
		return result, fmt.Errorf("TeamUseCase - banTeamTx - TeamRepo.CreateAuditLog: %w", err)
	}

	return result, nil
}

func (uc *TeamUseCase) afterTeamBanCommit(ctx context.Context, teamIDs []uuid.UUID, result teamBanTxResult, banMembers bool) {
	if banMembers {
		for _, id := range domain.UniqueUUIDs(result.bannedUserIDs) {
			if uc.deps.JWTRevoker != nil {
				err := uc.deps.JWTRevoker.RevokeAllForUser(ctx, id)
				if err != nil {
					uc.deps.Logger.WithError(err).Warn("TeamUseCase - BanTeam - RevokeAllForUser", logkit.UserID(id.String()))
				}
			}
		}
	}

	for _, teamID := range domain.UniqueUUIDs(teamIDs) {
		uc.invalidateTeamAndMembers(ctx, teamID, result.memberIDs)
	}
}
