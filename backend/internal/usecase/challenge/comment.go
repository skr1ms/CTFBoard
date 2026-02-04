package challenge

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/repo"
)

type CommentUseCase struct {
	commentRepo   repo.CommentRepository
	challengeRepo repo.ChallengeRepository
}

func NewCommentUseCase(
	commentRepo repo.CommentRepository,
	challengeRepo repo.ChallengeRepository,
) *CommentUseCase {
	return &CommentUseCase{
		commentRepo:   commentRepo,
		challengeRepo: challengeRepo,
	}
}

func (uc *CommentUseCase) GetByChallengeID(ctx context.Context, challengeID uuid.UUID) ([]*entity.Comment, error) {
	list, err := uc.commentRepo.GetByChallengeID(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("CommentUseCase - GetByChallengeID: %w", err)
	}
	return list, nil
}

func (uc *CommentUseCase) Create(ctx context.Context, userID, challengeID uuid.UUID, content string) (*entity.Comment, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, fmt.Errorf("CommentUseCase - Create: content is required")
	}
	_, err := uc.challengeRepo.GetByID(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("CommentUseCase - Create - GetByID challenge: %w", err)
	}
	comment := &entity.Comment{
		UserID:      userID,
		ChallengeID: challengeID,
		Content:     content,
	}
	if err := uc.commentRepo.Create(ctx, comment); err != nil {
		return nil, fmt.Errorf("CommentUseCase - Create: %w", err)
	}
	return comment, nil
}

func (uc *CommentUseCase) Delete(ctx context.Context, id, userID uuid.UUID) error {
	c, err := uc.commentRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("CommentUseCase - Delete - GetByID: %w", err)
	}
	if c.UserID != userID {
		return entityError.ErrCommentForbidden
	}
	if err := uc.commentRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("CommentUseCase - Delete: %w", err)
	}
	return nil
}
