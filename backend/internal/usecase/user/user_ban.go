package user

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-logkit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/cache"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/scoring"
)

//nolint:funlen // ban flow: lock, validations, DB updates, cache invalidation, notifications
func (uc *UserUseCase) BanUser(ctx context.Context, userID uuid.UUID, reason string, actorID uuid.UUID) error {
	if userID == actorID {
		return httperr.ErrAccessDenied
	}

	var (
		scoreboardInvalidateTeamIDs []uuid.UUID
		captainIDToNotify           uuid.UUID
		shouldNotifyCaptain         bool
	)

	if err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		if err := uc.deps.UserRepo.Lock(ctx, userID); err != nil {
			return fmt.Errorf("UserUseCase - BanUser - UserRepo.Lock: %w", err)
		}

		u, err := uc.deps.UserRepo.GetByID(ctx, userID)
		if err != nil {
			return fmt.Errorf("UserUseCase - BanUser - UserRepo.GetByID: %w", err)
		}

		if u.Role == domain.RoleAdmin {
			return httperr.ErrAccessDenied
		}

		if u.IsBanned {
			return nil
		}

		if err := uc.deps.UserRepo.Ban(ctx, userID, reason); err != nil {
			return fmt.Errorf("UserUseCase - BanUser - UserRepo.Ban: %w", err)
		}

		if uc.deps.SubmissionRepo != nil {
			if err := uc.deps.SubmissionRepo.SoftBanByUserID(ctx, userID); err != nil {
				return fmt.Errorf("UserUseCase - BanUser - SubmissionRepo.SoftBanByUserID: %w", err)
			}
		}

		if uc.deps.SolveRepo != nil {
			solves, err := uc.deps.SolveRepo.GetByUserIDWithDetails(ctx, userID)
			if err != nil {
				return fmt.Errorf("UserUseCase - BanUser - SolveRepo.GetByUserIDWithDetails: %w", err)
			}

			seenTeam := make(map[uuid.UUID]struct{})

			for _, s := range solves {
				if _, ok := seenTeam[s.TeamID]; ok {
					continue
				}

				seenTeam[s.TeamID] = struct{}{}
				if err := uc.banUserRemoveSolvesAndAdjustScores(ctx, s.TeamID, userID); err != nil {
					return fmt.Errorf("UserUseCase - BanUser - banUserRemoveSolvesAndAdjustScores: %w", err)
				}

				scoreboardInvalidateTeamIDs = append(scoreboardInvalidateTeamIDs, s.TeamID)
			}
		}

		if u.TeamID != nil && uc.deps.TeamRepo != nil {
			if err := uc.deps.TeamRepo.Lock(ctx, *u.TeamID); err != nil {
				return fmt.Errorf("UserUseCase - BanUser - TeamRepo.Lock: %w", err)
			}

			team, err := uc.deps.TeamRepo.GetByID(ctx, *u.TeamID)
			if err == nil && team != nil {
				if team.IsSolo && !team.IsHidden {
					if err := uc.deps.TeamRepo.SetHidden(ctx, team.ID, true); err != nil {
						return fmt.Errorf("UserUseCase - BanUser - TeamRepo.SetHidden: %w", err)
					}

					if uc.deps.AwardRepo != nil {
						if err := uc.deps.AwardRepo.SoftBanByTeamID(ctx, team.ID); err != nil {
							return fmt.Errorf("UserUseCase - BanUser - AwardRepo.SoftBanByTeamID: %w", err)
						}
					}

					if uc.deps.HintRepo != nil {
						if err := uc.deps.HintRepo.SoftBanUnlocksByTeamID(ctx, team.ID); err != nil {
							return fmt.Errorf("UserUseCase - BanUser - HintRepo.SoftBanUnlocksByTeamID: %w", err)
						}
					}

					scoreboardInvalidateTeamIDs = append(scoreboardInvalidateTeamIDs, team.ID)
				} else {
					if err := uc.deps.UserRepo.UpdateTeamID(ctx, userID, nil); err != nil {
						return fmt.Errorf("UserUseCase - BanUser - UserRepo.UpdateTeamID: %w", err)
					}

					auditLog := &domain.TeamAuditLog{
						TeamID:  team.ID,
						UserID:  &userID,
						Action:  domain.TeamActionMemberBanned,
						Details: map[string]any{"reason": "user_banned"},
					}
					if err := uc.deps.TeamRepo.CreateAuditLog(ctx, auditLog); err != nil {
						return fmt.Errorf("UserUseCase - BanUser - TeamRepo.CreateAuditLog: %w", err)
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
								slices.SortFunc(eligible, func(a, b *domain.User) int { return strings.Compare(a.ID.String(), b.ID.String()) })

								newCaptainID := eligible[0].ID
								if err := uc.deps.TeamRepo.UpdateCaptain(ctx, team.ID, newCaptainID); err != nil {
									return fmt.Errorf("UserUseCase - BanUser - TeamRepo.UpdateCaptain: %w", err)
								}

								currentCaptainID = newCaptainID
							} else {
								if err := uc.deps.TeamRepo.SetHidden(ctx, team.ID, true); err != nil {
									return fmt.Errorf("UserUseCase - BanUser - TeamRepo.SetHidden all banned: %w", err)
								}
							}
						} else if errRem == nil && len(remaining) == 0 {
							if err := uc.deps.TeamRepo.SetHidden(ctx, team.ID, true); err != nil {
								return fmt.Errorf("UserUseCase - BanUser - TeamRepo.SetHidden empty team: %w", err)
							}
						}
					}

					if uc.deps.CompRepo != nil && uc.deps.Logger != nil {
						comp, errComp := uc.deps.CompRepo.Get(ctx)
						if errComp == nil && comp != nil && comp.MinTeamSize > 0 {
							count, errCount := uc.deps.TeamRepo.CountTeamMembers(ctx, team.ID)
							if errCount == nil && count > 0 && count < comp.MinTeamSize {
								uc.deps.Logger.Warn("team below MinTeamSize after user ban", logkit.Fields{"team_id": team.ID.String(), "member_count": count, "min_team_size": comp.MinTeamSize})

								captainIDToNotify = currentCaptainID
								shouldNotifyCaptain = true
							}
						}
					}

					scoreboardInvalidateTeamIDs = append(scoreboardInvalidateTeamIDs, team.ID)
				}
			}
		}

		return nil
	}); err != nil {
		return fmt.Errorf("UserUseCase - BanUser - TM.Run: %w", err)
	}

	postCtx := context.WithoutCancel(ctx)

	if uc.deps.JWTService != nil {
		if err := uc.deps.JWTService.RevokeAllForUser(postCtx, userID); err != nil && uc.deps.Logger != nil {
			uc.deps.Logger.WithError(err).Warn("UserUseCase - BanUser - RevokeAllForUser")
		}
	}

	if uc.deps.UserCache != nil {
		uc.deps.UserCache.InvalidateUser(postCtx, userID)
	}

	for _, teamID := range scoreboardInvalidateTeamIDs {
		if uc.deps.ScoreboardCache != nil {
			uc.deps.ScoreboardCache.InvalidateForTeam(postCtx, teamID)
		}

		if uc.deps.TeamCache != nil {
			if err := uc.deps.TeamCache.Del(postCtx, cache.KeyTeam(teamID.String())); err != nil && uc.deps.Logger != nil {
				uc.deps.Logger.WithError(err).Warn("UserUseCase - BanUser - TeamCache.Del")
			}
		}

		if uc.deps.ChallengeListCache != nil {
			uc.deps.ChallengeListCache.InvalidateAll(postCtx)
		}
	}

	if shouldNotifyCaptain && uc.deps.PersonalNotificationSender != nil {
		_, err := uc.deps.PersonalNotificationSender.CreatePersonal(ctx, captainIDToNotify, "Team below minimum size", "A member of your team was banned. Your team is now below the minimum size required to submit. Please add members or contact an administrator.", domain.NotificationWarning)
		if err != nil && uc.deps.Logger != nil {
			uc.deps.Logger.WithError(err).Warn("UserUseCase - BanUser - CreatePersonal notification failed")
		}
	}

	return nil
}

