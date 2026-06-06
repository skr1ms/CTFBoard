package user

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-logkit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/scoring"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/cacheutil"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/computil"
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
// drops below the configured minimum size
//
//nolint:funlen // ban flow: lock, validations, DB updates, cache invalidation, notifications
func (uc *UserUseCase) BanUser(ctx context.Context, userID uuid.UUID, reason string, actorID uuid.UUID) error {
	if userID == actorID {
		return apperr.ErrAccessDenied
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
			return apperr.ErrAccessDenied
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
						Details: map[string]any{domain.TeamAuditDetailReason: "user_banned"},
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
								domain.SortUsersByID(eligible)

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

					if uc.deps.CompRepo != nil {
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
		if err := uc.deps.JWTService.RevokeAllForUser(postCtx, userID); err != nil {
			uc.deps.Logger.WithError(err).Warn("UserUseCase - BanUser - RevokeAllForUser")
		}
	}

	cacheutil.InvalidateUser(postCtx, uc.deps.UserCache, userID)

	comp := computil.Cached(postCtx, nil, uc.deps.CompRepo)
	frozen := comp != nil && comp.IsFreezeActive()

	for _, teamID := range scoreboardInvalidateTeamIDs {
		cacheutil.InvalidateWithFreezeAwareness(postCtx, uc.deps.ScoreboardCache, teamID, frozen)
		cacheutil.InvalidateTeam(postCtx, uc.deps.TeamCache, uc.deps.Logger, teamID)
		cacheutil.InvalidateChallengeList(postCtx, uc.deps.ChallengeListCache)
	}

	if shouldNotifyCaptain && uc.deps.PersonalNotificationSender != nil {
		_, err := uc.deps.PersonalNotificationSender.CreatePersonal(ctx, usecase.NotificationCreatePersonalParams{
			UserID:  captainIDToNotify,
			Title:   "Team below minimum size",
			Content: "A member of your team was banned. Your team is now below the minimum size required to submit. Please add members or contact an administrator.",
			Type:    domain.NotificationWarning,
		})
		if err != nil {
			uc.deps.Logger.WithError(err).Warn("UserUseCase - BanUser - CreatePersonal notification failed")
		}
	}

	return nil
}

// UnbanUser reverses a user ban. Inside a transaction it unsets the ban flag,
// clears the was_in_banned_team flag, and restores soft-banned submissions. For a
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

	cacheutil.InvalidateUser(postCtx, uc.deps.UserCache, userID)

	comp := computil.Cached(postCtx, nil, uc.deps.CompRepo)
	frozen := comp != nil && comp.IsFreezeActive()

	for _, teamID := range scoreboardInvalidateTeamIDs {
		cacheutil.InvalidateWithFreezeAwareness(postCtx, uc.deps.ScoreboardCache, teamID, frozen)
		cacheutil.InvalidateTeam(postCtx, uc.deps.TeamCache, uc.deps.Logger, teamID)
		cacheutil.InvalidateChallengeList(postCtx, uc.deps.ChallengeListCache)
	}

	return nil
}

// banUserRemoveSolvesAndAdjustScores removes all solves belonging to userID
// within teamID and then heals the scoring state for every affected challenge
// It first collects the distinct challenge IDs touched by the user, soft-bans
// those solve rows, recalculates the per-challenge solve counts, recomputes
// static point values via scoring.RecalculatePoints, and finally recalculates
// the per-solve point snapshots for dynamic (decay-based) challenges using
// scoring.RecalculatePointsAtSolveRows. Other team members' solves are left
// untouched.
func (uc *UserUseCase) banUserRemoveSolvesAndAdjustScores(ctx context.Context, teamID, userID uuid.UUID) error {
	if uc.deps.SolveRepo == nil || uc.deps.ChallengeRepo == nil {
		return nil
	}

	solves, err := uc.deps.SolveRepo.GetByTeamIDWithDetails(ctx, teamID)
	if err != nil {
		return fmt.Errorf("UserUseCase - banUserRemoveSolvesAndAdjustScores - SolveRepo.GetByTeamIDWithDetails: %w", err)
	}

	var rawIDs []uuid.UUID

	for _, s := range solves {
		if s.UserID != userID {
			continue
		}

		rawIDs = append(rawIDs, s.ChallengeID)
	}

	challengeIDs := domain.UniqueUUIDs(rawIDs)

	if len(challengeIDs) > 0 {
		if err := uc.deps.SolveRepo.SoftBanByTeamIDAndUserID(ctx, teamID, userID); err != nil {
			return fmt.Errorf("UserUseCase - banUserRemoveSolvesAndAdjustScores - SolveRepo.SoftBanByTeamIDAndUserID: %w", err)
		}
	}

	return uc.adjustDynamicScores(ctx, challengeIDs)
}

// unbanUserRestoreSolvesAndAdjustScores restores soft-banned solve rows for a user,
// then recalculates solve counts, static point values, and per-solve decay snapshots
// for all affected challenges. The mirror inverse of banUserRemoveSolvesAndAdjustScores.
func (uc *UserUseCase) unbanUserRestoreSolvesAndAdjustScores(ctx context.Context, userID uuid.UUID) error {
	if uc.deps.SolveRepo == nil || uc.deps.ChallengeRepo == nil {
		return nil
	}

	if err := uc.deps.SolveRepo.RestoreByBannedUserID(ctx, userID); err != nil {
		return fmt.Errorf("UserUseCase - unbanUserRestoreSolvesAndAdjustScores - SolveRepo.RestoreByBannedUserID: %w", err)
	}

	solves, err := uc.deps.SolveRepo.GetByUserIDWithDetails(ctx, userID)
	if err != nil {
		return fmt.Errorf("UserUseCase - unbanUserRestoreSolvesAndAdjustScores - SolveRepo.GetByUserIDWithDetails: %w", err)
	}

	rawIDs := make([]uuid.UUID, 0, len(solves))

	for _, s := range solves {
		rawIDs = append(rawIDs, s.ChallengeID)
	}

	return uc.adjustDynamicScores(ctx, domain.UniqueUUIDs(rawIDs))
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
