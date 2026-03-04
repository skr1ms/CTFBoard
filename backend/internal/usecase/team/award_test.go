package team

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAwardUseCase_Create(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		h := NewAwardTestHelper(t)
		h.TM().EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		})
		h.Repo().On("Create", mock.Anything, mock.MatchedBy(func(a *entity.Award) bool {
			return a.TeamID == h.TeamID() && a.Value == 100 && a.Description == "Bonus" && *a.CreatedBy == h.AdminID()
		})).Return(nil).Once()

		award, err := h.CreateUseCase().Create(ctx, h.TeamID(), 100, "Bonus", h.AdminID())

		assert.NoError(t, err)
		assert.NotNil(t, award)
		assert.Equal(t, 100, award.Value)
		assert.Equal(t, h.AdminID(), *award.CreatedBy)
	})

	t.Run("ZeroValue", func(t *testing.T) {
		t.Parallel()
		h := NewAwardTestHelper(t)
		award, err := h.CreateUseCase().Create(ctx, h.TeamID(), 0, "Zero", h.AdminID())

		assert.Error(t, err)
		assert.Nil(t, award)
		assert.Contains(t, err.Error(), "value cannot be 0")
	})

	t.Run("RepoError", func(t *testing.T) {
		t.Parallel()
		h := NewAwardTestHelper(t)
		h.TM().On("Run", mock.Anything, mock.Anything).Return(errors.New("db error")).Once()

		award, err := h.CreateUseCase().Create(ctx, h.TeamID(), 50, "Error", h.AdminID())

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
		h := NewAwardTestHelper(t)
		teamID := h.TeamID()
		expectedAwards := []*entity.Award{
			h.NewAward(teamID, 100, time.Now()),
			h.NewAward(teamID, -50, time.Now()),
		}

		h.Repo().On("GetByTeamID", ctx, teamID).Return(expectedAwards, nil).Once()

		awards, err := h.CreateUseCase().GetByTeamID(ctx, teamID)

		assert.NoError(t, err)
		assert.Equal(t, len(expectedAwards), len(awards))
		assert.Equal(t, expectedAwards[0].ID, awards[0].ID)
	})

	t.Run("RepoError", func(t *testing.T) {
		t.Parallel()
		h := NewAwardTestHelper(t)
		teamID := h.TeamID()
		h.Repo().On("GetByTeamID", ctx, teamID).Return(nil, errors.New("db error")).Once()

		awards, err := h.CreateUseCase().GetByTeamID(ctx, teamID)

		assert.Error(t, err)
		assert.Nil(t, awards)
	})
}