// UnbanUser does not restore team_id for non-solo teams: the user was removed from
// the team on ban and must re-join. For solo: team is unhidden and solves restored.
// For non-solo: solves are restored so the team regains points; user must re-join manually.
func (uc *UserUseCase) UnbanUser(ctx context.Context, userID, actorID uuid.UUID) error {
	if userID == actorID {
		return httperr.ErrAccessDenied
	}

	var scoreboardInvalidateTeamIDs []uuid.UUID

	if err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		if err := uc.deps.UserRepo.Lock(ctx, userID); err != nil {
			return fmt.Errorf("UserUseCase - UnbanUser - UserRepo.Lock: %w", err)
		}

		u, err := uc.deps.UserRepo.GetByID(ctx, userID)
		if err != nil {
			return fmt.Errorf("UserUseCase - UnbanUser - UserRepo.GetByID: %w", err)
		}

		if !u.IsBanned {
			return nil
		}

		if err := uc.deps.UserRepo.Unban(ctx, userID); err != nil {
			return fmt.Errorf("UserUseCase - UnbanUser - UserRepo.Unban: %w", err)
		}

		if err := uc.deps.UserRepo.SetWasInBannedTeamByIDs(ctx, []uuid.UUID{userID}, false); err != nil {
			return fmt.Errorf("UserUseCase - UnbanUser - UserRepo.SetWasInBannedTeamByIDs: %w", err)
		}

		if uc.deps.SubmissionRepo != nil {
			if err := uc.deps.SubmissionRepo.RestoreByBannedUserID(ctx, userID); err != nil {
				return fmt.Errorf("UserUseCase - UnbanUser - SubmissionRepo.RestoreByBannedUserID: %w", err)
			}
		}

		if u.TeamID != nil && uc.deps.TeamRepo != nil {
			team, err := uc.deps.TeamRepo.GetByID(ctx, *u.TeamID)
			if err == nil && team != nil && team.IsSolo && team.IsHidden {
				if err := uc.deps.TeamRepo.SetHidden(ctx, team.ID, false); err != nil {
					return fmt.Errorf("UserUseCase - UnbanUser - TeamRepo.SetHidden: %w", err)
				}

				if err := uc.unbanUserRestoreSolvesAndAdjustScores(ctx, userID); err != nil {
					return fmt.Errorf("UserUseCase - UnbanUser - unbanUserRestoreSolvesAndAdjustScores: %w", err)
				}

				if uc.deps.AwardRepo != nil {
					if err := uc.deps.AwardRepo.RestoreByBannedTeamID(ctx, team.ID); err != nil {
						return fmt.Errorf("UserUseCase - UnbanUser - AwardRepo.RestoreByBannedTeamID: %w", err)
					}
				}

				if uc.deps.HintRepo != nil {
					if err := uc.deps.HintRepo.RestoreUnlocksByBannedTeamID(ctx, team.ID); err != nil {
						return fmt.Errorf("UserUseCase - UnbanUser - HintRepo.RestoreUnlocksByBannedTeamID: %w", err)
					}
				}

				scoreboardInvalidateTeamIDs = append(scoreboardInvalidateTeamIDs, team.ID)

				return nil
			}
		}

		if u.TeamID == nil && uc.deps.SolveRepo != nil {
			if err := uc.unbanUserRestoreSolvesAndAdjustScores(ctx, userID); err != nil {
				return fmt.Errorf("UserUseCase - UnbanUser - unbanUserRestoreSolvesAndAdjustScores: %w", err)
			}

			solves, err := uc.deps.SolveRepo.GetByUserIDWithDetails(ctx, userID)
			if err != nil {
				return fmt.Errorf("UserUseCase - UnbanUser - SolveRepo.GetByUserIDWithDetails: %w", err)
			}

			seen := make(map[uuid.UUID]struct{})

			for _, s := range solves {
				if _, ok := seen[s.TeamID]; ok {
					continue
				}

				seen[s.TeamID] = struct{}{}
				scoreboardInvalidateTeamIDs = append(scoreboardInvalidateTeamIDs, s.TeamID)
			}
		}

		return nil
	}); err != nil {
		return fmt.Errorf("UserUseCase - UnbanUser - TM.Run: %w", err)
	}

	postCtx := context.WithoutCancel(ctx)

	if uc.deps.UserCache != nil {
		uc.deps.UserCache.InvalidateUser(postCtx, userID)
	}

	for _, teamID := range scoreboardInvalidateTeamIDs {
		if uc.deps.ScoreboardCache != nil {
			uc.deps.ScoreboardCache.InvalidateForTeam(postCtx, teamID)
		}

		if uc.deps.TeamCache != nil {
			if err := uc.deps.TeamCache.Del(postCtx, cache.KeyTeam(teamID.String())); err != nil && uc.deps.Logger != nil {
				uc.deps.Logger.WithError(err).Warn("UserUseCase - UnbanUser - TeamCache.Del")
			}
		}

		if uc.deps.ChallengeListCache != nil {
			uc.deps.ChallengeListCache.InvalidateAll(postCtx)
		}
	}

	return nil
}

