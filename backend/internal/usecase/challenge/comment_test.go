package challenge

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCommentUseCase_GetByChallengeID_Success(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	challengeID := uuid.New()
	userID := uuid.New()
	list := []*entity.Comment{h.NewComment(userID, challengeID, "text")}

	deps.commentRepo.EXPECT().GetByChallengeID(mock.Anything, challengeID).Return(list, nil)

	uc := h.CreateCommentUseCase()
	got, err := uc.GetByChallengeID(ctx, challengeID)

	assert.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Equal(t, challengeID, got[0].ChallengeID)
}

func TestCommentUseCase_GetByChallengeID_Error(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	challengeID := uuid.New()

	deps.commentRepo.EXPECT().GetByChallengeID(mock.Anything, challengeID).Return(nil, assert.AnError)

	uc := h.CreateCommentUseCase()
	got, err := uc.GetByChallengeID(ctx, challengeID)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestCommentUseCase_Create_Success(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	userID, challengeID := uuid.New(), uuid.New()
	content := "comment content"
	ch := h.NewChallenge(challengeID, "title", "cat", 100, "hash")

	deps.challengeRepo.EXPECT().GetByID(mock.Anything, challengeID).Return(ch, nil)
	deps.commentRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Run(func(_ context.Context, c *entity.Comment) {
		assert.Equal(t, userID, c.UserID)
		assert.Equal(t, challengeID, c.ChallengeID)
		assert.Equal(t, content, c.Content)
	})

	uc := h.CreateCommentUseCase()
	got, err := uc.Create(ctx, userID, challengeID, content)

	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, content, got.Content)
}

func TestCommentUseCase_Create_Error(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	userID, challengeID := uuid.New(), uuid.New()
	content := "content"
	ch := h.NewChallenge(challengeID, "t", "c", 10, "h")

	deps.challengeRepo.EXPECT().GetByID(mock.Anything, challengeID).Return(ch, nil)
	deps.commentRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(assert.AnError)

	uc := h.CreateCommentUseCase()
	got, err := uc.Create(ctx, userID, challengeID, content)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestCommentUseCase_Delete_Success(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	id, userID := uuid.New(), uuid.New()
	c := h.NewComment(userID, uuid.New(), "content")
	c.ID = id

	deps.commentRepo.EXPECT().GetByID(mock.Anything, id).Return(c, nil)
	deps.commentRepo.EXPECT().Delete(mock.Anything, id).Return(nil)

	uc := h.CreateCommentUseCase()
	err := uc.Delete(ctx, id, userID)

	assert.NoError(t, err)
}

func TestCommentUseCase_Delete_Error(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	id, userID := uuid.New(), uuid.New()

	deps.commentRepo.EXPECT().GetByID(mock.Anything, id).Return(nil, assert.AnError)

	uc := h.CreateCommentUseCase()
	err := uc.Delete(ctx, id, userID)

	assert.Error(t, err)
}
