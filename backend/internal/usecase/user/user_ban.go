package user

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-logkit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/scoring"
	"github.com/TakuyaYagam1/AstroCTFb/internal/txctx"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/cacheutil"
)

// BanUser bans the specified user. The operation refuses to ban the actor
// themselves or any admin. Inside a transaction it: soft-bans the user's
// submissions; removes all of the user's solves from every team they belong to
// and recalculates solve counts and dynamic scores for affected challenges; for a
// solo team hides the team and soft-bans its awards and hint unlocks, while for a
// non-solo team it removes the user from the team, reassigns captaincy if needed
// (choosing the lexicographically smallest eligible member), and hides the team
// if all remaining members are also banned. After the transaction it revokes all
// JWTs for the banned user, invalidates user and scoreboard caches, and
// optionally sends a personal notification to the team captain when the team
// drops below the configured minimum size.
func (uc *UserUseCase) BanUser(ctx context.Context, userID uuid.UUID, reason string, actorID uuid.UUID) error {
	if userID == actorID {
		return apperr.ErrAccessDenied
	}

	var result userBanTxResult

	if err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		var err error

		result, err = uc.banUserTx(ctx, userID, reason)
		if err != nil {
			return err
		}

		return nil
	}); err != nil {
		return fmt.Errorf("UserUseCase - BanUser - TM.Run: %w", err)
	}

	txctx.AfterCommitOrNow(ctx, func(ctx context.Context) { uc.afterUserBanCommit(ctx, []uuid.UUID{userID}, result) })

	return nil
}

type userBanTxResult struct {
	scoreboardInvalidateTeamIDs []uuid.UUID
	captainIDsToNotify          []uuid.UUID
	changed                     bool
}

