package team

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func TestTeamUseCase_TransferCaptain_Success(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	captainID := uuid.New()
	newCaptainID := uuid.New()
	teamID := uuid.New()

	captain := &domain.User{
		ID:     captainID,
		TeamID: &teamID,
	}

	newCaptain := &domain.User{
		ID:     newCaptainID,
		TeamID: &teamID,
	}

	team := &domain.Team{
		ID:        teamID,
		CaptainID: captainID,
	}

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.compRepo.EXPECT().Get(mock.Anything).Return(&domain.Competition{Mode: "teams_only", AllowTeamSwitch: true}, nil).Times(2)
	d.userRepo.EXPECT().Lock(mock.Anything, mock.Anything).Return(nil).Times(2)
	d.userRepo.EXPECT().GetByID(mock.Anything, captainID).Return(captain, nil).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, newCaptainID).Return(newCaptain, nil).Once()
	d.teamRepo.EXPECT().UpdateCaptain(mock.Anything, teamID, newCaptainID).Return(nil).Once()
	d.teamRepo.EXPECT().CreateAuditLog(mock.Anything, mock.MatchedBy(func(log *domain.TeamAuditLog) bool {
		return log.Action == domain.TeamActionCaptainTransfer
	})).Return(nil).Once()

	uc := d.createUseCase()

	err := uc.TransferCaptain(context.Background(), captainID, newCaptainID)

	assert.NoError(t, err)
}

func TestTeamUseCase_TransferCaptain_NotCaptain_Error(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	userID := uuid.New()
	realCaptainID := uuid.New()
	newCaptainID := uuid.New()
	teamID := uuid.New()

	user := &domain.User{
		ID:     userID,
		TeamID: &teamID,
	}

	team := &domain.Team{
		ID:        teamID,
		CaptainID: realCaptainID,
	}

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.compRepo.EXPECT().Get(mock.Anything).Return(&domain.Competition{Mode: "teams_only", AllowTeamSwitch: true}, nil).Times(2)
	d.userRepo.EXPECT().Lock(mock.Anything, mock.Anything).Return(nil).Times(2)
	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil).Once()
	newCaptain := &domain.User{ID: newCaptainID, TeamID: &teamID}
	d.userRepo.EXPECT().GetByID(mock.Anything, newCaptainID).Return(newCaptain, nil).Once()

	uc := d.createUseCase()

	err := uc.TransferCaptain(context.Background(), userID, newCaptainID)

	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrNotCaptain)
}

func TestTeamUseCase_RosterFrozen_BlocksAllOperations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		setup  func(deps *teamTestDeps)
		action func(uc *TeamUseCase) error
	}{
		{"Create", func(deps *teamTestDeps) {
			deps.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
				return fn(ctx)
			}).Once()
			deps.compRepo.EXPECT().Get(mock.Anything).Return(&domain.Competition{AllowTeamSwitch: false}, nil).Once()
		}, func(uc *TeamUseCase) error {
			_, err := uc.Create(context.Background(), "test_team", uuid.New(), false, false)

			return err
		}},
		{"Join", nil, func(uc *TeamUseCase) error {
			_, err := uc.Join(context.Background(), uuid.New(), uuid.New(), false)

			return err
		}},
		{"CreateSoloTeam", func(deps *teamTestDeps) {
			deps.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
				return fn(ctx)
			}).Once()
			deps.compRepo.EXPECT().Get(mock.Anything).Return(&domain.Competition{AllowTeamSwitch: false}, nil).Once()
		}, func(uc *TeamUseCase) error {
			_, err := uc.CreateSoloTeam(context.Background(), uuid.New(), false)

			return err
		}},
		{"Leave", nil, func(uc *TeamUseCase) error { return uc.Leave(context.Background(), uuid.New()) }},
		{"TransferCaptain", nil, func(uc *TeamUseCase) error {
			return uc.TransferCaptain(context.Background(), uuid.New(), uuid.New())
		}},
		{"DisbandTeam", nil, func(uc *TeamUseCase) error { return uc.DisbandTeam(context.Background(), uuid.New()) }},
		{"KickMember", nil, func(uc *TeamUseCase) error { return uc.KickMember(context.Background(), uuid.New(), uuid.New()) }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			d := newTeamTestDeps(t)

			if tc.setup != nil {
				tc.setup(d)
			} else {
				d.compRepo.EXPECT().Get(mock.Anything).Return(&domain.Competition{AllowTeamSwitch: false}, nil).Once()
			}

			uc := d.createUseCase()
			err := tc.action(uc)
			assert.Error(t, err)
			assert.ErrorIs(t, err, apperr.ErrRosterFrozen)
		})
	}
}
