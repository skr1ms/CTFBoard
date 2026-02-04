package integration_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommentRepo_Create_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user := f.CreateUser(t, "com")
	challenge := f.CreateChallenge(t, "comch", 100)
	comment := &entity.Comment{UserID: user.ID, ChallengeID: challenge.ID, Content: "hello"}
	err := f.CommentRepo.Create(ctx, comment)
	require.NoError(t, err)
	assert.NotEmpty(t, comment.ID)
}

func TestCommentRepo_Create_Error_InvalidUserID(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "comerr", 100)
	comment := &entity.Comment{UserID: uuid.New(), ChallengeID: challenge.ID, Content: "x"}
	err := f.CommentRepo.Create(ctx, comment)
	assert.Error(t, err)
}

func TestCommentRepo_GetByID_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	comment := f.CreateComment(t, f.CreateUser(t, "gbi").ID, f.CreateChallenge(t, "gbi", 100).ID, "content")
	got, err := f.CommentRepo.GetByID(ctx, comment.ID)
	require.NoError(t, err)
	assert.Equal(t, comment.ID, got.ID)
	assert.Equal(t, "content", got.Content)
}

func TestCommentRepo_GetByID_Error_NotFound(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, err := f.CommentRepo.GetByID(ctx, uuid.New())
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrCommentNotFound))
}

func TestCommentRepo_GetByChallengeID_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user := f.CreateUser(t, "gbch")
	challenge := f.CreateChallenge(t, "gbch", 100)
	f.CreateComment(t, user.ID, challenge.ID, "first")
	f.CreateComment(t, user.ID, challenge.ID, "second")
	list, err := f.CommentRepo.GetByChallengeID(ctx, challenge.ID)
	require.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestCommentRepo_GetByChallengeID_Error_CancelledContext(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	challenge := f.CreateChallenge(t, "gbcherr", 100)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := f.CommentRepo.GetByChallengeID(ctx, challenge.ID)
	assert.Error(t, err)
}

func TestCommentRepo_Update_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	comment := f.CreateComment(t, f.CreateUser(t, "upd").ID, f.CreateChallenge(t, "upd", 100).ID, "old")
	comment.Content = "new"
	err := f.CommentRepo.Update(ctx, comment)
	require.NoError(t, err)
	got, err := f.CommentRepo.GetByID(ctx, comment.ID)
	require.NoError(t, err)
	assert.Equal(t, "new", got.Content)
}

func TestCommentRepo_Update_Error_CancelledContext(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	comment := f.CreateComment(t, f.CreateUser(t, "upderr").ID, f.CreateChallenge(t, "upderr", 100).ID, "x")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := f.CommentRepo.Update(ctx, comment)
	assert.Error(t, err)
}

func TestCommentRepo_Delete_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	comment := f.CreateComment(t, f.CreateUser(t, "del").ID, f.CreateChallenge(t, "del", 100).ID, "x")
	err := f.CommentRepo.Delete(ctx, comment.ID)
	require.NoError(t, err)
	_, err = f.CommentRepo.GetByID(ctx, comment.ID)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrCommentNotFound))
}

func TestCommentRepo_Delete_Error_NotFound(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	err := f.CommentRepo.Delete(ctx, uuid.New())
	assert.NoError(t, err)
}