//nolint:funlen // shared tx helper keeps single and bulk user ban semantics identical.
func (uc *UserUseCase) banUserTx(ctx context.Context, userID uuid.UUID, reason string) (userBanTxResult, error) {
	var result userBanTxResult

	if err := uc.deps.UserRepo.Lock(ctx, userID); err != nil {
		return result, fmt.Errorf("UserUseCase - banUserTx - UserRepo.Lock: %w", err)
	}

	u, err := uc.deps.UserRepo.GetByID(ctx, userID)
	if err != nil {
		return result, fmt.Errorf("UserUseCase - banUserTx - UserRepo.GetByID: %w", err)
	}

	if u.Role == domain.RoleAdmin {
		return result, apperr.ErrAccessDenied
	}

	if u.IsBanned {
		return result, nil
	}

	if err := uc.deps.UserRepo.Ban(ctx, userID, reason); err != nil {
		return result, fmt.Errorf("UserUseCase - banUserTx - UserRepo.Ban: %w", err)
	}

	result.changed = true

	if uc.deps.SubmissionRepo != nil {
		if err := uc.deps.SubmissionRepo.SoftBanByUserID(ctx, userID); err != nil {
			return result, fmt.Errorf("UserUseCase - banUserTx - SubmissionRepo.SoftBanByUserID: %w", err)
		}
	}

	if uc.deps.SolveRepo != nil {
		affectedSolves, err := uc.deps.SolveRepo.GetModerationAffectedSolvesByUserID(ctx, userID)
		if err != nil {
			return result, fmt.Errorf("UserUseCase - banUserTx - SolveRepo.GetModerationAffectedSolvesByUserID: %w", err)
		}

		for _, group := range groupModerationAffectedSolvesByTeam(affectedSolves) {
			if err := uc.banUserRemoveSolvesAndAdjustScoresForChallenges(ctx, group.teamID, userID, group.challengeIDs); err != nil {
				return result, fmt.Errorf("UserUseCase - banUserTx - banUserRemoveSolvesAndAdjustScores: %w", err)
			}

			result.scoreboardInvalidateTeamIDs = append(result.scoreboardInvalidateTeamIDs, group.teamID)
		}
	}

	if u.TeamID != nil && uc.deps.TeamRepo != nil {
		if err := uc.deps.TeamRepo.Lock(ctx, *u.TeamID); err != nil {
			return result, fmt.Errorf("UserUseCase - banUserTx - TeamRepo.Lock: %w", err)
		}

		team, err := uc.deps.TeamRepo.GetByID(ctx, *u.TeamID)
		if err == nil && team != nil {
			if team.IsSolo && !team.IsHidden {
				if err := uc.deps.TeamRepo.SetHidden(ctx, team.ID, true); err != nil {
					return result, fmt.Errorf("UserUseCase - banUserTx - TeamRepo.SetHidden: %w", err)
				}

				if uc.deps.AwardRepo != nil {
					if err := uc.deps.AwardRepo.SoftBanByTeamID(ctx, team.ID); err != nil {
						return result, fmt.Errorf("UserUseCase - banUserTx - AwardRepo.SoftBanByTeamID: %w", err)
					}
				}

				if uc.deps.HintRepo != nil {
					if err := uc.deps.HintRepo.SoftBanUnlocksByTeamID(ctx, team.ID); err != nil {
						return result, fmt.Errorf("UserUseCase - banUserTx - HintRepo.SoftBanUnlocksByTeamID: %w", err)
					}
				}

				result.scoreboardInvalidateTeamIDs = append(result.scoreboardInvalidateTeamIDs, team.ID)
			} else {
				if err := uc.deps.UserRepo.UpdateTeamID(ctx, userID, nil); err != nil {
					return result, fmt.Errorf("UserUseCase - banUserTx - UserRepo.UpdateTeamID: %w", err)
				}

				auditLog := &domain.TeamAuditLog{
					TeamID:  team.ID,
					UserID:  &userID,
					Action:  domain.TeamActionMemberBanned,
					Details: map[string]any{domain.TeamAuditDetailReason: "user_banned"},
				}
				if err := uc.deps.TeamRepo.CreateAuditLog(ctx, auditLog); err != nil {
					return result, fmt.Errorf("UserUseCase - banUserTx - TeamRepo.CreateAuditLog: %w", err)
				}

				currentCaptainID := team.CaptainID
				if team.CaptainID == userID {
					remaining, errRem := uc.deps.UserRepo.GetByTeamID(ctx, team.ID)
					if errRem == nil && len(remaining) > 0 {
						var eligible []*domain.User

						for _, u := range remaining {
							if !u.IsBanned {
								eligible = append(eligible, u)
							}
						}

						if len(eligible) > 0 {
							domain.SortUsersByID(eligible)

							newCaptainID := eligible[0].ID
							if err := uc.deps.TeamRepo.UpdateCaptain(ctx, team.ID, newCaptainID); err != nil {
								return result, fmt.Errorf("UserUseCase - banUserTx - TeamRepo.UpdateCaptain: %w", err)
							}

							currentCaptainID = newCaptainID
						} else {
							if err := uc.deps.TeamRepo.SetHidden(ctx, team.ID, true); err != nil {
								return result, fmt.Errorf("UserUseCase - banUserTx - TeamRepo.SetHidden all banned: %w", err)
							}
						}
					} else if errRem == nil && len(remaining) == 0 {
						if err := uc.deps.TeamRepo.SetHidden(ctx, team.ID, true); err != nil {
							return result, fmt.Errorf("UserUseCase - banUserTx - TeamRepo.SetHidden empty team: %w", err)
						}
					}
				}

				if uc.deps.CompRepo != nil {
					comp, errComp := uc.deps.CompRepo.Get(ctx)
					if errComp == nil && comp != nil && comp.MinTeamSize > 0 {
						count, errCount := uc.deps.TeamRepo.CountTeamMembers(ctx, team.ID)
						if errCount == nil && count > 0 && count < comp.MinTeamSize {
							uc.deps.Logger.Warn("team below MinTeamSize after user ban", logkit.Fields{"team_id": team.ID.String(), "member_count": count, "min_team_size": comp.MinTeamSize})

							result.captainIDsToNotify = append(result.captainIDsToNotify, currentCaptainID)
						}
					}
				}

				result.scoreboardInvalidateTeamIDs = append(result.scoreboardInvalidateTeamIDs, team.ID)
			}
		}
	}

	return result, nil
}

