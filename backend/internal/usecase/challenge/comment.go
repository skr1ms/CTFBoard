package challenge

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/repo"
	"github.com/skr1ms/CTFBoard/pkg/usecaseutil"
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
		return nil, usecaseutil.Wrap(err, "CommentUseCase - GetByChallengeID")
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
		return nil, usecaseutil.Wrap(err, "CommentUseCase - Create - GetByID challenge")
	}
	comment := &entity.Comment{
		UserID:      userID,
		ChallengeID: challengeID,
		Content:     content,
	}
	if err := uc.commentRepo.Create(ctx, comment); err != nil {
		return nil, usecaseutil.Wrap(err, "CommentUseCase - Create")
	}
	return comment, nil
}

func (uc *CommentUseCase) Delete(ctx context.Context, id, userID uuid.UUID) error {
	c, err := uc.commentRepo.GetByID(ctx, id)
	if err != nil {
		return usecaseutil.Wrap(err, "CommentUseCase - Delete - GetByID")
	}
	if c.UserID != userID {
		return entityError.ErrCommentForbidden
	}
	if err := uc.commentRepo.Delete(ctx, id); err != nil {
		return usecaseutil.Wrap(err, "CommentUseCase - Delete")
	}
	return nil
}
