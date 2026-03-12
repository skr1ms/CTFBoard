package challenge

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

type CommentUseCase struct {
	deps CommentDeps
}

type CommentDeps struct {
	CommentRepo   repo.CommentRepository
	ChallengeRepo repo.ChallengeRepository
	UserRepo      repo.UserRepository
	TeamRepo      repo.TeamRepository
	TM            repo.TransactionManager
}

var _ usecase.CommentUseCase = (*CommentUseCase)(nil)

func NewCommentUseCase(deps CommentDeps) *CommentUseCase {
	return &CommentUseCase{deps: deps}
}

func (uc *CommentUseCase) GetByChallengeID(ctx context.Context, challengeID uuid.UUID) ([]*entity.Comment, error) {
	challenge, err := uc.deps.ChallengeRepo.GetByID(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("CommentUseCase - GetByChallengeID - ChallengeRepo.GetByID: %w", err)
	}
	if challenge.IsHidden {
		return nil, httperr.ErrChallengeNotFound
	}
	list, err := uc.deps.CommentRepo.GetByChallengeID(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("CommentUseCase - GetByChallengeID - CommentRepo.GetByChallengeID: %w", err)
	}
	return list, nil
}

func (uc *CommentUseCase) Create(ctx context.Context, userID, challengeID uuid.UUID, content string) (*entity.Comment, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, httperr.ErrCommentContentRequired
	}
	if uc.deps.UserRepo != nil {
		user, err := uc.deps.UserRepo.GetByID(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("CommentUseCase - Create - UserRepo.GetByID: %w", err)
		}
		if user.IsBanned {
			return nil, httperr.ErrUserBanned
		}
		if uc.deps.TeamRepo != nil && user.TeamID != nil {
			team, err := uc.deps.TeamRepo.GetByID(ctx, *user.TeamID)
			if err != nil {
				return nil, fmt.Errorf("CommentUseCase - Create - TeamRepo.GetByID: %w", err)
			}
			if team.IsBanned {
				return nil, httperr.ErrTeamBanned
			}
		}
	}
	challenge, err := uc.deps.ChallengeRepo.GetByID(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("CommentUseCase - Create - ChallengeRepo.GetByID: %w", err)
	}
	if challenge.IsHidden {
		return nil, httperr.ErrChallengeNotFound
	}
	comment := &entity.Comment{
		UserID:      userID,
		ChallengeID: challengeID,
		Content:     content,
	}
	if err := uc.deps.CommentRepo.Create(ctx, comment); err != nil {
		return nil, fmt.Errorf("CommentUseCase - Create - CommentRepo.Create: %w", err)
	}
	return comment, nil
}

func (uc *CommentUseCase) Delete(ctx context.Context, ID, userID uuid.UUID, isAdmin bool) error {
	run := func(ctx context.Context) error {
		c, err := uc.deps.CommentRepo.GetByID(ctx, ID)
		if err != nil {
			return fmt.Errorf("CommentUseCase - Delete - CommentRepo.GetByID: %w", err)
		}
		if !isAdmin && c.UserID != userID {
			return httperr.ErrCommentForbidden
		}
		if err := uc.deps.CommentRepo.Delete(ctx, ID); err != nil {
			return fmt.Errorf("CommentUseCase - Delete - CommentRepo.Delete: %w", err)
		}
		return nil
	}
	if uc.deps.TM != nil {
		if err := uc.deps.TM.Run(ctx, run); err != nil {
			return fmt.Errorf("CommentUseCase - Delete - TM.Run: %w", err)
		}
		return nil
	}
	return run(ctx)
}
