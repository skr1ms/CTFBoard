package challenge

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func expectTopicTx(d *challengeTestDeps) {
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
}

func TestTopicUseCase_Create_Success(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	ctx := context.Background()
	name := "Web"

	d.topicRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Run(func(_ context.Context, topic *domain.Topic) {
		assert.Equal(t, name, topic.Name)
		assert.NotEqual(t, uuid.Nil, topic.ID)
	}).Once()

	got, err := d.createTopicUseCase().Create(ctx, name)

	require.NoError(t, err)
	assert.Equal(t, name, got.Name)
	assert.NotEqual(t, uuid.Nil, got.ID)
}

func TestTopicUseCase_Create_NameRequired(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)

	got, err := d.createTopicUseCase().Create(context.Background(), "")

	require.ErrorIs(t, err, apperr.ErrTopicNameRequired)
	assert.Nil(t, got)
}

func TestTopicUseCase_Update_Success(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	ctx := context.Background()
	id := uuid.New()
	topic := newTestTopic("old")
	topic.ID = id

	d.topicRepo.EXPECT().GetByID(mock.Anything, id).Return(topic, nil).Once()
	d.topicRepo.EXPECT().Update(mock.Anything, mock.Anything).Return(nil).Run(func(_ context.Context, topic *domain.Topic) {
		assert.Equal(t, id, topic.ID)
		assert.Equal(t, "new", topic.Name)
	}).Once()

	got, err := d.createTopicUseCase().Update(ctx, id, "new")

	require.NoError(t, err)
	assert.Equal(t, "new", got.Name)
}

func TestTopicUseCase_GetByChallengeID_ChallengeNotFound(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	ctx := context.Background()
	challengeID := uuid.New()

	d.challengeRepo.EXPECT().GetByID(mock.Anything, challengeID).Return(nil, apperr.ErrChallengeNotFound).Once()

	got, err := d.createTopicUseCase().GetByChallengeID(ctx, challengeID)

	require.ErrorIs(t, err, apperr.ErrChallengeNotFound)
	assert.Nil(t, got)
}

func TestTopicUseCase_SetByChallengeID_ReplacesDedupedTopics(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	ctx := context.Background()
	challengeID := uuid.New()
	topicA := newTestTopic("web")
	topicB := newTestTopic("crypto")
	rawIDs := []uuid.UUID{topicA.ID, topicB.ID, topicA.ID}
	wantIDs := []uuid.UUID{topicA.ID, topicB.ID}

	expectTopicTx(d)
	d.challengeRepo.EXPECT().GetByID(mock.Anything, challengeID).Return(newTestChallenge(challengeID, "ch", "web", 100, "hash"), nil).Once()
	d.topicRepo.EXPECT().GetByIDs(mock.Anything, wantIDs).Return(map[uuid.UUID]*domain.Topic{
		topicA.ID: topicA,
		topicB.ID: topicB,
	}, nil).Once()
	d.topicRepo.EXPECT().SetByChallengeID(mock.Anything, challengeID, wantIDs).Return(nil).Once()

	err := d.createTopicUseCase().SetByChallengeID(ctx, challengeID, rawIDs)

	require.NoError(t, err)
}

func TestTopicUseCase_SetByChallengeID_EmptyClearsTopics(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	ctx := context.Background()
	challengeID := uuid.New()

	expectTopicTx(d)
	d.challengeRepo.EXPECT().GetByID(mock.Anything, challengeID).Return(newTestChallenge(challengeID, "ch", "web", 100, "hash"), nil).Once()
	d.topicRepo.EXPECT().SetByChallengeID(mock.Anything, challengeID, []uuid.UUID{}).Return(nil).Once()

	err := d.createTopicUseCase().SetByChallengeID(ctx, challengeID, nil)

	require.NoError(t, err)
}

func TestTopicUseCase_SetByChallengeID_InvalidTopic(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	ctx := context.Background()
	challengeID := uuid.New()
	missingTopicID := uuid.New()

	expectTopicTx(d)
	d.challengeRepo.EXPECT().GetByID(mock.Anything, challengeID).Return(newTestChallenge(challengeID, "ch", "web", 100, "hash"), nil).Once()
	d.topicRepo.EXPECT().GetByIDs(mock.Anything, []uuid.UUID{missingTopicID}).Return(map[uuid.UUID]*domain.Topic{}, nil).Once()

	err := d.createTopicUseCase().SetByChallengeID(ctx, challengeID, []uuid.UUID{missingTopicID})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid topic_id")
}