func (uc *UserUseCase) afterUserBanCommit(ctx context.Context, userIDs []uuid.UUID, result userBanTxResult) {
	for _, userID := range domain.UniqueUUIDs(userIDs) {
		if uc.deps.JWTService != nil {
			if err := uc.deps.JWTService.RevokeAllForUser(ctx, userID); err != nil {
				uc.deps.Logger.WithError(err).Warn("UserUseCase - BanUser - RevokeAllForUser")
			}
		}

		cacheutil.InvalidateUser(ctx, uc.deps.UserCache, userID)
	}

	for _, teamID := range domain.UniqueUUIDs(result.scoreboardInvalidateTeamIDs) {
		cacheutil.InvalidateScoreboardForTeam(ctx, uc.deps.ScoreboardCache, teamID)
		cacheutil.InvalidateTeam(ctx, uc.deps.TeamCache, uc.deps.Logger, teamID)
		cacheutil.InvalidateChallengeList(ctx, uc.deps.ChallengeListCache)
	}

	if result.changed {
		cacheutil.InvalidateStatistics(ctx, uc.deps.StatsCache, uc.deps.Logger, "UserUseCase - BanUser")
	}

	if uc.deps.PersonalNotificationSender == nil {
		return
	}

	for _, captainID := range domain.UniqueUUIDs(result.captainIDsToNotify) {
		_, err := uc.deps.PersonalNotificationSender.CreatePersonal(ctx, usecase.NotificationCreatePersonalParams{
			UserID:  captainID,
			Title:   "Team below minimum size",
			Content: "A member of your team was banned. Your team is now below the minimum size required to submit. Please add members or contact an administrator.",
			Type:    domain.NotificationWarning,
		})
		if err != nil {
			uc.deps.Logger.WithError(err).Warn("UserUseCase - BanUser - CreatePersonal notification failed")
		}
	}
}

// UnbanUser reverses a direct user ban. Inside a transaction it unsets the ban flag
// and restores soft-banned submissions. Team-inherited ban state is owned by team
// unban and is not cleared here. For a
// solo team that was hidden at ban time it unhides the team, restores the user's
// solves with score recalculation, and restores soft-banned awards and hint
// unlocks. For a user whose team_id is nil (was removed from a non-solo team on
// ban) it restores the solves so the former team regains the associated points
// the user must re-join the team manually. After the transaction the user cache
// and all affected scoreboard caches are invalidated.
func (uc *UserUseCase) UnbanUser(ctx context.Context, userID, actorID uuid.UUID) error {
	if userID == actorID {
		return apperr.ErrAccessDenied
	}

	var result userBanRestoreTxResult

	if err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		var err error

		result, err = uc.restoreUserBanTx(ctx, userID, true)
		if err != nil {
			return err
		}

		return nil
	}); err != nil {
		return fmt.Errorf("UserUseCase - UnbanUser - TM.Run: %w", err)
	}

	txctx.AfterCommitOrNow(ctx, func(ctx context.Context) {
		uc.invalidateUserBanRestore(ctx, []uuid.UUID{userID}, result.scoreboardInvalidateTeamIDs)
	})

	return nil
}

type userBanRestoreTxResult struct {
	scoreboardInvalidateTeamIDs []uuid.UUID
	changed                     bool
}

