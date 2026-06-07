package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func (f *TestFixture) CreateUser(t *testing.T, suffix string) *domain.User {
	t.Helper()
	// Username and email must fit varchar(50): "user_" = 5, "@x.com" = 6, so unique at most 39
	unique := suffix + "_" + uuid.NewString()[:8]
	if len(unique) > 39 {
		unique = unique[:39]
	}

	ctx := context.Background()
	user := &domain.User{
		Username:     "user_" + unique,
		Email:        "user_" + unique + "@x.com",
		PasswordHash: "hash123",
	}
	err := f.UserRepo.Create(ctx, user)
	require.NoError(t, err)

	gotUser, err := f.UserRepo.GetByEmail(ctx, user.Email)
	require.NoError(t, err)

	user.ID = gotUser.ID

	return user
}

func (f *TestFixture) CreateTeam(t *testing.T, suffix string, captainID uuid.UUID) *domain.Team {
	t.Helper()

	unique := suffix + "_" + uuid.NewString()[:8]
	ctx := context.Background()
	team := &domain.Team{
		Name:        "team_" + unique,
		InviteToken: uuid.New(),
		CaptainID:   captainID,
	}
	err := f.TM.Run(ctx, func(ctx context.Context) error {
		return f.TeamRepo.Create(ctx, team)
	})
	require.NoError(t, err)

	return team
}

func (f *TestFixture) CreateUserWithTeam(t *testing.T, suffix string) (*domain.User, *domain.Team) {
	t.Helper()
	user := f.CreateUser(t, suffix)
	team := f.CreateTeam(t, suffix, user.ID)
	err := f.UserRepo.UpdateTeamID(context.Background(), user.ID, &team.ID)
	require.NoError(t, err)

	user.TeamID = &team.ID

	return user, team
}

func (f *TestFixture) AddUserToTeam(t *testing.T, userID, teamID uuid.UUID) {
	t.Helper()

	ctx := context.Background()
	_, err := f.Pool.Exec(ctx, "UPDATE users SET team_id = $1 WHERE id = $2", teamID, userID)
	require.NoError(t, err)
}

func (f *TestFixture) BackdateTeamDeletedAt(t *testing.T, teamID uuid.UUID, deletedAt time.Time) {
	t.Helper()

	ctx := context.Background()
	_, err := f.Pool.Exec(ctx, "UPDATE teams SET deleted_at = $1 WHERE id = $2", deletedAt, teamID)
	require.NoError(t, err)
}
