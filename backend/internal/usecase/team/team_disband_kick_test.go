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

func TestTeamUseCase_DisbandTeam_Success(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	captainID := uuid.New()
	teamID := uuid.New()
	captain := &domain.User{ID: captainID, TeamID: &teamID}

	d.compRepo.EXPECT().Get(mock.Anything).Return(&domain.Competition{Mode: domain.ModeTeamsOnly, AllowTeamSwitch: true}, nil).Once()
	d.compRepo.EXPECT().GetForUpdate(mock.Anything).Return(&domain.Competition{Mode: domain.ModeTeamsOnly, AllowTeamSwitch: true}, nil).Once()
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, captainID).Return(nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, captainID).Return(captain, nil).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(&domain.Team{ID: teamID, CaptainID: captainID, Name: "test_team"}, nil).Once()
	d.userRepo.EXPECT().GetByTeamID(mock.Anything, teamID).Return([]*domain.User{captain}, nil).Once()
	d.solveRepo.EXPECT().GetByTeamIDWithDetails(mock.Anything, teamID).Return(nil, nil).Once()
	d.solveRepo.EXPECT().DeleteByTeamID(mock.Anything, teamID).Return(nil).Once()
	d.submissionRepo.EXPECT().DeleteByTeamID(mock.Anything, teamID).Return(nil).Once()
	d.awardRepo.EXPECT().DeleteByTeamID(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().CreateAuditLog(mock.Anything, mock.MatchedBy(func(l *domain.TeamAuditLog) bool {
		return l.Action == domain.TeamActionDeleted && l.TeamID == teamID && l.UserID != nil && *l.UserID == captainID
	})).Return(nil).Once()
	d.userRepo.EXPECT().UpdateTeamIDBatch(mock.Anything, mock.MatchedBy(func(ids []uuid.UUID) bool {
		return len(ids) == 1 && ids[0] == captainID
	}), mock.Anything).Return(nil).Once()
	d.teamRepo.EXPECT().Delete(mock.Anything, teamID).Return(nil).Once()

	uc := d.createUseCase()
	err := uc.DisbandTeam(context.Background(), captainID)

	assert.NoError(t, err)
}

func TestTeamUseCase_DisbandTeam_WithSolves_UsesGetByIDs(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	captainID := uuid.New()
	teamID := uuid.New()
	ch1ID := uuid.New()
	ch2ID := uuid.New()
	captain := &domain.User{ID: captainID, TeamID: &teamID}

	solvesWithDetails := []*domain.SolveWithDetails{
		{Solve: domain.Solve{ChallengeID: ch1ID}, ChallengePoints: 100, ChallengeTitle: "Ch1"},
		{Solve: domain.Solve{ChallengeID: ch1ID}, ChallengePoints: 100, ChallengeTitle: "Ch1"},
		{Solve: domain.Solve{ChallengeID: ch2ID}, ChallengePoints: 200, ChallengeTitle: "Ch2"},
	}

	d.compRepo.EXPECT().Get(mock.Anything).Return(&domain.Competition{Mode: domain.ModeTeamsOnly, AllowTeamSwitch: true}, nil).Once()
	d.compRepo.EXPECT().GetForUpdate(mock.Anything).Return(&domain.Competition{Mode: domain.ModeTeamsOnly, AllowTeamSwitch: true}, nil).Once()
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, captainID).Return(nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, captainID).Return(captain, nil).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(&domain.Team{ID: teamID, CaptainID: captainID, Name: "test_team"}, nil).Once()
	d.userRepo.EXPECT().GetByTeamID(mock.Anything, teamID).Return([]*domain.User{captain}, nil).Once()
	d.solveRepo.EXPECT().GetByTeamIDWithDetails(mock.Anything, teamID).Return(solvesWithDetails, nil).Once()
	d.challengeRepo.EXPECT().RecalculateSolveCounts(mock.Anything, mock.MatchedBy(func(ids []uuid.UUID) bool {
		return len(ids) == 2 && (ids[0] == ch1ID && ids[1] == ch2ID || ids[0] == ch2ID && ids[1] == ch1ID)
	})).Return(nil).Once()

	ch1After := &domain.Challenge{ID: ch1ID, InitialValue: 100, MinValue: 50, Decay: 10, SolveCount: 0}
	ch2After := &domain.Challenge{ID: ch2ID, InitialValue: 200, MinValue: 100, Decay: 20, SolveCount: 0}
	d.challengeRepo.EXPECT().GetByIDs(mock.Anything, mock.MatchedBy(func(ids []uuid.UUID) bool {
		return len(ids) == 2 && (ids[0] == ch1ID && ids[1] == ch2ID || ids[0] == ch2ID && ids[1] == ch1ID)
	})).Return(map[uuid.UUID]*domain.Challenge{ch1ID: ch1After, ch2ID: ch2After}, nil).Once()
	d.challengeRepo.EXPECT().BatchUpdatePoints(mock.Anything, mock.MatchedBy(func(ids []uuid.UUID) bool {
		return len(ids) == 2
	}), mock.MatchedBy(func(points []int) bool {
		return len(points) == 2 && ((points[0] == 100 && points[1] == 200) || (points[0] == 200 && points[1] == 100))
	})).Return(nil).Once()
	d.solveRepo.EXPECT().GetSolvesForPointsRecalc(mock.Anything, mock.MatchedBy(func(ids []uuid.UUID) bool {
		return len(ids) == 2 && (ids[0] == ch1ID && ids[1] == ch2ID || ids[0] == ch2ID && ids[1] == ch1ID)
	})).Return(nil, nil).Once()
	d.solveRepo.EXPECT().DeleteByTeamID(mock.Anything, teamID).Return(nil).Once()
	d.submissionRepo.EXPECT().DeleteByTeamID(mock.Anything, teamID).Return(nil).Once()
	d.awardRepo.EXPECT().DeleteByTeamID(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().CreateAuditLog(mock.Anything, mock.MatchedBy(func(l *domain.TeamAuditLog) bool {
		return l.Action == domain.TeamActionDeleted && l.TeamID == teamID && l.UserID != nil && *l.UserID == captainID
	})).Return(nil).Once()
	d.userRepo.EXPECT().UpdateTeamIDBatch(mock.Anything, mock.MatchedBy(func(ids []uuid.UUID) bool {
		return len(ids) == 1 && ids[0] == captainID
	}), mock.Anything).Return(nil).Once()
	d.teamRepo.EXPECT().Delete(mock.Anything, teamID).Return(nil).Once()

	uc := d.createUseCase()
	err := uc.DisbandTeam(context.Background(), captainID)

	assert.NoError(t, err)
}

