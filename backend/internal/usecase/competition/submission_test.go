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

	// httputil.FetchPage runs fetchFn then countFn sequentially; a fetch error short-circuits the count call.
	d.submissionRepo.EXPECT().GetByChallenge(mock.Anything, challengeID, mock.Anything, perPage, 0).Return(nil, assert.AnError)

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

	// httputil.FetchPage runs fetchFn then countFn sequentially; a fetch error short-circuits the count call.
	d.submissionRepo.EXPECT().GetByUser(mock.Anything, userID, mock.Anything, perPage, 0).Return(nil, assert.AnError)

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

	// httputil.FetchPage runs fetchFn then countFn sequentially; a fetch error short-circuits the count call.
	d.submissionRepo.EXPECT().GetByTeam(mock.Anything, teamID, mock.Anything, perPage, 0).Return(nil, assert.AnError)

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

	// httputil.FetchPage runs fetchFn then countFn sequentially; a fetch error short-circuits the count call.
	d.submissionRepo.EXPECT().GetAll(mock.Anything, mock.Anything, perPage, 0).Return(nil, assert.AnError)

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

func TestSubmissionUseCase_Update_RequiresTransactionalDeps(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)

	uc := d.createSubmissionUseCase()
	got, err := uc.Update(context.Background(), uuid.New(), true)

	assert.Error(t, err)
	assert.Nil(t, got)
	assert.Contains(t, err.Error(), "transaction manager required")
}

func TestSubmissionUseCase_Discard_RequiresTransactionalDeps(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)

	uc := d.createSubmissionUseCase()
	got, err := uc.Discard(context.Background(), uuid.New())

	assert.Error(t, err)
	assert.Nil(t, got)
	assert.Contains(t, err.Error(), "transaction manager required")
}

func TestSubmissionUseCase_Delete_RequiresTransactionalDeps(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)

	uc := d.createSubmissionUseCase()
	err := uc.Delete(context.Background(), uuid.New())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "transaction manager required")
}

func TestSubmissionUseCase_AdminCreate_Success(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	userID, challengeID := uuid.New(), uuid.New()

	expected := &domain.SubmissionWithDetails{
		Submission: domain.Submission{
			UserID: userID, ChallengeID: challengeID,
			SubmittedFlag: "wrong_flag", IsCorrect: false,
		},
	}

	d.submissionRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)
	d.submissionRepo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(expected, nil)

	uc := d.createSubmissionUseCase()
	got, err := uc.AdminCreate(ctx, userID, nil, challengeID, "wrong_flag", false, "127.0.0.1")

	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, expected.SubmittedFlag, got.SubmittedFlag)
}

func TestSubmissionUseCase_AdminCreate_UserBanned(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	userID, challengeID := uuid.New(), uuid.New()

	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(&domain.User{ID: userID, IsBanned: true}, nil)

	uc := NewSubmissionUseCase(SubmissionDeps{
		SubmissionRepo: d.submissionRepo,
		UserRepo:       d.userRepo,
		Logger:         d.logger,
	})
	got, err := uc.AdminCreate(ctx, userID, nil, challengeID, "flag", false, "127.0.0.1")

	assert.Error(t, err)
	assert.Nil(t, got)
}
