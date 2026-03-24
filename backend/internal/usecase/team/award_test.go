package team

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	teamMock "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/team/mock"
)

type awardTestDeps struct {
	repo    *teamMock.MockAwardRepository
	tm      *teamMock.MockTransactionManager
	teamID  uuid.UUID
	adminID uuid.UUID
}

func newAwardTestDeps(t *testing.T) *awardTestDeps {
	t.Helper()
	return &awardTestDeps{
		repo:    teamMock.NewMockAwardRepository(t),
		tm:      teamMock.NewMockTransactionManager(t),
		teamID:  uuid.New(),
		adminID: uuid.New(),
	}
}

func (d *awardTestDeps) createUseCase() *AwardUseCase {
	return NewAwardUseCase(AwardDeps{AwardRepo: d.repo, TM: d.tm})
}

func newTestAward(teamID uuid.UUID, value int, createdAt time.Time) *domain.Award {
	return &domain.Award{
		ID: uuid.New(), TeamID: teamID, Value: value, CreatedAt: createdAt,
	}
}

func TestAwardUseCase_Create(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		d := newAwardTestDeps(t)
		d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		})
		d.repo.On("Create", mock.Anything, mock.MatchedBy(func(a *domain.Award) bool {
			return a.TeamID == d.teamID && a.Value == 100 && a.Description == "Bonus" && *a.CreatedBy == d.adminID
		})).Return(nil).Once()

		award, err := d.createUseCase().Create(ctx, d.teamID, 100, "Bonus", d.adminID)

		assert.NoError(t, err)
		assert.NotNil(t, award)
		assert.Equal(t, 100, award.Value)
		assert.Equal(t, d.adminID, *award.CreatedBy)
	})

	t.Run("ZeroValue", func(t *testing.T) {
		t.Parallel()
		d := newAwardTestDeps(t)
		award, err := d.createUseCase().Create(ctx, d.teamID, 0, "Zero", d.adminID)

		assert.Error(t, err)
		assert.Nil(t, award)
		assert.Contains(t, err.Error(), "value cannot be 0")
	})

	t.Run("RepoError", func(t *testing.T) {
		t.Parallel()
		d := newAwardTestDeps(t)
		d.tm.On("Run", mock.Anything, mock.Anything).Return(errors.New("db error")).Once()

		award, err := d.createUseCase().Create(ctx, d.teamID, 50, "Error", d.adminID)

		assert.Error(t, err)
		assert.Nil(t, award)
		assert.Contains(t, err.Error(), "db error")
	})
}

func TestAwardUseCase_GetByTeamID(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		d := newAwardTestDeps(t)
		teamID := d.teamID
		expectedAwards := []*domain.Award{
			newTestAward(teamID, 100, time.Now()),
			newTestAward(teamID, -50, time.Now()),
		}

		d.repo.On("GetByTeamID", ctx, teamID).Return(expectedAwards, nil).Once()

		awards, err := d.createUseCase().GetByTeamID(ctx, teamID)

		assert.NoError(t, err)
		assert.Equal(t, len(expectedAwards), len(awards))
		assert.Equal(t, expectedAwards[0].ID, awards[0].ID)
	})

	t.Run("RepoError", func(t *testing.T) {
		t.Parallel()
		d := newAwardTestDeps(t)
		teamID := d.teamID
		d.repo.On("GetByTeamID", ctx, teamID).Return(nil, errors.New("db error")).Once()

		awards, err := d.createUseCase().GetByTeamID(ctx, teamID)

		assert.Error(t, err)
		assert.Nil(t, awards)
	})
}
