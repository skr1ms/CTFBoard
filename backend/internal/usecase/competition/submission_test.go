package competition

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func TestSubmissionUseCase_LogSubmission_Success(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	userID, challengeID := uuid.New(), uuid.New()
	teamID := uuid.New()
	sub := newTestSubmission(userID, &teamID, challengeID, "flag{test}", false)

	d.submissionRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)

	uc := d.createSubmissionUseCase()
	err := uc.LogSubmission(ctx, sub)

	assert.NoError(t, err)
}

func TestSubmissionUseCase_LogSubmission_Error(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	userID, challengeID := uuid.New(), uuid.New()
	sub := newTestSubmission(userID, nil, challengeID, "flag", false)

	d.submissionRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(assert.AnError)

	uc := d.createSubmissionUseCase()
	err := uc.LogSubmission(ctx, sub)

	assert.Error(t, err)
}

func TestSubmissionUseCase_GetByChallenge_Success(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	challengeID := uuid.New()
	page, perPage := 1, 20

	var list []*domain.SubmissionWithDetails

	total := int64(0)

	d.submissionRepo.EXPECT().GetByChallenge(mock.Anything, challengeID, mock.Anything, perPage, 0).Return(list, nil)
	d.submissionRepo.EXPECT().CountByChallenge(mock.Anything, challengeID, mock.Anything).Return(total, nil)

	uc := d.createSubmissionUseCase()
	got, err := uc.GetByChallenge(ctx, challengeID, page, perPage, false)

	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, total, got.Total)
}

func TestSubmissionUseCase_GetByChallenge_Error(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	challengeID := uuid.New()
	page, perPage := 1, 20

	d.submissionRepo.EXPECT().GetByChallenge(mock.Anything, challengeID, mock.Anything, perPage, 0).Return(nil, assert.AnError)
	d.submissionRepo.EXPECT().CountByChallenge(mock.Anything, challengeID, mock.Anything).Return(int64(0), nil)

	uc := d.createSubmissionUseCase()
	got, err := uc.GetByChallenge(ctx, challengeID, page, perPage, false)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestSubmissionUseCase_GetByUser_Success(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	userID := uuid.New()
	page, perPage := 1, 20

	var list []*domain.SubmissionWithDetails

	total := int64(0)

	d.submissionRepo.EXPECT().GetByUser(mock.Anything, userID, mock.Anything, perPage, 0).Return(list, nil)
	d.submissionRepo.EXPECT().CountByUser(mock.Anything, userID, mock.Anything).Return(total, nil)

	uc := d.createSubmissionUseCase()
	got, err := uc.GetByUser(ctx, userID, page, perPage, false)

	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, total, got.Total)
}

func TestSubmissionUseCase_GetByUser_Error(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	userID := uuid.New()
	page, perPage := 1, 20

	d.submissionRepo.EXPECT().GetByUser(mock.Anything, userID, mock.Anything, perPage, 0).Return(nil, assert.AnError)
	d.submissionRepo.EXPECT().CountByUser(mock.Anything, userID, mock.Anything).Return(int64(0), nil)

	uc := d.createSubmissionUseCase()
	got, err := uc.GetByUser(ctx, userID, page, perPage, false)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestSubmissionUseCase_GetByTeam_Success(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	teamID := uuid.New()
	page, perPage := 1, 20

	var list []*domain.SubmissionWithDetails

	total := int64(0)

	d.submissionRepo.EXPECT().GetByTeam(mock.Anything, teamID, mock.Anything, perPage, 0).Return(list, nil)
	d.submissionRepo.EXPECT().CountByTeam(mock.Anything, teamID, mock.Anything).Return(total, nil)

	uc := d.createSubmissionUseCase()
	got, err := uc.GetByTeam(ctx, teamID, page, perPage, false)

	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, total, got.Total)
}

func TestSubmissionUseCase_GetByTeam_Error(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	teamID := uuid.New()
	page, perPage := 1, 20

	d.submissionRepo.EXPECT().GetByTeam(mock.Anything, teamID, mock.Anything, perPage, 0).Return(nil, assert.AnError)
	d.submissionRepo.EXPECT().CountByTeam(mock.Anything, teamID, mock.Anything).Return(int64(0), nil)

	uc := d.createSubmissionUseCase()
	got, err := uc.GetByTeam(ctx, teamID, page, perPage, false)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestSubmissionUseCase_GetAll_Success(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	page, perPage := 1, 20

	var list []*domain.SubmissionWithDetails

	total := int64(0)

	d.submissionRepo.EXPECT().GetAll(mock.Anything, mock.Anything, perPage, 0).Return(list, nil)
	d.submissionRepo.EXPECT().CountAll(mock.Anything, mock.Anything).Return(total, nil)

	uc := d.createSubmissionUseCase()
	got, err := uc.GetAll(ctx, page, perPage, false)

	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, total, got.Total)
}

func TestSubmissionUseCase_GetAll_Error(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	page, perPage := 1, 20

	d.submissionRepo.EXPECT().GetAll(mock.Anything, mock.Anything, perPage, 0).Return(nil, assert.AnError)
	d.submissionRepo.EXPECT().CountAll(mock.Anything, mock.Anything).Return(int64(0), nil)

	uc := d.createSubmissionUseCase()
	got, err := uc.GetAll(ctx, page, perPage, false)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestSubmissionUseCase_GetStats_Success(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	challengeID := uuid.New()
	stats := &domain.SubmissionStats{Total: 10, Correct: 3, Incorrect: 7}

	d.submissionRepo.EXPECT().GetStats(mock.Anything, challengeID, mock.Anything).Return(stats, nil)

	uc := d.createSubmissionUseCase()
	got, err := uc.GetStats(ctx, challengeID, false)

	assert.NoError(t, err)
	assert.Equal(t, stats.Total, got.Total)
	assert.Equal(t, stats.Correct, got.Correct)
}

func TestSubmissionUseCase_GetStats_Error(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	challengeID := uuid.New()

	d.submissionRepo.EXPECT().GetStats(mock.Anything, challengeID, mock.Anything).Return(nil, assert.AnError)

	uc := d.createSubmissionUseCase()
	got, err := uc.GetStats(ctx, challengeID, false)

	assert.Error(t, err)
	assert.Nil(t, got)
}
