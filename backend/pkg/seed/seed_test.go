package seed

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/repo"
	"github.com/skr1ms/CTFBoard/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockUserRepo struct {
	getByEmailUser *entity.User
	getByEmailErr  error
	createErr      error
	created        *entity.User
}

func (m *mockUserRepo) GetByEmail(_ context.Context, _ string) (*entity.User, error) {
	return m.getByEmailUser, m.getByEmailErr
}

func (m *mockUserRepo) Create(_ context.Context, u *entity.User) error {
	m.created = u
	return m.createErr
}

func (m *mockUserRepo) GetByID(context.Context, uuid.UUID) (*entity.User, error)    { return nil, nil }
func (m *mockUserRepo) Update(context.Context, *entity.User) error                  { return nil }
func (m *mockUserRepo) GetByUsername(context.Context, string) (*entity.User, error) { return nil, nil }
func (m *mockUserRepo) GetByTeamID(context.Context, uuid.UUID) ([]*entity.User, error) {
	return nil, nil
}
func (m *mockUserRepo) GetAll(context.Context) ([]*entity.User, error)            { return nil, nil }
func (m *mockUserRepo) UpdateTeamID(context.Context, uuid.UUID, *uuid.UUID) error { return nil }
func (m *mockUserRepo) SetVerified(context.Context, uuid.UUID) error              { return nil }
func (m *mockUserRepo) UpdatePassword(context.Context, uuid.UUID, string) error   { return nil }

var _ repo.UserRepository = (*mockUserRepo)(nil)

func TestCreateDefaultAdmin_Success(t *testing.T) {
	repo := &mockUserRepo{getByEmailErr: errors.New("not found")}
	log := logger.New(&logger.Options{Level: logger.InfoLevel, Output: logger.ConsoleOutput})

	err := CreateDefaultAdmin(context.Background(), repo, "admin", "admin@test.com", "password123", log)
	require.NoError(t, err)
	require.NotNil(t, repo.created)
	assert.Equal(t, "admin", repo.created.Username)
	assert.Equal(t, "admin@test.com", repo.created.Email)
	assert.Equal(t, entity.RoleAdmin, repo.created.Role)
	assert.True(t, repo.created.IsVerified)
}

func TestCreateDefaultAdmin_AlreadyExists_Success(t *testing.T) {
	existing := &entity.User{ID: uuid.New(), Email: "admin@test.com"}
	repo := &mockUserRepo{getByEmailUser: existing}
	log := logger.New(&logger.Options{Level: logger.InfoLevel, Output: logger.ConsoleOutput})

	err := CreateDefaultAdmin(context.Background(), repo, "admin", "admin@test.com", "password123", log)
	require.NoError(t, err)
	require.Nil(t, repo.created)
}

func TestCreateDefaultAdmin_CreateError_Error(t *testing.T) {
	repo := &mockUserRepo{getByEmailErr: errors.New("not found"), createErr: errors.New("db error")}
	log := logger.New(&logger.Options{Level: logger.InfoLevel, Output: logger.ConsoleOutput})

	err := CreateDefaultAdmin(context.Background(), repo, "admin", "admin@test.com", "password123", log)
	require.Error(t, err)
	assert.Equal(t, "db error", err.Error())
}
