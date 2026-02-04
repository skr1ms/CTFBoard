package competition

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSubmissionUseCase_LogSubmission_Success(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	userID, challengeID := uuid.New(), uuid.New()
	teamID := uuid.New()
	sub := h.NewSubmission(userID, &teamID, challengeID, "flag{test}", false)

	deps.submissionRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)

	uc := h.CreateSubmissionUseCase()
	err := uc.LogSubmission(ctx, sub)

	assert.NoError(t, err)
}

func TestSubmissionUseCase_LogSubmission_Error(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	userID, challengeID := uuid.New(), uuid.New()
	sub := h.NewSubmission(userID, nil, challengeID, "flag", false)

	deps.submissionRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(assert.AnError)

	uc := h.CreateSubmissionUseCase()
	err := uc.LogSubmission(ctx, sub)

	assert.Error(t, err)
}

func TestSubmissionUseCase_GetByChallenge_Success(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	challengeID := uuid.New()
	page, perPage := 1, 20
	list := []*entity.SubmissionWithDetails{}
	total := int64(0)

	deps.submissionRepo.EXPECT().GetByChallenge(mock.Anything, challengeID, perPage, 0).Return(list, nil)
	deps.submissionRepo.EXPECT().CountByChallenge(mock.Anything, challengeID).Return(total, nil)

	uc := h.CreateSubmissionUseCase()
	got, gotTotal, err := uc.GetByChallenge(ctx, challengeID, page, perPage)

	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, total, gotTotal)
}

func TestSubmissionUseCase_GetByChallenge_Error(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	challengeID := uuid.New()
	page, perPage := 1, 20

	deps.submissionRepo.EXPECT().GetByChallenge(mock.Anything, challengeID, perPage, 0).Return(nil, assert.AnError)

	uc := h.CreateSubmissionUseCase()
	got, gotTotal, err := uc.GetByChallenge(ctx, challengeID, page, perPage)

	assert.Error(t, err)
	assert.Nil(t, got)
	assert.Equal(t, int64(0), gotTotal)
}

func TestSubmissionUseCase_GetByUser_Success(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	userID := uuid.New()
	page, perPage := 1, 20
	list := []*entity.SubmissionWithDetails{}
	total := int64(0)

	deps.submissionRepo.EXPECT().GetByUser(mock.Anything, userID, perPage, 0).Return(list, nil)
	deps.submissionRepo.EXPECT().CountByUser(mock.Anything, userID).Return(total, nil)

	uc := h.CreateSubmissionUseCase()
	got, gotTotal, err := uc.GetByUser(ctx, userID, page, perPage)

	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, total, gotTotal)
}

func TestSubmissionUseCase_GetByUser_Error(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	userID := uuid.New()
	page, perPage := 1, 20

	deps.submissionRepo.EXPECT().GetByUser(mock.Anything, userID, perPage, 0).Return(nil, assert.AnError)

	uc := h.CreateSubmissionUseCase()
	got, gotTotal, err := uc.GetByUser(ctx, userID, page, perPage)

	assert.Error(t, err)
	assert.Nil(t, got)
	assert.Equal(t, int64(0), gotTotal)
}

func TestSubmissionUseCase_GetByTeam_Success(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	teamID := uuid.New()
	page, perPage := 1, 20
	list := []*entity.SubmissionWithDetails{}
	total := int64(0)

	deps.submissionRepo.EXPECT().GetByTeam(mock.Anything, teamID, perPage, 0).Return(list, nil)
	deps.submissionRepo.EXPECT().CountByTeam(mock.Anything, teamID).Return(total, nil)

	uc := h.CreateSubmissionUseCase()
	got, gotTotal, err := uc.GetByTeam(ctx, teamID, page, perPage)

	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, total, gotTotal)
}

func TestSubmissionUseCase_GetByTeam_Error(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	teamID := uuid.New()
	page, perPage := 1, 20

	deps.submissionRepo.EXPECT().GetByTeam(mock.Anything, teamID, perPage, 0).Return(nil, assert.AnError)

	uc := h.CreateSubmissionUseCase()
	got, gotTotal, err := uc.GetByTeam(ctx, teamID, page, perPage)

	assert.Error(t, err)
	assert.Nil(t, got)
	assert.Equal(t, int64(0), gotTotal)
}

func TestSubmissionUseCase_GetAll_Success(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	page, perPage := 1, 20
	list := []*entity.SubmissionWithDetails{}
	total := int64(0)

	deps.submissionRepo.EXPECT().GetAll(mock.Anything, perPage, 0).Return(list, nil)
	deps.submissionRepo.EXPECT().CountAll(mock.Anything).Return(total, nil)

	uc := h.CreateSubmissionUseCase()
	got, gotTotal, err := uc.GetAll(ctx, page, perPage)

	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, total, gotTotal)
}

func TestSubmissionUseCase_GetAll_Error(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	page, perPage := 1, 20

	deps.submissionRepo.EXPECT().GetAll(mock.Anything, perPage, 0).Return(nil, assert.AnError)

	uc := h.CreateSubmissionUseCase()
	got, gotTotal, err := uc.GetAll(ctx, page, perPage)

	assert.Error(t, err)
	assert.Nil(t, got)
	assert.Equal(t, int64(0), gotTotal)
}

func TestSubmissionUseCase_GetStats_Success(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	challengeID := uuid.New()
	stats := &entity.SubmissionStats{Total: 10, Correct: 3, Incorrect: 7}

	deps.submissionRepo.EXPECT().GetStats(mock.Anything, challengeID).Return(stats, nil)

	uc := h.CreateSubmissionUseCase()
	got, err := uc.GetStats(ctx, challengeID)

	assert.NoError(t, err)
	assert.Equal(t, stats.Total, got.Total)
	assert.Equal(t, stats.Correct, got.Correct)
}

func TestSubmissionUseCase_GetStats_Error(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	challengeID := uuid.New()

	deps.submissionRepo.EXPECT().GetStats(mock.Anything, challengeID).Return(nil, assert.AnError)

	uc := h.CreateSubmissionUseCase()
	got, err := uc.GetStats(ctx, challengeID)

	assert.Error(t, err)
	assert.Nil(t, got)
}
