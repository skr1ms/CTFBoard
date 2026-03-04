package challenge

import (
	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/google/uuid"
)

func (h *ChallengeTestHelper) CreateCommentUseCase() *CommentUseCase {
	h.t.Helper()
	return NewCommentUseCase(CommentDeps{CommentRepo: h.deps.commentRepo, ChallengeRepo: h.deps.challengeRepo})
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
