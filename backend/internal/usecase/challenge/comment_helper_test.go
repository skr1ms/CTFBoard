package challenge

import (
	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
)

func (h *ChallengeTestHelper) CreateCommentUseCase() *CommentUseCase {
	h.t.Helper()
	return NewCommentUseCase(h.deps.commentRepo, h.deps.challengeRepo)
}

func (h *ChallengeTestHelper) NewComment(userID, challengeID uuid.UUID, content string) *entity.Comment {
	h.t.Helper()
	return &entity.Comment{
		ID:          uuid.New(),
		UserID:      userID,
		ChallengeID: challengeID,
		Content:     content,
	}
}
