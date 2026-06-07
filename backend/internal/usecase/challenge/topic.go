package challenge

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

type TopicDeps struct {
	TopicRepo     repo.TopicRepository
	ChallengeRepo repo.ChallengeRepository
	TM            repo.TransactionManager
}

type TopicUseCase struct {
	deps TopicDeps
}

var _ usecase.TopicUseCase = (*TopicUseCase)(nil)

func NewTopicUseCase(deps TopicDeps) *TopicUseCase {
	return &TopicUseCase{deps: deps}
}

func (uc *TopicUseCase) Create(ctx context.Context, name string) (*domain.Topic, error) {
	if name == "" {
		return nil, apperr.ErrTopicNameRequired
	}

	topic := &domain.Topic{
		ID:   uuid.New(),
		Name: name,
	}

	err := uc.deps.TopicRepo.Create(ctx, topic)
	if err != nil {
		return nil, fmt.Errorf("TopicUseCase - Create - TopicRepo.Create: %w", err)
	}

	return topic, nil
}

func (uc *TopicUseCase) GetByID(ctx context.Context, ID uuid.UUID) (*domain.Topic, error) {
	topic, err := uc.deps.TopicRepo.GetByID(ctx, ID)
	if err != nil {
		return nil, fmt.Errorf("TopicUseCase - GetByID - TopicRepo.GetByID: %w", err)
	}

	return topic, nil
}

func (uc *TopicUseCase) GetAll(ctx context.Context) ([]*domain.Topic, error) {
	topics, err := uc.deps.TopicRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("TopicUseCase - GetAll - TopicRepo.GetAll: %w", err)
	}

	return topics, nil
}

func (uc *TopicUseCase) Update(ctx context.Context, ID uuid.UUID, name string) (*domain.Topic, error) {
	if name == "" {
		return nil, apperr.ErrTopicNameRequired
	}

	topic, err := uc.deps.TopicRepo.GetByID(ctx, ID)
	if err != nil {
		return nil, fmt.Errorf("TopicUseCase - Update - TopicRepo.GetByID: %w", err)
	}

	topic.Name = name
	if err := uc.deps.TopicRepo.Update(ctx, topic); err != nil {
		return nil, fmt.Errorf("TopicUseCase - Update - TopicRepo.Update: %w", err)
	}

	return topic, nil
}

func (uc *TopicUseCase) Delete(ctx context.Context, ID uuid.UUID) error {
	err := uc.deps.TopicRepo.Delete(ctx, ID)
	if err != nil {
		return fmt.Errorf("TopicUseCase - Delete - TopicRepo.Delete: %w", err)
	}

	return nil
}

func (uc *TopicUseCase) GetByChallengeID(ctx context.Context, challengeID uuid.UUID) ([]*domain.Topic, error) {
	if _, err := uc.deps.ChallengeRepo.GetByID(ctx, challengeID); err != nil {
		return nil, fmt.Errorf("TopicUseCase - GetByChallengeID - ChallengeRepo.GetByID: %w", err)
	}

	topics, err := uc.deps.TopicRepo.GetByChallengeID(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("TopicUseCase - GetByChallengeID - TopicRepo.GetByChallengeID: %w", err)
	}

	return topics, nil
}

func (uc *TopicUseCase) SetByChallengeID(ctx context.Context, challengeID uuid.UUID, topicIDs []uuid.UUID) error {
	topicIDs = domain.UniqueUUIDs(topicIDs)

	return uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		if _, err := uc.deps.ChallengeRepo.GetByID(ctx, challengeID); err != nil {
			return fmt.Errorf("TopicUseCase - SetByChallengeID - ChallengeRepo.GetByID: %w", err)
		}

		if len(topicIDs) > 0 {
			topics, err := uc.deps.TopicRepo.GetByIDs(ctx, topicIDs)
			if err != nil {
				return fmt.Errorf("TopicUseCase - SetByChallengeID - TopicRepo.GetByIDs: %w", err)
			}

			for _, topicID := range topicIDs {
				if _, ok := topics[topicID]; !ok {
					return apperr.NewValidationErrorf("invalid topic_id")
				}
			}
		}

		if err := uc.deps.TopicRepo.SetByChallengeID(ctx, challengeID, topicIDs); err != nil {
			return fmt.Errorf("TopicUseCase - SetByChallengeID - TopicRepo.SetByChallengeID: %w", err)
		}

		return nil
	})
}