func (uc *UserUseCase) restoreUserBanTx(ctx context.Context, userID uuid.UUID, rejectInheritedOnly bool) (userBanRestoreTxResult, error) {
	var result userBanRestoreTxResult

	if err := uc.deps.UserRepo.Lock(ctx, userID); err != nil {
		return result, fmt.Errorf("UserUseCase - restoreUserBanTx - UserRepo.Lock: %w", err)
	}

	u, err := uc.deps.UserRepo.GetByID(ctx, userID)
	if err != nil {
		return result, fmt.Errorf("UserUseCase - restoreUserBanTx - UserRepo.GetByID: %w", err)
	}

	if !u.IsBanned {
		if rejectInheritedOnly && u.WasInBannedTeam && u.Role != domain.RoleAdmin {
			return result, apperr.NewValidationErrorf("team-inherited ban must be cleared by unbanning the team")
		}

		return result, nil
	}

	if err := uc.deps.UserRepo.Unban(ctx, userID); err != nil {
		return result, fmt.Errorf("UserUseCase - restoreUserBanTx - UserRepo.Unban: %w", err)
	}

	result.changed = true

	if uc.deps.SubmissionRepo != nil {
		if err := uc.deps.SubmissionRepo.RestoreByBannedUserID(ctx, userID); err != nil {
			return result, fmt.Errorf("UserUseCase - restoreUserBanTx - SubmissionRepo.RestoreByBannedUserID: %w", err)
		}
	}

	if u.TeamID != nil && uc.deps.TeamRepo != nil {
		if err := uc.deps.TeamRepo.Lock(ctx, *u.TeamID); err != nil {
			return result, fmt.Errorf("UserUseCase - restoreUserBanTx - TeamRepo.Lock: %w", err)
		}

		team, err := uc.deps.TeamRepo.GetByID(ctx, *u.TeamID)
		if err == nil && team != nil && team.IsSolo && team.IsHidden {
			if err := uc.deps.TeamRepo.SetHidden(ctx, team.ID, false); err != nil {
				return result, fmt.Errorf("UserUseCase - restoreUserBanTx - TeamRepo.SetHidden: %w", err)
			}

			restoredTeamIDs, err := uc.unbanUserRestoreSolvesAndAdjustScores(ctx, userID)
			if err != nil {
				return result, fmt.Errorf("UserUseCase - restoreUserBanTx - unbanUserRestoreSolvesAndAdjustScores: %w", err)
			}

			if uc.deps.AwardRepo != nil {
				if err := uc.deps.AwardRepo.RestoreByBannedTeamID(ctx, team.ID); err != nil {
					return result, fmt.Errorf("UserUseCase - restoreUserBanTx - AwardRepo.RestoreByBannedTeamID: %w", err)
				}
			}

			if uc.deps.HintRepo != nil {
				if err := uc.deps.HintRepo.RestoreUnlocksByBannedTeamID(ctx, team.ID); err != nil {
					return result, fmt.Errorf("UserUseCase - restoreUserBanTx - HintRepo.RestoreUnlocksByBannedTeamID: %w", err)
				}
			}

			result.scoreboardInvalidateTeamIDs = append(result.scoreboardInvalidateTeamIDs, team.ID)
			result.scoreboardInvalidateTeamIDs = append(result.scoreboardInvalidateTeamIDs, restoredTeamIDs...)

			return result, nil
		}
	}

	if u.TeamID == nil && uc.deps.SolveRepo != nil {
		restoredTeamIDs, err := uc.unbanUserRestoreSolvesAndAdjustScores(ctx, userID)
		if err != nil {
			return result, fmt.Errorf("UserUseCase - restoreUserBanTx - unbanUserRestoreSolvesAndAdjustScores: %w", err)
		}

		result.scoreboardInvalidateTeamIDs = append(result.scoreboardInvalidateTeamIDs, restoredTeamIDs...)
	}

	return result, nil
}

func (uc *UserUseCase) restoreAppealedUserBanTx(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	result, err := uc.restoreUserBanTx(ctx, userID, true)
	if err != nil {
		return nil, err
	}

	return result.scoreboardInvalidateTeamIDs, nil
}

func (uc *UserUseCase) invalidateAppealedUserBanRestore(ctx context.Context, userID uuid.UUID, teamIDs []uuid.UUID) {
	uc.invalidateUserBanRestore(ctx, []uuid.UUID{userID}, teamIDs)
}

func (uc *UserUseCase) invalidateUserBanRestore(ctx context.Context, userIDs, teamIDs []uuid.UUID) {
	for _, userID := range domain.UniqueUUIDs(userIDs) {
		cacheutil.InvalidateUser(ctx, uc.deps.UserCache, userID)
	}

	for _, teamID := range domain.UniqueUUIDs(teamIDs) {
		cacheutil.InvalidateScoreboardForTeam(ctx, uc.deps.ScoreboardCache, teamID)
		cacheutil.InvalidateTeam(ctx, uc.deps.TeamCache, uc.deps.Logger, teamID)
		cacheutil.InvalidateChallengeList(ctx, uc.deps.ChallengeListCache)
	}

	cacheutil.InvalidateStatistics(ctx, uc.deps.StatsCache, uc.deps.Logger, "UserUseCase - UnbanUser")
}

