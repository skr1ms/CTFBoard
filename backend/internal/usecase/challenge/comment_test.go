package challenge

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func TestCommentUseCase_GetByChallengeID_Success(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	ctx := context.Background()
	challengeID := uuid.New()
	userID := uuid.New()
	ch := newTestChallenge(challengeID, "title", "cat", 100, "hash")
	list := []*domain.Comment{newTestComment(userID, challengeID, "text")}

	d.challengeRepo.EXPECT().GetByID(mock.Anything, challengeID).Return(ch, nil)
	d.challengeRepo.EXPECT().GetRequirementsForEnforcement(mock.Anything, challengeID).Return(nil, nil)
	d.commentRepo.EXPECT().GetByChallengeID(mock.Anything, challengeID).Return(list, nil)

	uc := d.createCommentUseCase()
	got, err := uc.GetByChallengeID(ctx, challengeID, nil)

	assert.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Equal(t, challengeID, got[0].ChallengeID)
}

func TestCommentUseCase_GetByChallengeID_Error(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	ctx := context.Background()
	challengeID := uuid.New()
	ch := newTestChallenge(challengeID, "title", "cat", 100, "hash")

	d.challengeRepo.EXPECT().GetByID(mock.Anything, challengeID).Return(ch, nil)
	d.challengeRepo.EXPECT().GetRequirementsForEnforcement(mock.Anything, challengeID).Return(nil, nil)
	d.commentRepo.EXPECT().GetByChallengeID(mock.Anything, challengeID).Return(nil, assert.AnError)

	uc := d.createCommentUseCase()
	got, err := uc.GetByChallengeID(ctx, challengeID, nil)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestCommentUseCase_GetByChallengeID_RequirementsNotMet(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	ctx := context.Background()
	challengeID := uuid.New()
	teamID := uuid.New()
	prereqID := uuid.New()
	ch := newTestChallenge(challengeID, "title", "cat", 100, "hash")
	requirements := []*domain.ChallengeRequirement{{ChallengeID: prereqID, ChallengeTitle: "Prereq"}}

	d.challengeRepo.EXPECT().GetByID(mock.Anything, challengeID).Return(ch, nil)
	d.challengeRepo.EXPECT().GetRequirementsForEnforcement(mock.Anything, challengeID).Return(requirements, nil)
	d.solveRepo.EXPECT().GetSolvedChallengeIDsByTeam(mock.Anything, teamID, mock.Anything).Return([]uuid.UUID{}, nil)

	uc := d.createCommentUseCase()
	got, err := uc.GetByChallengeID(ctx, challengeID, &teamID)

	assert.ErrorIs(t, err, apperr.ErrChallengeNotFound)
	assert.Nil(t, got)
}

func TestCommentUseCase_Create_Success(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	ctx := context.Background()
	userID, challengeID := uuid.New(), uuid.New()
	content := "comment content"
	ch := newTestChallenge(challengeID, "title", "cat", 100, "hash")

	d.challengeRepo.EXPECT().GetByID(mock.Anything, challengeID).Return(ch, nil)
	d.challengeRepo.EXPECT().GetRequirementsForEnforcement(mock.Anything, challengeID).Return(nil, nil)
	d.commentRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Run(func(_ context.Context, c *domain.Comment) {
		assert.Equal(t, userID, c.UserID)
		assert.Equal(t, challengeID, c.ChallengeID)
		assert.Equal(t, content, c.Content)
	})

	uc := d.createCommentUseCase()
	got, err := uc.Create(ctx, userID, challengeID, content)

	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, content, got.Content)
}

