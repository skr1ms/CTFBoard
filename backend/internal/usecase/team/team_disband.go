package team

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/cacheutil"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/guard"
)

// DisbandTeam dissolves a team initiated by its captain. It checks competition-level
// team-switch guard, validates competition state restrictions, then runs a transaction
// that cascades deletion of solves, submissions, awards, and hint unlocks, recalculates
// dynamic scoring, and clears team membership for all members.
func (uc *TeamUseCase) DisbandTeam(ctx context.Context, captainID uuid.UUID) error {
	if _, err := uc.deps.Guard.RequireTeamSwitch(ctx); err != nil {
		return fmt.Errorf("TeamUseCase - DisbandTeam - Guard: %w", err)
	}

	var memberIDs []uuid.UUID

	err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		comp, err := uc.deps.CompRepo.GetForUpdate(ctx)
		if err != nil {
			return fmt.Errorf("TeamUseCase - DisbandTeam - CompetitionRepo.GetForUpdate: %w", err)
		}

		if err := guard.ValidateTeamSwitchState(comp); err != nil {
			return fmt.Errorf("TeamUseCase - DisbandTeam - ValidateTeamSwitchState: %w", err)
		}

		user, team, members, err := uc.disbandPrepare(ctx, captainID)
		if err != nil {
			return fmt.Errorf("TeamUseCase - DisbandTeam - disbandPrepare: %w", err)
		}

		if err := uc.disbandValidate(user, team, captainID, comp); err != nil {
			return fmt.Errorf("TeamUseCase - DisbandTeam - disbandValidate: %w", err)
		}

		ids := make([]uuid.UUID, len(members))
		for i, m := range members {
			ids[i] = m.ID
		}

		memberIDs = ids

		return uc.disbandExecute(ctx, team, members, captainID)
	})
	if err != nil {
		return fmt.Errorf("TeamUseCase - DisbandTeam - TM.Run: %w", err)
	}

	for _, id := range memberIDs {
		cacheutil.InvalidateUser(ctx, uc.deps.UserCache, id)
	}

	cacheutil.InvalidateScoreboard(ctx, uc.deps.ScoreboardCache)
	cacheutil.InvalidateChallengeList(ctx, uc.deps.ChallengeListCache)
	cacheutil.InvalidateStatistics(ctx, uc.deps.StatsCache, uc.deps.Logger, "TeamUseCase - DisbandTeam")

	return nil
}

func (uc *TeamUseCase) disbandPrepare(ctx context.Context, captainID uuid.UUID) (*domain.User, *domain.Team, []*domain.User, error) {
	if err := uc.deps.UserRepo.Lock(ctx, captainID); err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - disbandPrepare - UserRepo.Lock: %w", err)
	}

	user, err := uc.deps.UserRepo.GetByID(ctx, captainID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - disbandPrepare - UserRepo.GetByID: %w", err)
	}

	if user.TeamID == nil {
		return nil, nil, nil, apperr.ErrTeamNotFound
	}

	if err := uc.deps.TeamRepo.Lock(ctx, *user.TeamID); err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - disbandPrepare - TeamRepo.Lock: %w", err)
	}

	team, err := uc.deps.TeamRepo.GetByID(ctx, *user.TeamID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - disbandPrepare - TeamRepo.GetByID: %w", err)
	}

	members, err := uc.deps.UserRepo.GetByTeamID(ctx, team.ID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - disbandPrepare - UserRepo.GetByTeamID: %w", err)
	}

	return user, team, members, nil
}

func (uc *TeamUseCase) disbandValidate(user *domain.User, team *domain.Team, captainID uuid.UUID, comp *domain.Competition) error {
	if team.CaptainID != captainID {
		return apperr.ErrNotCaptain
	}

	if team.IsSolo && comp.Mode == domain.ModeSoloOnly {
		return apperr.ErrCannotDisbandSoloTeam
	}

	if user.IsBanned {
		return apperr.ErrUserBanned
	}

	if team.IsBanned {
		return apperr.ErrTeamBanned
	}

	return nil
}

// disbandExecute cascades a team deletion inside a transaction. It collects the
// challenge IDs affected by the team's solves so that solve counts and dynamic scoring
// can be recalculated after deletion. It then hard-deletes all solves, submissions,
// awards, and hint unlocks, calls adjustSolveCountsForChallenges, writes a disbanded
// audit log entry, clears the team_id field for all members in a single batch update,
// removes any custom field values, and finally soft-deletes the team record. Solo teams
// for the former members are not created here; callers are responsible for that if
// required by the competition mode.
func (uc *TeamUseCase) disbandExecute(ctx context.Context, team *domain.Team, members []*domain.User, captainID uuid.UUID) error {
	challengeIDs, err := uc.getChallengeIDsForTeam(ctx, team.ID)
	if err != nil {
		return fmt.Errorf("TeamUseCase - disbandExecute - getChallengeIDsForTeam: %w", err)
	}

	if err := uc.cascadeDelete(ctx, team.ID); err != nil {
		return fmt.Errorf("TeamUseCase - disbandExecute - cascadeDelete: %w", err)
	}

	if err := uc.adjustSolveCountsForChallenges(ctx, challengeIDs); err != nil {
		return fmt.Errorf("TeamUseCase - disbandExecute - adjustSolveCountsForChallenges: %w", err)
	}

	auditLog := &domain.TeamAuditLog{
		TeamID: team.ID, UserID: &captainID, Action: domain.TeamActionDeleted,
		Details: map[string]any{domain.TeamAuditDetailReason: "disbanded_by_captain"},
	}
	if err := uc.deps.TeamRepo.CreateAuditLog(ctx, auditLog); err != nil {
		return fmt.Errorf("TeamUseCase - disbandExecute - TeamRepo.CreateAuditLog: %w", err)
	}

	memberIDs := make([]uuid.UUID, len(members))
	for i, m := range members {
		memberIDs[i] = m.ID
	}

	if err := uc.deps.UserRepo.UpdateTeamIDBatch(ctx, memberIDs, nil); err != nil {
		return fmt.Errorf("TeamUseCase - disbandExecute - UserRepo.UpdateTeamIDBatch: %w", err)
	}

	if uc.deps.FieldValueRepo != nil {
		err := uc.deps.FieldValueRepo.DeleteByEntityID(ctx, team.ID)
		if err != nil {
			return fmt.Errorf("TeamUseCase - disbandExecute - FieldValueRepo.DeleteByEntityID: %w", err)
		}
	}

	if err := uc.deps.TeamRepo.Delete(ctx, team.ID); err != nil {
		return fmt.Errorf("TeamUseCase - disbandExecute - TeamRepo.Delete: %w", err)
	}

	return nil
}
