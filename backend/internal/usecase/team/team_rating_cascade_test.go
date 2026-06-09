package team

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestTeamUseCase_RatingCascadeSoftBanRestoreDelete(t *testing.T) {
	t.Parallel()

	d := newTeamTestDeps(t)
	d.enableRatingRepo(t)
	uc := d.createUseCase()
	teamID := uuid.New()

	d.solveRepo.EXPECT().SoftBanByTeamID(mock.Anything, teamID).Return(nil).Once()
	d.submissionRepo.EXPECT().SoftBanByTeamID(mock.Anything, teamID).Return(nil).Once()
	d.awardRepo.EXPECT().SoftBanByTeamID(mock.Anything, teamID).Return(nil).Once()
	d.ratingRepo.EXPECT().SoftBanByTeamID(mock.Anything, teamID).Return(nil).Once()
	require.NoError(t, uc.cascadeSoftBan(context.Background(), teamID))

	d.solveRepo.EXPECT().RestoreByBannedTeamID(mock.Anything, teamID).Return(nil).Once()
	d.submissionRepo.EXPECT().RestoreByBannedTeamID(mock.Anything, teamID).Return(nil).Once()
	d.awardRepo.EXPECT().RestoreByBannedTeamID(mock.Anything, teamID).Return(nil).Once()
	d.ratingRepo.EXPECT().RestoreByBannedTeamID(mock.Anything, teamID).Return(nil).Once()
	require.NoError(t, uc.cascadeRestore(context.Background(), teamID))

	d.solveRepo.EXPECT().DeleteByTeamID(mock.Anything, teamID).Return(nil).Once()
	d.submissionRepo.EXPECT().DeleteByTeamID(mock.Anything, teamID).Return(nil).Once()
	d.awardRepo.EXPECT().DeleteByTeamID(mock.Anything, teamID).Return(nil).Once()
	d.ratingRepo.EXPECT().DeleteByTeamID(mock.Anything, teamID).Return(nil).Once()
	require.NoError(t, uc.cascadeDelete(context.Background(), teamID))
}