// banUserRemoveSolvesAndAdjustScores removes only the solves of the given user
// for the given team, then decrements challenge solve counts and recalculates
// dynamic scoring for affected challenges. Other team members' solves are unchanged.
func (uc *UserUseCase) banUserRemoveSolvesAndAdjustScores(ctx context.Context, teamID, userID uuid.UUID) error {
	if uc.deps.SolveRepo == nil || uc.deps.ChallengeRepo == nil {
		return nil
	}

	solves, err := uc.deps.SolveRepo.GetByTeamIDWithDetails(ctx, teamID)
	if err != nil {
		return fmt.Errorf("SolveRepo.GetByTeamIDWithDetails: %w", err)
	}

	var challengeIDs []uuid.UUID

	seen := make(map[uuid.UUID]struct{})

	for _, s := range solves {
		if s.UserID != userID {
			continue
		}

		if _, ok := seen[s.ChallengeID]; ok {
			continue
		}

		seen[s.ChallengeID] = struct{}{}
		challengeIDs = append(challengeIDs, s.ChallengeID)
	}

	if len(challengeIDs) > 0 {
		if err := uc.deps.SolveRepo.SoftBanByTeamIDAndUserID(ctx, teamID, userID); err != nil {
			return fmt.Errorf("SolveRepo.SoftBanByTeamIDAndUserID: %w", err)
		}
	}

	if len(challengeIDs) == 0 {
		return nil
	}

	if err := uc.deps.ChallengeRepo.BatchDecrementSolveCount(ctx, challengeIDs); err != nil {
		return fmt.Errorf("ChallengeRepo.BatchDecrementSolveCount: %w", err)
	}

	challengesMap, err := uc.deps.ChallengeRepo.GetByIDs(ctx, challengeIDs)
	if err != nil {
		return fmt.Errorf("ChallengeRepo.GetByIDs: %w", err)
	}

	ids, points := scoring.RecalculatePoints(challengesMap)
	if len(ids) > 0 {
		if err := uc.deps.ChallengeRepo.BatchUpdatePoints(ctx, ids, points); err != nil {
			return fmt.Errorf("ChallengeRepo.BatchUpdatePoints: %w", err)
		}
	}

	var dynamicIDs []uuid.UUID

	for _, id := range challengeIDs {
		if c := challengesMap[id]; c != nil && c.InitialValue > 0 && c.Decay > 0 {
			dynamicIDs = append(dynamicIDs, id)
		}
	}

	if len(dynamicIDs) > 0 {
		rows, err := uc.deps.SolveRepo.GetSolvesForPointsRecalc(ctx, dynamicIDs)
		if err != nil {
			return fmt.Errorf("SolveRepo.GetSolvesForPointsRecalc: %w", err)
		}

		recalcRows := make([]*scoring.SolveRowForPointsRecalc, 0, len(rows))
		for _, r := range rows {
			recalcRows = append(recalcRows, &scoring.SolveRowForPointsRecalc{
				ID: r.ID, ChallengeID: r.ChallengeID,
				InitialValue: r.InitialValue, MinValue: r.MinValue, Decay: r.Decay,
			})
		}

		solveIDs, newPoints := scoring.RecalculatePointsAtSolveRows(recalcRows)
		if len(solveIDs) > 0 {
			if err := uc.deps.SolveRepo.BatchUpdateSolvePoints(ctx, solveIDs, newPoints); err != nil {
				return fmt.Errorf("SolveRepo.BatchUpdateSolvePoints: %w", err)
			}
		}
	}

	return nil
}

