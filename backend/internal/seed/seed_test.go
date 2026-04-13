package seed

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/wahrwelt-kit/go-logkit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	seedMock "github.com/TakuyaYagam1/AstroCTFb/internal/seed/mock"
)

func anyAdmin() any {
	return mock.MatchedBy(func(u *domain.User) bool {
		return u.Username == "admin" &&
			u.Email == "admin@test.com" &&
			u.Role == domain.RoleAdmin &&
			u.IsVerified
	})
}

func TestCreateDefaultAdmin_Success(t *testing.T) {
	t.Parallel()

	repo := seedMock.NewMockUserRepository(t)
	repo.EXPECT().GetByEmail(context.Background(), "admin@test.com").Return(nil, apperr.ErrUserNotFound)
	repo.EXPECT().Create(context.Background(), anyAdmin()).Return(nil)

	log, logErr := logkit.New(logkit.WithLevel(logkit.InfoLevel), logkit.WithOutput(logkit.ConsoleOutput))
	require.NoError(t, logErr)

	err := CreateDefaultAdmin(context.Background(), repo, "admin", "admin@test.com", "password123", log, 0)
	require.NoError(t, err)
}

func TestCreateDefaultAdmin_AlreadyExists_Success(t *testing.T) {
	t.Parallel()

	repo := seedMock.NewMockUserRepository(t)
	repo.EXPECT().GetByEmail(context.Background(), "admin@test.com").Return(&domain.User{Email: "admin@test.com"}, nil)

	log, logErr := logkit.New(logkit.WithLevel(logkit.InfoLevel), logkit.WithOutput(logkit.ConsoleOutput))
	require.NoError(t, logErr)

	err := CreateDefaultAdmin(context.Background(), repo, "admin", "admin@test.com", "password123", log, 0)
	require.NoError(t, err)
}

func TestCreateDefaultAdmin_CreateError_Error(t *testing.T) {
	t.Parallel()

	repo := seedMock.NewMockUserRepository(t)
	repo.EXPECT().GetByEmail(context.Background(), "admin@test.com").Return(nil, apperr.ErrUserNotFound)
	repo.EXPECT().Create(context.Background(), anyAdmin()).Return(errors.New("db error"))

	log, logErr := logkit.New(logkit.WithLevel(logkit.InfoLevel), logkit.WithOutput(logkit.ConsoleOutput))
	require.NoError(t, logErr)

	err := CreateDefaultAdmin(context.Background(), repo, "admin", "admin@test.com", "password123", log, 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "db error")
}
