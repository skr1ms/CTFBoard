package challenge

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
)

func TestCommentUseCase_GetByChallengeID_Success(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	ctx := context.Background()
	challengeID := uuid.New()
	userID := uuid.New()
	ch := newTestChallenge(challengeID, "title", "cat", 100, "hash")
	list := []*entity.Comment{newTestComment(userID, challengeID, "text")}

	d.challengeRepo.EXPECT().GetByID(mock.Anything, challengeID).Return(ch, nil)
	d.commentRepo.EXPECT().GetByChallengeID(mock.Anything, challengeID).Return(list, nil)

	uc := d.createCommentUseCase()
	got, err := uc.GetByChallengeID(ctx, challengeID)

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
	d.commentRepo.EXPECT().GetByChallengeID(mock.Anything, challengeID).Return(nil, assert.AnError)

	uc := d.createCommentUseCase()
	got, err := uc.GetByChallengeID(ctx, challengeID)

	assert.Error(t, err)
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
	d.commentRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Run(func(_ context.Context, c *entity.Comment) {
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
	d.commentRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(assert.AnError)

	uc := d.createCommentUseCase()
	got, err := uc.Create(ctx, userID, challengeID, content)

	assert.Error(t, err)
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
