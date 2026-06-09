package team

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/cacheutil"
)

// lockTeamWithMembers loads team members, locks each user row in lexicographic UUID
// order, then locks the team row, and re-reads the member list. Returns ErrTeamConflict
// when the membership changes between the two snapshots (TOCTOU guard).
func (uc *TeamUseCase) lockTeamWithMembers(ctx context.Context, teamID uuid.UUID) ([]*domain.User, error) {
	members, err := uc.deps.UserRepo.GetByTeamID(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - lockTeamWithMembers - UserRepo.GetByTeamID: %w", err)
	}

	domain.SortUsersByID(members)

	for _, m := range members {
		if err := uc.deps.UserRepo.Lock(ctx, m.ID); err != nil {
			return nil, fmt.Errorf("TeamUseCase - lockTeamWithMembers - UserRepo.Lock: %w", err)
		}
	}

	if err := uc.deps.TeamRepo.Lock(ctx, teamID); err != nil {
		return nil, fmt.Errorf("TeamUseCase - lockTeamWithMembers - TeamRepo.Lock: %w", err)
	}

	membersAfter, err := uc.deps.UserRepo.GetByTeamID(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - lockTeamWithMembers - UserRepo.GetByTeamID (recheck): %w", err)
	}

	domain.SortUsersByID(membersAfter)

	if len(membersAfter) != len(members) {
		return nil, apperr.ErrTeamConflict
	}

	for i := range members {
		if members[i].ID != membersAfter[i].ID {
			return nil, apperr.ErrTeamConflict
		}
	}

	return membersAfter, nil
}

// cascadeSoftBan soft-bans all solves, submissions, awards, ratings, and hint unlocks
// for the given team. Used when banning a team to hide their data from public views.
func (uc *TeamUseCase) cascadeSoftBan(ctx context.Context, teamID uuid.UUID) error {
	if err := uc.deps.SolveRepo.SoftBanByTeamID(ctx, teamID); err != nil {
		return fmt.Errorf("TeamUseCase - cascadeSoftBan - SolveRepo.SoftBanByTeamID: %w", err)
	}

	if err := uc.deps.SubmissionRepo.SoftBanByTeamID(ctx, teamID); err != nil {
		return fmt.Errorf("TeamUseCase - cascadeSoftBan - SubmissionRepo.SoftBanByTeamID: %w", err)
	}

	if err := uc.deps.AwardRepo.SoftBanByTeamID(ctx, teamID); err != nil {
		return fmt.Errorf("TeamUseCase - cascadeSoftBan - AwardRepo.SoftBanByTeamID: %w", err)
	}

	if uc.deps.HintRepo != nil {
		if err := uc.deps.HintRepo.SoftBanUnlocksByTeamID(ctx, teamID); err != nil {
			return fmt.Errorf("TeamUseCase - cascadeSoftBan - HintRepo.SoftBanUnlocksByTeamID: %w", err)
		}
	}

	if uc.deps.RatingRepo != nil {
		if err := uc.deps.RatingRepo.SoftBanByTeamID(ctx, teamID); err != nil {
			return fmt.Errorf("TeamUseCase - cascadeSoftBan - RatingRepo.SoftBanByTeamID: %w", err)
		}
	}

	return nil
}