func TestCommentUseCase_Create_Error(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	ctx := context.Background()
	userID, challengeID := uuid.New(), uuid.New()
	content := "content"
	ch := newTestChallenge(challengeID, "t", "c", 10, "h")

	d.challengeRepo.EXPECT().GetByID(mock.Anything, challengeID).Return(ch, nil)
	d.challengeRepo.EXPECT().GetRequirementsForEnforcement(mock.Anything, challengeID).Return(nil, nil)
	d.commentRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(assert.AnError)

	uc := d.createCommentUseCase()
	got, err := uc.Create(ctx, userID, challengeID, content)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestCommentUseCase_Create_RequirementsNotMet(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	ctx := context.Background()
	userID, teamID, challengeID := uuid.New(), uuid.New(), uuid.New()
	prereqID := uuid.New()
	ch := newTestChallenge(challengeID, "title", "cat", 100, "hash")
	requirements := []*domain.ChallengeRequirement{{ChallengeID: prereqID, ChallengeTitle: "Prereq"}}

	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(&domain.User{ID: userID, TeamID: &teamID}, nil)
	d.challengeRepo.EXPECT().GetByID(mock.Anything, challengeID).Return(ch, nil)
	d.challengeRepo.EXPECT().GetRequirementsForEnforcement(mock.Anything, challengeID).Return(requirements, nil)
	d.solveRepo.EXPECT().GetSolvedChallengeIDsByTeam(mock.Anything, teamID, []uuid.UUID{prereqID}).Return([]uuid.UUID{}, nil)

	uc := NewCommentUseCase(CommentDeps{
		CommentRepo:   d.commentRepo,
		ChallengeRepo: d.challengeRepo,
		SolveRepo:     d.solveRepo,
		UserRepo:      d.userRepo,
	})
	got, err := uc.Create(ctx, userID, challengeID, "content")

	assert.ErrorIs(t, err, apperr.ErrChallengeNotFound)
	assert.Nil(t, got)
}

func TestCommentUseCase_Create_WasInBannedTeamRejected(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	ctx := context.Background()
	userID, teamID, challengeID := uuid.New(), uuid.New(), uuid.New()

	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(&domain.User{
		ID:              userID,
		TeamID:          &teamID,
		Role:            domain.RoleUser,
		WasInBannedTeam: true,
	}, nil)

	uc := NewCommentUseCase(CommentDeps{
		CommentRepo:   d.commentRepo,
		ChallengeRepo: d.challengeRepo,
		SolveRepo:     d.solveRepo,
		UserRepo:      d.userRepo,
		TeamRepo:      d.teamRepo,
	})
	got, err := uc.Create(ctx, userID, challengeID, "content")

	assert.ErrorIs(t, err, apperr.ErrUserWasInBannedTeam)
	assert.Nil(t, got)
}

func TestCommentUseCase_Delete_Success(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	ctx := context.Background()
	id, userID := uuid.New(), uuid.New()
	c := newTestComment(userID, uuid.New(), "content")
	c.ID = id

	d.commentRepo.EXPECT().GetByID(mock.Anything, id).Return(c, nil)
	d.commentRepo.EXPECT().Delete(mock.Anything, id).Return(nil)

	uc := d.createCommentUseCase()
	err := uc.Delete(ctx, id, userID, false)

	assert.NoError(t, err)
}

func TestCommentUseCase_Delete_Error(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	ctx := context.Background()
	id, userID := uuid.New(), uuid.New()

	d.commentRepo.EXPECT().GetByID(mock.Anything, id).Return(nil, assert.AnError)

	uc := d.createCommentUseCase()
	err := uc.Delete(ctx, id, userID, false)

	assert.Error(t, err)
}

func TestCommentUseCase_Delete_AdminCanDeleteAny_Success(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	ctx := context.Background()
	id, ownerID := uuid.New(), uuid.New()
	adminID := uuid.New()
	c := newTestComment(ownerID, uuid.New(), "content")
	c.ID = id

	d.commentRepo.EXPECT().GetByID(mock.Anything, id).Return(c, nil)
	d.commentRepo.EXPECT().Delete(mock.Anything, id).Return(nil)

	uc := d.createCommentUseCase()
	err := uc.Delete(ctx, id, adminID, true)

	assert.NoError(t, err)
}
