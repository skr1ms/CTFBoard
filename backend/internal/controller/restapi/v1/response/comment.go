package response

import (
	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func FromComment(c *entity.Comment) openapi.CommentResponse {
	return openapi.CommentResponse{
		ID:          ptr(c.ID.String()),
		UserID:      ptr(c.UserID.String()),
		ChallengeID: ptr(c.ChallengeID.String()),
		Content:     ptr(c.Content),
		CreatedAt:   ptr(c.CreatedAt),
		UpdatedAt:   ptr(c.UpdatedAt),
	}
}

func FromCommentList(items []*entity.Comment) []openapi.CommentResponse {
	res := make([]openapi.CommentResponse, len(items))
	for i, item := range items {
		res[i] = FromComment(item)
	}
	return res
}
