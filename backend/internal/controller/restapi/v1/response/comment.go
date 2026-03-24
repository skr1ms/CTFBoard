package response

import (
	"github.com/samber/lo"
	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func FromComment(c *domain.Comment) openapi.CommentResponse {
	return openapi.CommentResponse{
		ID:          httputil.Ptr(c.ID.String()),
		UserID:      httputil.Ptr(c.UserID.String()),
		ChallengeID: httputil.Ptr(c.ChallengeID.String()),
		Content:     httputil.Ptr(c.Content),
		CreatedAt:   httputil.Ptr(c.CreatedAt),
		UpdatedAt:   httputil.Ptr(c.UpdatedAt),
	}
}

func FromCommentList(items []*domain.Comment) []openapi.CommentResponse {
	return lo.Map(items, func(item *domain.Comment, _ int) openapi.CommentResponse { return FromComment(item) })
}