// cascadeRestore restores soft-banned solves, submissions, awards, ratings, and hint
// unlocks for the given team. Used when unbanning a team.
func (uc *TeamUseCase) cascadeRestore(ctx context.Context, teamID uuid.UUID) error {
	if err := uc.deps.SolveRepo.RestoreByBannedTeamID(ctx, teamID); err != nil {
		return fmt.Errorf("TeamUseCase - cascadeRestore - SolveRepo.RestoreByBannedTeamID: %w", err)
	}

	if err := uc.deps.SubmissionRepo.RestoreByBannedTeamID(ctx, teamID); err != nil {
		return fmt.Errorf("TeamUseCase - cascadeRestore - SubmissionRepo.RestoreByBannedTeamID: %w", err)
	}

	if err := uc.deps.AwardRepo.RestoreByBannedTeamID(ctx, teamID); err != nil {
		return fmt.Errorf("TeamUseCase - cascadeRestore - AwardRepo.RestoreByBannedTeamID: %w", err)
	}

	if uc.deps.HintRepo != nil {
		if err := uc.deps.HintRepo.RestoreUnlocksByBannedTeamID(ctx, teamID); err != nil {
			return fmt.Errorf("TeamUseCase - cascadeRestore - HintRepo.RestoreUnlocksByBannedTeamID: %w", err)
		}
	}

	if uc.deps.RatingRepo != nil {
		if err := uc.deps.RatingRepo.RestoreByBannedTeamID(ctx, teamID); err != nil {
			return fmt.Errorf("TeamUseCase - cascadeRestore - RatingRepo.RestoreByBannedTeamID: %w", err)
		}
	}

	return nil
}

// cascadeDelete hard-deletes all solves, submissions, awards, ratings, and hint unlocks
// for the given team. Used when deleting or disbanding a team.
func (uc *TeamUseCase) cascadeDelete(ctx context.Context, teamID uuid.UUID) error {
	if err := uc.deps.SolveRepo.DeleteByTeamID(ctx, teamID); err != nil {
		return fmt.Errorf("TeamUseCase - cascadeDelete - SolveRepo.DeleteByTeamID: %w", err)
	}

	if err := uc.deps.SubmissionRepo.DeleteByTeamID(ctx, teamID); err != nil {
		return fmt.Errorf("TeamUseCase - cascadeDelete - SubmissionRepo.DeleteByTeamID: %w", err)
	}

	if err := uc.deps.AwardRepo.DeleteByTeamID(ctx, teamID); err != nil {
		return fmt.Errorf("TeamUseCase - cascadeDelete - AwardRepo.DeleteByTeamID: %w", err)
	}

	if uc.deps.HintRepo != nil {
		if err := uc.deps.HintRepo.DeleteUnlocksByTeamID(ctx, teamID); err != nil {
			return fmt.Errorf("TeamUseCase - cascadeDelete - HintRepo.DeleteUnlocksByTeamID: %w", err)
		}
	}

	if uc.deps.RatingRepo != nil {
		if err := uc.deps.RatingRepo.DeleteByTeamID(ctx, teamID); err != nil {
			return fmt.Errorf("TeamUseCase - cascadeDelete - RatingRepo.DeleteByTeamID: %w", err)
		}
	}

	return nil
}

// resolveMaxTeamSize returns comp.MaxTeamSize when positive, otherwise fallback.
// comp may be nil (e.g. competition not yet configured), in which case fallback is returned.
func resolveMaxTeamSize(comp *domain.Competition, fallback int) int {
	if comp != nil && comp.MaxTeamSize > 0 {
		return comp.MaxTeamSize
	}

	return fallback
}

// invalidateTeamAndMembers evicts the user cache for each member, the team cache, the
// scoreboard entry, and the challenge-list cache. Used after ban/unban operations that
// affect scoring visibility, including frozen scoreboards.
func (uc *TeamUseCase) invalidateTeamAndMembers(ctx context.Context, teamID uuid.UUID, memberIDs []uuid.UUID) {
	for _, id := range memberIDs {
		cacheutil.InvalidateUser(ctx, uc.deps.UserCache, id)
	}

	cacheutil.InvalidateTeam(ctx, uc.deps.TeamCache, uc.deps.Logger, teamID)
	cacheutil.InvalidateScoreboardForTeam(ctx, uc.deps.ScoreboardCache, teamID)
	cacheutil.InvalidateChallengeList(ctx, uc.deps.ChallengeListCache)
	cacheutil.InvalidateStatistics(ctx, uc.deps.StatsCache, uc.deps.Logger, "TeamUseCase - invalidateTeamAndMembers")
}
