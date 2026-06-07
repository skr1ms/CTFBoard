package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func (f *TestFixture) CreateChallenge(t *testing.T, suffix string, points int) *domain.Challenge {
	t.Helper()

	unique := suffix + "_" + uuid.NewString()[:8]
	ctx := context.Background()
	challenge := &domain.Challenge{
		Title:        "Challenge " + unique,
		Description:  "Description " + unique,
		Category:     "Web",
		Points:       points,
		FlagHash:     "hash_" + unique,
		State:        domain.ChallengeStateVisible,
		InitialValue: points,
		MinValue:     points,
		Decay:        0,
	}
	challenge.ID = uuid.New()
	err := f.TM.Run(ctx, func(ctx context.Context) error {
		return f.ChallengeRepo.Create(ctx, challenge)
	})
	require.NoError(t, err)

	return challenge
}

func (f *TestFixture) CreateDynamicChallenge(t *testing.T, suffix string, initial, minValue, decay int) *domain.Challenge {
	t.Helper()

	unique := suffix + "_" + uuid.NewString()[:8]
	ctx := context.Background()
	challenge := &domain.Challenge{
		Title:        "Dynamic " + unique,
		Description:  "Description " + unique,
		Category:     "Pwn",
		Points:       initial,
		FlagHash:     "hash_" + unique,
		State:        domain.ChallengeStateVisible,
		InitialValue: initial,
		MinValue:     minValue,
		Decay:        decay,
	}
	challenge.ID = uuid.New()
	err := f.TM.Run(ctx, func(ctx context.Context) error {
		return f.ChallengeRepo.Create(ctx, challenge)
	})
	require.NoError(t, err)

	return challenge
}

func (f *TestFixture) CreateHint(t *testing.T, challengeID uuid.UUID, cost, order int) *domain.Hint {
	t.Helper()

	ctx := context.Background()
	hint := &domain.Hint{
		ChallengeID: challengeID,
		Content:     "Hint content",
		Cost:        cost,
		OrderIndex:  order,
	}
	err := f.HintRepo.Create(ctx, hint)
	require.NoError(t, err)

	return hint
}

func (f *TestFixture) CreateSolve(t *testing.T, userID, teamID, challengeID uuid.UUID) *domain.Solve {
	t.Helper()

	ctx := context.Background()
	challenge, err := f.ChallengeRepo.GetByID(ctx, challengeID)
	require.NoError(t, err)

	solve := &domain.Solve{
		UserID:        userID,
		TeamID:        teamID,
		ChallengeID:   challengeID,
		PointsAtSolve: challenge.Points,
	}
	err = f.TM.Run(ctx, func(ctx context.Context) error {
		return f.SolveRepo.Create(ctx, solve)
	})
	require.NoError(t, err)

	gotSolve, err := f.SolveRepo.GetByTeamAndChallenge(ctx, teamID, challengeID)
	require.NoError(t, err)

	solve.ID = gotSolve.ID
	solve.SolvedAt = gotSolve.SolvedAt

	return solve
}

func (f *TestFixture) CreateAwardTx(t *testing.T, ctx context.Context, teamID uuid.UUID, value int, desc string) *domain.Award {
	t.Helper()

	award := &domain.Award{
		TeamID:      teamID,
		Value:       value,
		Description: desc,
	}
	err := f.AwardRepo.Create(ctx, award)
	require.NoError(t, err)

	return award
}

// CreateAward creates an award inside a transaction (production path). Use in tests.

func (f *TestFixture) CreateAward(t *testing.T, teamID uuid.UUID, value int, desc string, createdBy *uuid.UUID) *domain.Award {
	t.Helper()

	ctx := context.Background()
	award := &domain.Award{
		TeamID:      teamID,
		Value:       value,
		Description: desc,
		CreatedBy:   createdBy,
	}
	err := f.TM.Run(ctx, func(ctx context.Context) error {
		return f.AwardRepo.Create(ctx, award)
	})
	require.NoError(t, err)

	return award
}

func (f *TestFixture) ActiveCompetition(t *testing.T, name string) *domain.Competition {
	t.Helper()

	start := time.Now().UTC().Add(-1 * time.Hour)
	end := time.Now().UTC().Add(24 * time.Hour)

	return &domain.Competition{
		ID:              1,
		Name:            name,
		StartTime:       &start,
		EndTime:         &end,
		Mode:            domain.ModeTeamsOnly,
		AllowTeamSwitch: true,
		MinTeamSize:     1,
		MaxTeamSize:     10,
	}
}

func (f *TestFixture) ResetCompetition(t *testing.T) {
	t.Helper()

	require.NoError(t, seedCompetition(context.Background(), f.Pool))
}
