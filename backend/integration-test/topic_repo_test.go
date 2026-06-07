package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	backupuc "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/backup"
)

func TestTopicRepo_CRUDAndChallengeAssignments(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "topic_assignment", 100)
	topicA := f.CreateTopic(t, "web")
	topicB := f.CreateTopic(t, "crypto")
	assert.False(t, topicA.CreatedAt.IsZero())

	require.NoError(t, f.TopicRepo.SetByChallengeID(ctx, challenge.ID, []uuid.UUID{topicA.ID}))

	got, err := f.TopicRepo.GetByChallengeID(ctx, challenge.ID)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, topicA.ID, got[0].ID)

	topicA.Name = "topic_updated_" + uuid.NewString()[:8]
	require.NoError(t, f.TopicRepo.Update(ctx, topicA))
	updated, err := f.TopicRepo.GetByID(ctx, topicA.ID)
	require.NoError(t, err)
	assert.Equal(t, topicA.Name, updated.Name)

	require.NoError(t, f.TopicRepo.SetByChallengeID(ctx, challenge.ID, []uuid.UUID{topicA.ID, topicB.ID}))
	gotByChallenge, err := f.TopicRepo.GetByChallengeIDs(ctx, []uuid.UUID{challenge.ID})
	require.NoError(t, err)
	require.Len(t, gotByChallenge[challenge.ID], 2)

	require.NoError(t, f.TopicRepo.SetByChallengeID(ctx, challenge.ID, nil))
	cleared, err := f.TopicRepo.GetByChallengeID(ctx, challenge.ID)
	require.NoError(t, err)
	assert.Empty(t, cleared)

	require.NoError(t, f.TopicRepo.Delete(ctx, topicB.ID))
	_, err = f.TopicRepo.GetByID(ctx, topicB.ID)
	require.Error(t, err)
}

func TestTopicRepo_ChallengeDeleteCascadesAssignments(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "topic_cascade", 100)
	topic := f.CreateTopic(t, "cascade")
	require.NoError(t, f.TopicRepo.SetByChallengeID(ctx, challenge.ID, []uuid.UUID{topic.ID}))
	require.NoError(t, f.ChallengeRepo.Delete(ctx, challenge.ID))

	var count int

	err := f.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM challenge_topics WHERE challenge_id = $1", challenge.ID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestBackupRepo_ImportTopicsAndChallengeTopicsTx_Success(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challengeID := uuid.New()
	topicID := uuid.New()
	data := &domain.BackupData{
		Topics: []domain.Topic{
			{ID: topicID, Name: "topic_imported", CreatedAt: time.Now().UTC()},
		},
		Challenges: []domain.ChallengeExport{
			{
				Challenge: domain.Challenge{
					ID:           challengeID,
					Title:        "Topic Backup Chall",
					Description:  "Desc",
					Category:     "Web",
					Points:       100,
					FlagHash:     "hash",
					InitialValue: 100,
					MinValue:     100,
					Decay:        0,
				},
				TopicIDs: []uuid.UUID{topicID},
			},
		},
	}

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		if err := f.BackupRepo.ImportTopics(txCtx, data); err != nil {
			return err
		}

		if err := f.BackupRepo.ImportChallenges(txCtx, data); err != nil {
			return err
		}

		return f.BackupRepo.ImportChallengeTopics(txCtx, data)
	})
	require.NoError(t, err)

	topics, err := f.TopicRepo.GetByChallengeID(ctx, challengeID)
	require.NoError(t, err)
	require.Len(t, topics, 1)
	assert.Equal(t, topicID, topics[0].ID)
}

func TestBackupUseCase_ExportIncludesTopics(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "topic_export", 100)
	topic := f.CreateTopic(t, "export")
	require.NoError(t, f.TopicRepo.SetByChallengeID(ctx, challenge.ID, []uuid.UUID{topic.ID}))

	uc := backupuc.NewBackupUseCase(backupuc.BackupDeps{
		CompetitionRepo: f.CompetitionRepo,
		ChallengeRepo:   f.ChallengeRepo,
		TagRepo:         f.TagRepo,
		TopicRepo:       f.TopicRepo,
		HintRepo:        f.HintRepo,
		BracketRepo:     f.BracketRepo,
		CommentRepo:     f.CommentRepo,
		FieldRepo:       f.FieldRepo,
		FieldValueRepo:  f.FieldValueRepo,
		RatingRepo:      f.RatingRepo,
	})

	data, err := uc.Export(ctx, domain.ExportOptions{})
	require.NoError(t, err)

	assert.Contains(t, topicIDsFromBackup(data.Topics), topic.ID)

	var exported *domain.ChallengeExport

	for i := range data.Challenges {
		if data.Challenges[i].ID == challenge.ID {
			exported = &data.Challenges[i]

			break
		}
	}

	require.NotNil(t, exported)
	assert.Contains(t, exported.TopicIDs, topic.ID)
}

func topicIDsFromBackup(topics []domain.Topic) []uuid.UUID {
	ids := make([]uuid.UUID, len(topics))
	for i, topic := range topics {
		ids[i] = topic.ID
	}

	return ids
}
