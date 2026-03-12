package helper

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/competition/mocks"
)

func TestNormalizePerPage_Default(t *testing.T) {
	t.Parallel()
	repo := mocks.NewMockSettingsRepository(t)
	repo.EXPECT().Get(context.Background()).Return(&entity.Settings{DefaultPerPage: 20, MaxPerPage: 100}, nil)

	result, err := NormalizePerPage(context.Background(), repo, nil)
	require.NoError(t, err)
	assert.Equal(t, 20, result)
}

func TestNormalizePerPage_RequestedWithinLimit(t *testing.T) {
	t.Parallel()
	repo := mocks.NewMockSettingsRepository(t)
	repo.EXPECT().Get(context.Background()).Return(&entity.Settings{DefaultPerPage: 20, MaxPerPage: 100}, nil)

	perPage := 50
	result, err := NormalizePerPage(context.Background(), repo, &perPage)
	require.NoError(t, err)
	assert.Equal(t, 50, result)
}

func TestNormalizePerPage_ExceedsMax(t *testing.T) {
	t.Parallel()
	repo := mocks.NewMockSettingsRepository(t)
	repo.EXPECT().Get(context.Background()).Return(&entity.Settings{DefaultPerPage: 20, MaxPerPage: 100}, nil)

	perPage := 999
	result, err := NormalizePerPage(context.Background(), repo, &perPage)
	require.NoError(t, err)
	assert.Equal(t, 100, result)
}

func TestNormalizePerPage_ZeroRequested(t *testing.T) {
	t.Parallel()
	repo := mocks.NewMockSettingsRepository(t)
	repo.EXPECT().Get(context.Background()).Return(&entity.Settings{DefaultPerPage: 20, MaxPerPage: 100}, nil)

	perPage := 0
	result, err := NormalizePerPage(context.Background(), repo, &perPage)
	require.NoError(t, err)
	assert.Equal(t, 20, result)
}

func TestNormalizePerPage_NegativeRequested(t *testing.T) {
	t.Parallel()
	repo := mocks.NewMockSettingsRepository(t)
	repo.EXPECT().Get(context.Background()).Return(&entity.Settings{DefaultPerPage: 20, MaxPerPage: 100}, nil)

	perPage := -1
	result, err := NormalizePerPage(context.Background(), repo, &perPage)
	require.NoError(t, err)
	assert.Equal(t, 20, result)
}

func TestNormalizePerPage_FallbackDefaults(t *testing.T) {
	t.Parallel()
	repo := mocks.NewMockSettingsRepository(t)
	repo.EXPECT().Get(context.Background()).Return(&entity.Settings{DefaultPerPage: 0, MaxPerPage: 0}, nil)

	result, err := NormalizePerPage(context.Background(), repo, nil)
	require.NoError(t, err)
	assert.Equal(t, 20, result)
}
