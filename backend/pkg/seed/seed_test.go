package seed

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/logger"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/seed/mocks"
)

func anyAdmin() any {
	return mock.MatchedBy(func(u *entity.User) bool {
		return u.Username == "admin" &&
			u.Email == "admin@test.com" &&
			u.Role == entity.RoleAdmin &&
			u.IsVerified
	})
}

func TestCreateDefaultAdmin_Success(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockUserRepository(t)
	repo.EXPECT().GetByEmail(context.Background(), "admin@test.com").Return(nil, httperr.ErrUserNotFound)
	repo.EXPECT().Create(context.Background(), anyAdmin()).Return(nil)

	log := logger.New(&logger.Options{Level: logger.InfoLevel, Output: logger.ConsoleOutput})

	err := CreateDefaultAdmin(context.Background(), repo, "admin", "admin@test.com", "password123", log)
	require.NoError(t, err)
}

func TestCreateDefaultAdmin_AlreadyExists_Success(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockUserRepository(t)
	repo.EXPECT().GetByEmail(context.Background(), "admin@test.com").Return(&entity.User{Email: "admin@test.com"}, nil)

	log := logger.New(&logger.Options{Level: logger.InfoLevel, Output: logger.ConsoleOutput})

	err := CreateDefaultAdmin(context.Background(), repo, "admin", "admin@test.com", "password123", log)
	require.NoError(t, err)
}

func TestCreateDefaultAdmin_CreateError_Error(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockUserRepository(t)
	repo.EXPECT().GetByEmail(context.Background(), "admin@test.com").Return(nil, httperr.ErrUserNotFound)
	repo.EXPECT().Create(context.Background(), anyAdmin()).Return(errors.New("db error"))

	log := logger.New(&logger.Options{Level: logger.InfoLevel, Output: logger.ConsoleOutput})

	err := CreateDefaultAdmin(context.Background(), repo, "admin", "admin@test.com", "password123", log)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "db error")
}
