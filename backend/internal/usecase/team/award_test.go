package team

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/repo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAwardUseCase_Create(t *testing.T) {
	h := NewAwardTestHelper(t)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		h.TxRepo().On("RunTransaction", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
			ctx, ok := args.Get(0).(context.Context)
			if !ok {
				return
			}
			fn, ok := args.Get(1).(func(context.Context, repo.Transaction) error)
			if !ok {
				return
			}
			assert.NoError(t, fn(ctx, nil))
		}).Return(nil).Once()
		h.TxRepo().On("CreateAwardTx", mock.Anything, mock.Anything, mock.MatchedBy(func(a *entity.Award) bool {
			return a.TeamID == h.TeamID() && a.Value == 100 && a.Description == "Bonus" && *a.CreatedBy == h.AdminID()
		})).Return(nil).Once()

		award, err := h.CreateUseCase().Create(ctx, h.TeamID(), 100, "Bonus", h.AdminID())

		assert.NoError(t, err)
		assert.NotNil(t, award)
		assert.Equal(t, 100, award.Value)
		assert.Equal(t, h.AdminID(), *award.CreatedBy)
	})

	t.Run("ZeroValue", func(t *testing.T) {
		award, err := h.CreateUseCase().Create(ctx, h.TeamID(), 0, "Zero", h.AdminID())

		assert.Error(t, err)
		assert.Nil(t, award)
		assert.Contains(t, err.Error(), "value cannot be 0")
	})

	t.Run("RepoError", func(t *testing.T) {
		h.TxRepo().On("RunTransaction", mock.Anything, mock.Anything).Return(errors.New("db error")).Once()

		award, err := h.CreateUseCase().Create(ctx, h.TeamID(), 50, "Error", h.AdminID())

		assert.Error(t, err)
		assert.Nil(t, award)
		assert.Contains(t, err.Error(), "db error")
	})
}

func TestAwardUseCase_GetByTeamID(t *testing.T) {
	h := NewAwardTestHelper(t)
	ctx := context.Background()
	teamID := h.TeamID()

	t.Run("Success", func(t *testing.T) {
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
		h.Repo().On("GetByTeamID", ctx, teamID).Return(nil, errors.New("db error")).Once()

		awards, err := h.CreateUseCase().GetByTeamID(ctx, teamID)

		assert.Error(t, err)
		assert.Nil(t, awards)
	})
}
