package response

import (
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/openapi"
)

func FromComment(c *entity.Comment) openapi.ResponseCommentResponse {
	return openapi.ResponseCommentResponse{
		ID:          ptr(c.ID.String()),
		UserID:      ptr(c.UserID.String()),
		ChallengeID: ptr(c.ChallengeID.String()),
		Content:     ptr(c.Content),
		CreatedAt:   ptr(c.CreatedAt),
		UpdatedAt:   ptr(c.UpdatedAt),
	}
}

func FromCommentList(items []*entity.Comment) []openapi.ResponseCommentResponse {
	res := make([]openapi.ResponseCommentResponse, len(items))
	for i, item := range items {
		res[i] = FromComment(item)
	}
	return res
}
