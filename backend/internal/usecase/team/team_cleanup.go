package team

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/scoring"
)

// getChallengeIDsForTeam returns the deduplicated set of challenge IDs the team has
// solved, independent of challenge visibility. Returns nil when SolveRepo is not wired.
func (uc *TeamUseCase) getChallengeIDsForTeam(ctx context.Context, teamID uuid.UUID) ([]uuid.UUID, error) {
	if uc.deps.SolveRepo == nil {
		return nil, nil
	}

	challengeIDs, err := uc.deps.SolveRepo.GetModerationAffectedChallengeIDsByTeamID(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - getChallengeIDsForTeam - SolveRepo.GetModerationAffectedChallengeIDsByTeamID: %w", err)
	}

	return domain.UniqueUUIDs(challengeIDs), nil
}

func (uc *TeamUseCase) adjustSolveCountsForChallenges(ctx context.Context, challengeIDs []uuid.UUID) error {
	if uc.deps.ChallengeRepo == nil || len(challengeIDs) == 0 {
		return nil
	}

	var (
		getSolves   func(context.Context, []uuid.UUID) ([]*scoring.SolveRowForPointsRecalc, error)
		batchUpdate func(context.Context, []uuid.UUID, []int) error
	)

	if uc.deps.SolveRepo != nil {
		getSolves = scoring.MapSolvesForRecalcFn(
			uc.deps.SolveRepo.GetSolvesForPointsRecalc,
			scoring.DefaultSolveMapper,
			"TeamUseCase - adjustSolveCountsForChallenges",
		)
		batchUpdate = uc.deps.SolveRepo.BatchUpdateSolvePoints
	}

	return scoring.AdjustDynamicScores(
		ctx, challengeIDs, uc.deps.ChallengeRepo,
		getSolves, batchUpdate,
		scoring.GetDecayFn(ctx, uc.deps.CompParamUC),
	)
}

func (uc *TeamUseCase) adjustSolveCountsForTeam(ctx context.Context, teamID uuid.UUID) error {
	challengeIDs, err := uc.getChallengeIDsForTeam(ctx, teamID)
	if err != nil {
		return err
	}

	return uc.adjustSolveCountsForChallenges(ctx, challengeIDs)
}

// orderTeamLockIDs returns the two team IDs in lexicographic string order so that
// callers always acquire advisory/row locks in a consistent sequence and avoid
// deadlocks. When newTeamID is nil the second return value is uuid.Nil.
func orderTeamLockIDs(oldTeamID uuid.UUID, newTeamID *uuid.UUID) (first, second uuid.UUID) {
	if newTeamID == nil {
		return oldTeamID, uuid.Nil
	}

	if oldTeamID.String() < newTeamID.String() {
		return oldTeamID, *newTeamID
	}

	return *newTeamID, oldTeamID
}

// handleSoloTeamCleanup removes the user's current solo or auto-created team inside
// the transaction. To prevent deadlocks when two operations touch both the old and new
// team rows, advisory locks are acquired in a deterministic order derived from the
// lexicographic comparison of both team UUIDs (orderTeamLockIDs). Once locked, the
// function verifies the team is still eligible for cleanup (single-member solo/auto
// team). If confirmReset is false it returns ErrConfirmationRequired without making any
// changes. Otherwise it deletes all solves, submissions, awards, and hint unlocks for
// the old team, calls adjustSolveCountsForChallenges to keep scoring consistent, detaches
// the user, writes an audit log, and hard-deletes the old team record.
func (uc *TeamUseCase) handleSoloTeamCleanup(ctx context.Context, user *domain.User, actorID uuid.UUID, confirmReset bool, newTeamID *uuid.UUID) error {
	if user.TeamID == nil {
		return nil
	}

	firstID, secondID := orderTeamLockIDs(*user.TeamID, newTeamID)
	if err := uc.deps.TeamRepo.Lock(ctx, firstID); err != nil {
		return fmt.Errorf("TeamUseCase - handleSoloTeamCleanup - TeamRepo.Lock: %w", err)
	}
	if secondID != uuid.Nil {
		err := uc.deps.TeamRepo.Lock(ctx, secondID)
		if err != nil {
			return fmt.Errorf("TeamUseCase - handleSoloTeamCleanup - TeamRepo.Lock(second): %w", err)
		}
	}

	oldTeam, err := uc.deps.TeamRepo.GetByID(ctx, *user.TeamID)
	if err != nil {
		return fmt.Errorf("TeamUseCase - handleSoloTeamCleanup - TeamRepo.GetByID: %w", err)
	}

	members, err := uc.deps.UserRepo.GetByTeamID(ctx, *user.TeamID)
	if err != nil {
		return fmt.Errorf("TeamUseCase - handleSoloTeamCleanup - UserRepo.GetByTeamID: %w", err)
	}

	if !uc.shouldCleanupSoloTeam(user, members, oldTeam) {
		if oldTeam.IsBanned {
			return apperr.ErrTeamBanned
		}

		return apperr.ErrUserAlreadyInTeam
	}

	if oldTeam.IsBanned {
		return apperr.ErrTeamBanned
	}

	if !confirmReset {
		return apperr.ErrConfirmationRequired
	}

	oldTeamID := *user.TeamID

	challengeIDs, err := uc.getChallengeIDsForTeam(ctx, oldTeamID)
	if err != nil {
		return fmt.Errorf("TeamUseCase - handleSoloTeamCleanup - getChallengeIDsForTeam: %w", err)
	}

	if err := uc.cascadeDelete(ctx, oldTeamID); err != nil {
		return fmt.Errorf("TeamUseCase - handleSoloTeamCleanup - cascadeDelete: %w", err)
	}

	if err := uc.adjustSolveCountsForChallenges(ctx, challengeIDs); err != nil {
		return fmt.Errorf("TeamUseCase - handleSoloTeamCleanup - adjustSolveCountsForChallenges: %w", err)
	}

	if err := uc.deps.UserRepo.UpdateTeamID(ctx, actorID, nil); err != nil {
		return fmt.Errorf("TeamUseCase - handleSoloTeamCleanup - UserRepo.UpdateTeamID: %w", err)
	}

	auditLog := &domain.TeamAuditLog{
		TeamID: oldTeamID,
		UserID: &actorID,
		Action: domain.TeamActionDeleted,
		Details: map[string]any{
			domain.TeamAuditDetailReason: "solo_team_cleanup",
		},
	}
	if err := uc.deps.TeamRepo.CreateAuditLog(ctx, auditLog); err != nil {
		return fmt.Errorf("TeamUseCase - handleSoloTeamCleanup - TeamRepo.CreateAuditLog: %w", err)
	}

	if err := uc.deps.TeamRepo.Delete(ctx, oldTeamID); err != nil {
		return fmt.Errorf("TeamUseCase - handleSoloTeamCleanup - TeamRepo.Delete: %w", err)
	}

	return nil
}

func (uc *TeamUseCase) shouldCleanupSoloTeam(user *domain.User, members []*domain.User, oldTeam *domain.Team) bool {
	return len(members) == 1 && members[0].ID == user.ID && (oldTeam.IsSolo || oldTeam.IsAutoCreated)
}