func (uc *UserUseCase) unbanUserRestoreSolvesAndAdjustScores(ctx context.Context, userID uuid.UUID) error {
	if uc.deps.SolveRepo == nil || uc.deps.ChallengeRepo == nil {
		return nil
	}

	if err := uc.deps.SolveRepo.RestoreByBannedUserID(ctx, userID); err != nil {
		return fmt.Errorf("SolveRepo.RestoreByBannedUserID: %w", err)
	}

	solves, err := uc.deps.SolveRepo.GetByUserIDWithDetails(ctx, userID)
	if err != nil {
		return fmt.Errorf("SolveRepo.GetByUserIDWithDetails: %w", err)
	}

	seen := make(map[uuid.UUID]struct{})

	var challengeIDs []uuid.UUID

	for _, s := range solves {
		if _, ok := seen[s.ChallengeID]; ok {
			continue
		}

		seen[s.ChallengeID] = struct{}{}
		challengeIDs = append(challengeIDs, s.ChallengeID)
	}

	if len(challengeIDs) == 0 {
		return nil
	}

	if err := uc.deps.ChallengeRepo.BatchIncrementSolveCount(ctx, challengeIDs); err != nil {
		return fmt.Errorf("ChallengeRepo.BatchIncrementSolveCount: %w", err)
	}

	challengesMap, err := uc.deps.ChallengeRepo.GetByIDs(ctx, challengeIDs)
	if err != nil {
		return fmt.Errorf("ChallengeRepo.GetByIDs: %w", err)
	}

	ids, points := scoring.RecalculatePoints(challengesMap)
	if len(ids) > 0 {
		if err := uc.deps.ChallengeRepo.BatchUpdatePoints(ctx, ids, points); err != nil {
			return fmt.Errorf("ChallengeRepo.BatchUpdatePoints: %w", err)
		}
	}

	var dynamicIDs []uuid.UUID

	for _, id := range challengeIDs {
		if c := challengesMap[id]; c != nil && c.InitialValue > 0 && c.Decay > 0 {
			dynamicIDs = append(dynamicIDs, id)
		}
	}

	if len(dynamicIDs) > 0 {
		rows, err := uc.deps.SolveRepo.GetSolvesForPointsRecalc(ctx, dynamicIDs)
		if err != nil {
			return fmt.Errorf("SolveRepo.GetSolvesForPointsRecalc: %w", err)
		}

		recalcRows := make([]*scoring.SolveRowForPointsRecalc, 0, len(rows))
		for _, r := range rows {
			recalcRows = append(recalcRows, &scoring.SolveRowForPointsRecalc{
				ID: r.ID, ChallengeID: r.ChallengeID,
				InitialValue: r.InitialValue, MinValue: r.MinValue, Decay: r.Decay,
			})
		}

		solveIDs, newPoints := scoring.RecalculatePointsAtSolveRows(recalcRows)
		if len(solveIDs) > 0 {
			if err := uc.deps.SolveRepo.BatchUpdateSolvePoints(ctx, solveIDs, newPoints); err != nil {
				return fmt.Errorf("SolveRepo.BatchUpdateSolvePoints: %w", err)
			}
		}
	}

	return nil
}
