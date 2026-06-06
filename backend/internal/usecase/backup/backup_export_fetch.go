package backup

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

// fetchChallengesWithHints loads all challenges for backup, resolves their hint lists
// with a single batch query, and annotates each challenge with its tag IDs. Tags are
// deduplicated into a flat list and stored on the returned slice for the caller to
// attach to BackupData.Tags.
func (uc *BackupUseCase) fetchChallengesWithHints(ctx context.Context) ([]domain.ChallengeExport, error) {
	challengesWithSolved, err := uc.deps.ChallengeRepo.GetAllForBackup(ctx)
	if err != nil {
		return nil, fmt.Errorf("BackupUseCase - fetchChallengesWithHints - ChallengeRepo.GetAll: %w", err)
	}

	challengeIDs := make([]uuid.UUID, len(challengesWithSolved))
	for i, cws := range challengesWithSolved {
		challengeIDs[i] = cws.Challenge.ID
	}

	var hintsByChallenge map[uuid.UUID][]*domain.Hint

	if uc.deps.HintRepo != nil {
		var err error

		hintsByChallenge, err = uc.deps.HintRepo.GetByChallengeIDs(ctx, challengeIDs)
		if err != nil {
			return nil, fmt.Errorf("BackupUseCase - fetchChallengesWithHints - HintRepo.GetByChallengeIDs: %w", err)
		}
	}

	if hintsByChallenge == nil {
		hintsByChallenge = make(map[uuid.UUID][]*domain.Hint)
	}

	result := make([]domain.ChallengeExport, len(challengesWithSolved))
	for i, cws := range challengesWithSolved {
		hints := hintsByChallenge[cws.Challenge.ID]

		hintsCopy := make([]domain.Hint, len(hints))
		for j, h := range hints {
			hintsCopy[j] = *h
		}

		flagRegex := ""

		if cws.Challenge.FlagRegex != nil {
			flagRegex = *cws.Challenge.FlagRegex
		}

		result[i] = domain.ChallengeExport{
			Challenge:      *cws.Challenge,
			State:          cws.Challenge.State,
			FlagHash:       cws.Challenge.FlagHash,
			FlagRegex:      flagRegex,
			ConnectionInfo: cws.Challenge.ConnectionInfo,
			MaxAttempts:    cws.Challenge.MaxAttempts,
			Position:       cws.Challenge.Position,
			Hints:          hintsCopy,
		}
	}

	return result, nil
}

// fetchTeamsWithMembers loads all teams and resolves member lists with a single
// batch query (GetByTeamIDs) to avoid N+1 database round-trips.
func (uc *BackupUseCase) fetchTeamsWithMembers(ctx context.Context) ([]domain.TeamExport, error) {
	teams, err := uc.deps.TeamRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("BackupUseCase - fetchTeamsWithMembers - TeamRepo.GetAll: %w", err)
	}

	teamIDs := make([]uuid.UUID, len(teams))
	for i, t := range teams {
		teamIDs[i] = t.ID
	}

	membersByTeam, err := uc.deps.UserRepo.GetByTeamIDs(ctx, teamIDs)
	if err != nil {
		return nil, fmt.Errorf("BackupUseCase - fetchTeamsWithMembers - UserRepo.GetByTeamIDs: %w", err)
	}

	result := make([]domain.TeamExport, len(teams))
	for i, team := range teams {
		members := membersByTeam[team.ID]

		memberIDs := make([]uuid.UUID, len(members))
		for j, m := range members {
			memberIDs[j] = m.ID
		}

		result[i] = domain.TeamExport{
			Team:                 *team,
			InviteToken:          team.InviteToken,
			InviteTokenExpiresAt: team.InviteTokenExpiresAt,
			MemberIDs:            memberIDs,
		}
	}

	return result, nil
}

func (uc *BackupUseCase) fetchUsers(ctx context.Context) ([]domain.UserExport, error) {
	users, err := uc.deps.UserRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("BackupUseCase - fetchUsers - UserRepo.GetAll: %w", err)
	}

	result := make([]domain.UserExport, 0, len(users))
	for _, u := range users {
		result = append(result, domain.UserExport{
			ID:           u.ID,
			Username:     u.Username,
			Email:        u.Email,
			Role:         string(u.Role),
			TeamID:       u.TeamID,
			IsVerified:   u.IsVerified,
			VerifiedAt:   u.VerifiedAt,
			IsBanned:     u.IsBanned,
			BannedAt:     u.BannedAt,
			BannedReason: u.BannedReason,
			CreatedAt:    u.CreatedAt,
		})
	}

	return result, nil
}

func (uc *BackupUseCase) fetchAwards(ctx context.Context) ([]domain.Award, error) {
	awards, err := uc.deps.AwardRepo.GetAllForBackup(ctx)
	if err != nil {
		return nil, fmt.Errorf("BackupUseCase - fetchAwards - AwardRepo.GetAllForBackup: %w", err)
	}

	result := make([]domain.Award, len(awards))
	for i, a := range awards {
		result[i] = *a
	}

	return result, nil
}

func (uc *BackupUseCase) fetchSolves(ctx context.Context) ([]domain.Solve, error) {
	solves, err := uc.deps.SolveRepo.GetAllForBackup(ctx)
	if err != nil {
		return nil, fmt.Errorf("BackupUseCase - fetchSolves - SolveRepo.GetAllForBackup: %w", err)
	}

	result := make([]domain.Solve, len(solves))
	for i, s := range solves {
		result[i] = *s
	}

	return result, nil
}

func (uc *BackupUseCase) fetchFiles(ctx context.Context) ([]domain.File, error) {
	files, err := uc.deps.FileRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("BackupUseCase - fetchFiles - FileRepo.GetAll: %w", err)
	}

	result := make([]domain.File, len(files))
	for i, f := range files {
		result[i] = *f
	}

	return result, nil
}
