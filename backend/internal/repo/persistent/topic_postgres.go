package persistent

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wahrwelt-kit/go-pgkit/pgutil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
)

type TopicRepo struct {
	BaseRepo
}

var _ repo.TopicRepository = (*TopicRepo)(nil)

func NewTopicRepo(pool *pgxpool.Pool) *TopicRepo {
	return &TopicRepo{BaseRepo: BaseRepo{pool: pool}}
}

func (r *TopicRepo) Create(ctx context.Context, topic *domain.Topic) error {
	EnsureID(&topic.ID)

	row, err := r.Q(ctx).CreateTopic(ctx, sqlc.CreateTopicParams{
		ID:   topic.ID,
		Name: topic.Name,
	})
	if err != nil {
		return fmt.Errorf("TopicRepo - Create: %w", err)
	}

	*topic = *topicFromSQLC(row)

	return nil
}

func (r *TopicRepo) GetByID(ctx context.Context, ID uuid.UUID) (*domain.Topic, error) {
	row, err := GetOrNotFound(func() (sqlc.Topic, error) { return r.Q(ctx).GetTopicByID(ctx, ID) }, apperr.ErrTopicNotFound, "TopicRepo - GetByID")
	if err != nil {
		return nil, err
	}

	return topicFromSQLC(row), nil
}

func (r *TopicRepo) GetByName(ctx context.Context, name string) (*domain.Topic, error) {
	row, err := GetOrNotFound(func() (sqlc.Topic, error) { return r.Q(ctx).GetTopicByName(ctx, name) }, apperr.ErrTopicNotFound, "TopicRepo - GetByName")
	if err != nil {
		return nil, err
	}

	return topicFromSQLC(row), nil
}

func (r *TopicRepo) GetByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*domain.Topic, error) {
	if len(ids) == 0 {
		return map[uuid.UUID]*domain.Topic{}, nil
	}

	rows, err := r.Q(ctx).GetTopicsByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("TopicRepo - GetByIDs: %w", err)
	}

	out := make(map[uuid.UUID]*domain.Topic, len(rows))
	for _, row := range rows {
		topic := topicFromSQLC(row)
		out[topic.ID] = topic
	}

	return out, nil
}

func (r *TopicRepo) GetAll(ctx context.Context) ([]*domain.Topic, error) {
	rows, err := r.Q(ctx).GetAllTopics(ctx)
	if err != nil {
		return nil, fmt.Errorf("TopicRepo - GetAll: %w", err)
	}

	out := make([]*domain.Topic, len(rows))
	for i, row := range rows {
		out[i] = topicFromSQLC(row)
	}

	return out, nil
}

func (r *TopicRepo) Update(ctx context.Context, topic *domain.Topic) error {
	err := r.Q(ctx).UpdateTopic(ctx, sqlc.UpdateTopicParams{
		ID:   topic.ID,
		Name: topic.Name,
	})
	if err != nil {
		return fmt.Errorf("TopicRepo - Update: %w", err)
	}

	return nil
}

// Delete removes a topic by ID. Idempotent: returns nil if the topic does not exist.
func (r *TopicRepo) Delete(ctx context.Context, ID uuid.UUID) error {
	err := r.Q(ctx).DeleteTopic(ctx, ID)
	if err != nil {
		return fmt.Errorf("TopicRepo - Delete: %w", err)
	}

	return nil
}

func (r *TopicRepo) GetByChallengeID(ctx context.Context, challengeID uuid.UUID) ([]*domain.Topic, error) {
	rows, err := r.Q(ctx).GetTopicsByChallengeID(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("TopicRepo - GetByChallengeID: %w", err)
	}

	out := make([]*domain.Topic, len(rows))
	for i, row := range rows {
		out[i] = &domain.Topic{
			ID:        row.ID,
			Name:      row.Name,
			CreatedAt: pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(row.CreatedAt)),
		}
	}

	return out, nil
}

func (r *TopicRepo) GetByChallengeIDs(ctx context.Context, challengeIDs []uuid.UUID) (map[uuid.UUID][]*domain.Topic, error) {
	if len(challengeIDs) == 0 {
		return map[uuid.UUID][]*domain.Topic{}, nil
	}

	rows, err := r.Q(ctx).GetTopicsByChallengeIDs(ctx, challengeIDs)
	if err != nil {
		return nil, fmt.Errorf("TopicRepo - GetByChallengeIDs: %w", err)
	}

	out := make(map[uuid.UUID][]*domain.Topic)

	for _, row := range rows {
		topic := &domain.Topic{
			ID:        row.ID,
			Name:      row.Name,
			CreatedAt: pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(row.CreatedAt)),
		}
		out[row.ChallengeID] = append(out[row.ChallengeID], topic)
	}

	return out, nil
}

func (r *TopicRepo) SetByChallengeID(ctx context.Context, challengeID uuid.UUID, topicIDs []uuid.UUID) error {
	if err := r.Q(ctx).DeleteChallengeTopics(ctx, challengeID); err != nil {
		return fmt.Errorf("TopicRepo - SetByChallengeID - DeleteChallengeTopics: %w", err)
	}

	if len(topicIDs) == 0 {
		return nil
	}

	err := r.Q(ctx).InsertChallengeTopics(ctx, sqlc.InsertChallengeTopicsParams{
		ChallengeID: challengeID,
		TopicIds:    topicIDs,
	})
	if err != nil {
		return fmt.Errorf("TopicRepo - SetByChallengeID - InsertChallengeTopics: %w", err)
	}

	return nil
}

func topicFromSQLC(row sqlc.Topic) *domain.Topic {
	return &domain.Topic{
		ID:        row.ID,
		Name:      row.Name,
		CreatedAt: pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(row.CreatedAt)),
	}
}