func (uc *UserUseCase) banUserRemoveSolvesAndAdjustScoresForChallenges(ctx context.Context, teamID, userID uuid.UUID, challengeIDs []uuid.UUID) error {
	if uc.deps.SolveRepo == nil || uc.deps.ChallengeRepo == nil {
		return nil
	}

	challengeIDs = domain.UniqueUUIDs(challengeIDs)

	if len(challengeIDs) > 0 {
		if err := uc.deps.SolveRepo.SoftBanByTeamIDAndUserID(ctx, teamID, userID); err != nil {
			return fmt.Errorf("UserUseCase - banUserRemoveSolvesAndAdjustScoresForChallenges - SolveRepo.SoftBanByTeamIDAndUserID: %w", err)
		}
	}

	return uc.adjustDynamicScores(ctx, challengeIDs)
}

// unbanUserRestoreSolvesAndAdjustScores restores soft-banned solve rows for a user,
// then recalculates solve counts, static point values, and per-solve decay snapshots
// for all affected challenges. The mirror inverse of banUserRemoveSolvesAndAdjustScores.
func (uc *UserUseCase) unbanUserRestoreSolvesAndAdjustScores(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	if uc.deps.SolveRepo == nil || uc.deps.ChallengeRepo == nil {
		return nil, nil
	}

	affectedSolves, err := uc.deps.SolveRepo.GetModerationAffectedSolvesByBannedUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("UserUseCase - unbanUserRestoreSolvesAndAdjustScores - SolveRepo.GetModerationAffectedSolvesByBannedUserID: %w", err)
	}

	if err := uc.deps.SolveRepo.RestoreByBannedUserID(ctx, userID); err != nil {
		return nil, fmt.Errorf("UserUseCase - unbanUserRestoreSolvesAndAdjustScores - SolveRepo.RestoreByBannedUserID: %w", err)
	}

	var (
		rawChallengeIDs []uuid.UUID
		rawTeamIDs      []uuid.UUID
	)

	for _, s := range affectedSolves {
		rawChallengeIDs = append(rawChallengeIDs, s.ChallengeID)
		rawTeamIDs = append(rawTeamIDs, s.TeamID)
	}

	return domain.UniqueUUIDs(rawTeamIDs), uc.adjustDynamicScores(ctx, domain.UniqueUUIDs(rawChallengeIDs))
}

type moderationAffectedTeamSolves struct {
	teamID       uuid.UUID
	challengeIDs []uuid.UUID
}

func groupModerationAffectedSolvesByTeam(solves []*domain.ModerationAffectedSolve) []moderationAffectedTeamSolves {
	groups := make([]moderationAffectedTeamSolves, 0)
	indexByTeam := make(map[uuid.UUID]int)

	for _, solve := range solves {
		if solve == nil {
			continue
		}

		idx, ok := indexByTeam[solve.TeamID]
		if !ok {
			indexByTeam[solve.TeamID] = len(groups)
			groups = append(groups, moderationAffectedTeamSolves{teamID: solve.TeamID})
			idx = len(groups) - 1
		}

		groups[idx].challengeIDs = append(groups[idx].challengeIDs, solve.ChallengeID)
	}

	for i := range groups {
		groups[i].challengeIDs = domain.UniqueUUIDs(groups[i].challengeIDs)
	}

	return groups
}

func (uc *UserUseCase) adjustDynamicScores(ctx context.Context, challengeIDs []uuid.UUID) error {
	if len(challengeIDs) == 0 {
		return nil
	}

	getSolves := scoring.MapSolvesForRecalcFn(
		uc.deps.SolveRepo.GetSolvesForPointsRecalc,
		scoring.DefaultSolveMapper,
		"UserUseCase - adjustDynamicScores",
	)

	return scoring.AdjustDynamicScores(
		ctx, challengeIDs, uc.deps.ChallengeRepo,
		getSolves,
		uc.deps.SolveRepo.BatchUpdateSolvePoints,
		scoring.GetDecayFn(ctx, uc.deps.CompParamUC),
	)
}