func TestTeamUseCase_DisbandTeam_BannedCaptain_ReturnsErrUserBanned(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	captainID := uuid.New()
	teamID := uuid.New()
	captain := &domain.User{ID: captainID, TeamID: &teamID, IsBanned: true}

	d.compRepo.EXPECT().Get(mock.Anything).Return(&domain.Competition{Mode: domain.ModeTeamsOnly, AllowTeamSwitch: true}, nil).Once()
	d.compRepo.EXPECT().GetForUpdate(mock.Anything).Return(&domain.Competition{Mode: domain.ModeTeamsOnly, AllowTeamSwitch: true}, nil).Once()
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, captainID).Return(nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, captainID).Return(captain, nil).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(&domain.Team{ID: teamID, CaptainID: captainID, Name: "test_team"}, nil).Once()
	d.userRepo.EXPECT().GetByTeamID(mock.Anything, teamID).Return([]*domain.User{captain}, nil).Once()

	uc := d.createUseCase()
	err := uc.DisbandTeam(context.Background(), captainID)

	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrUserBanned)
}

func TestTeamUseCase_KickMember_Success(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	captainID := uuid.New()
	targetID := uuid.New()
	teamID := uuid.New()
	captain := &domain.User{ID: captainID, TeamID: &teamID}
	target := &domain.User{ID: targetID, TeamID: &teamID}

	d.compRepo.EXPECT().Get(mock.Anything).Return(&domain.Competition{Mode: "teams_only", AllowTeamSwitch: true}, nil).Once()
	d.compRepo.EXPECT().GetForUpdate(mock.Anything).Return(&domain.Competition{Mode: "teams_only", AllowTeamSwitch: true}, nil).Once()
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, mock.Anything).Return(nil).Twice()
	d.userRepo.EXPECT().GetByID(mock.Anything, captainID).Return(captain, nil).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(&domain.Team{ID: teamID, CaptainID: captainID, Name: "test_team"}, nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, targetID).Return(target, nil).Once()
	d.userRepo.EXPECT().GetByTeamID(mock.Anything, teamID).Return([]*domain.User{captain, target}, nil).Once()
	d.userRepo.EXPECT().UpdateTeamID(mock.Anything, targetID, (*uuid.UUID)(nil)).Return(nil).Once()
	d.teamRepo.EXPECT().CreateAuditLog(mock.Anything, mock.MatchedBy(func(l *domain.TeamAuditLog) bool {
		targetIDStr := targetID.String()
		detailsTargetID, ok := l.Details["target_user_id"].(string)

		return l.Action == domain.TeamActionMemberKicked &&
			l.TeamID == teamID &&
			l.UserID != nil && *l.UserID == captainID &&
			ok && detailsTargetID == targetIDStr
	})).Return(nil).Once()

	uc := d.createUseCase()
	err := uc.KickMember(context.Background(), captainID, targetID)

	assert.NoError(t, err)
}
