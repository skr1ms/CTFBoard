package response

import (
	"github.com/samber/lo"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func FromComment(c *domain.Comment) openapi.CommentResponse {
	return openapi.CommentResponse{
		ID:          new(c.ID.String()),
		UserID:      new(c.UserID.String()),
		Username:    new(c.Username),
		ChallengeID: new(c.ChallengeID.String()),
		Content:     new(c.Content),
		CreatedAt:   new(c.CreatedAt),
		UpdatedAt:   new(c.UpdatedAt),
	}
}

func FromCommentList(items []*domain.Comment) []openapi.CommentResponse {
	return lo.Map(items, func(item *domain.Comment, _ int) openapi.CommentResponse { return FromComment(item) })
}
