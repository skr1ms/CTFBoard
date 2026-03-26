package integration_test

import (
	"context"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/competition"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/team"
)

func TestTeamUseCase_Create_Concurrent_DuplicateName(t *testing.T) {
	t.Parallel()
	pool := SetupTestPool(t)
	f := NewTestFixture(pool.Pool)
	ctx := context.Background()

	_, err := f.Pool.Exec(ctx, "UPDATE competition SET allow_team_switch = true, mode = 'flexible' WHERE id = 1")
	require.NoError(t, err)

	guard := competition.NewGuard(f.CompetitionRepo)
	uc := team.NewTeamUseCase(team.TeamDeps{
		TeamRepo:           f.TeamRepo,
		UserRepo:           f.UserRepo,
		SolveRepo:          f.SolveRepo,
		SubmissionRepo:     f.SubmissionRepo,
		AwardRepo:          f.AwardRepo,
		CompRepo:           f.CompetitionRepo,
		SettingsGetter:     f.SettingsRepo,
		ChallengeRepo:      f.ChallengeRepo,
		TM:                 f.TM,
		Guard:              guard,
		HintRepo:           f.HintRepo,
		DefaultMaxTeamSize: 10,
	})

	u1 := f.CreateUser(t, "racer_1")
	u2 := f.CreateUser(t, "racer_2")
	teamName := "RaceTeam"

	var wg sync.WaitGroup
	wg.Add(2)

	errCh := make(chan error, 2)
	teamCh := make(chan *domain.Team, 2)

	go func() {
		defer wg.Done()

		team, err := uc.Create(ctx, teamName, u1.ID, false, false)
		if err != nil {
			errCh <- err
		} else {
			teamCh <- team
		}
	}()

	go func() {
		defer wg.Done()

		team, err := uc.Create(ctx, teamName, u2.ID, false, false)
		if err != nil {
			errCh <- err
		} else {
			teamCh <- team
		}
	}()

	wg.Wait()
	close(errCh)
	close(teamCh)

	var teams []*domain.Team

	for tm := range teamCh {
		teams = append(teams, tm)
	}

	var errors []error

	for err := range errCh {
		errors = append(errors, err)
	}

	assert.Len(t, teams, 1, "Exactly one team should be created")
	assert.Len(t, errors, 1, "Exactly one creation should fail")

	if len(teams) > 0 {
		assert.Equal(t, teamName, teams[0].Name)
	}
}

func TestTeamUseCase_Join_Concurrent_MaxCapacity(t *testing.T) {
	t.Parallel()
	pool := SetupTestPool(t)
	f := NewTestFixture(pool.Pool)
	ctx := context.Background()

	_, err := f.Pool.Exec(ctx, "UPDATE competition SET max_team_size = 2, allow_team_switch = true, mode = 'flexible' WHERE id = 1")
	require.NoError(t, err)

	guard := competition.NewGuard(f.CompetitionRepo)
	uc := team.NewTeamUseCase(team.TeamDeps{
		TeamRepo:           f.TeamRepo,
		UserRepo:           f.UserRepo,
		SolveRepo:          f.SolveRepo,
		SubmissionRepo:     f.SubmissionRepo,
		AwardRepo:          f.AwardRepo,
		CompRepo:           f.CompetitionRepo,
		SettingsGetter:     f.SettingsRepo,
		ChallengeRepo:      f.ChallengeRepo,
		TM:                 f.TM,
		Guard:              guard,
		HintRepo:           f.HintRepo,
		DefaultMaxTeamSize: 10,
	})

	captain := f.CreateUser(t, "captain")
	team, err := uc.Create(ctx, "MaxCapTeam", captain.ID, false, false)
	require.NoError(t, err)

	u1 := f.CreateUser(t, "joiner_1")
	u2 := f.CreateUser(t, "joiner_2")

	var wg sync.WaitGroup
	wg.Add(2)

	errCh := make(chan error, 2)
	successCh := make(chan string, 2)

	opts := func(uID uuid.UUID, name string) {
		defer wg.Done()

		_, err := uc.Join(ctx, team.InviteToken, uID, false)
		if err != nil {
			errCh <- err
		} else {
			successCh <- name
		}
	}

	go opts(u1.ID, "joiner_1")
	go opts(u2.ID, "joiner_2")

	wg.Wait()
	close(errCh)
	close(successCh)

	var succeeded []string

	for s := range successCh {
		succeeded = append(succeeded, s)
	}

	var failures []error

	for err := range errCh {
		failures = append(failures, err)
	}

	assert.Len(t, succeeded, 1, "Only one user should be able to join")
	assert.Len(t, failures, 1, "One user should fail to join")

	members, err := uc.GetTeamMembers(ctx, team.ID)
	require.NoError(t, err)
	assert.Len(t, members, 2, "Team should have exactly 2 members")
}
