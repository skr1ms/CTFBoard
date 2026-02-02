package integration_test

import (
	"context"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/usecase/team"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTeamUseCase_Create_Concurrent_DuplicateName(t *testing.T) {
	t.Helper()
	pool := SetupTestPool(t)
	f := NewTestFixture(pool.Pool)
	ctx := context.Background()

	uc := team.NewTeamUseCase(f.TeamRepo, f.UserRepo, f.CompetitionRepo, f.TxRepo, nil)

	u1 := f.CreateUser(t, "racer_1")
	u2 := f.CreateUser(t, "racer_2")
	teamName := "RaceTeam"

	var wg sync.WaitGroup
	wg.Add(2)

	errCh := make(chan error, 2)
	teamCh := make(chan *entity.Team, 2)

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

	var teams []*entity.Team
	for tm := range teamCh {
		teams = append(teams, tm)
	}

	var errors []error
	for err := range errCh {
		errors = append(errors, err)
	}

	assert.Equal(t, 1, len(teams), "Exactly one team should be created")
	assert.Equal(t, 1, len(errors), "Exactly one creation should fail")

	if len(teams) > 0 {
		assert.Equal(t, teamName, teams[0].Name)
	}
}

func TestTeamUseCase_Join_Concurrent_MaxCapacity(t *testing.T) {
	t.Helper()
	pool := SetupTestPool(t)
	f := NewTestFixture(pool.Pool)
	ctx := context.Background()

	uc := team.NewTeamUseCaseWithSize(f.TeamRepo, f.UserRepo, f.CompetitionRepo, f.TxRepo, nil, 2)

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

	assert.Equal(t, 1, len(succeeded), "Only one user should be able to join")
	assert.Equal(t, 1, len(failures), "One user should fail to join")

	members, err := uc.GetTeamMembers(ctx, team.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, len(members), "Team should have exactly 2 members")
}
