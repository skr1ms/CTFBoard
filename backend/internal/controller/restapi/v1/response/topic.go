package response

import (
	"github.com/samber/lo"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func FromTopic(t *domain.Topic) openapi.TopicResponse {
	return openapi.TopicResponse{
		ID:        new(t.ID.String()),
		Name:      new(t.Name),
		CreatedAt: new(t.CreatedAt),
	}
}

func FromTopicList(items []*domain.Topic) []openapi.TopicResponse {
	return lo.Map(items, func(item *domain.Topic, _ int) openapi.TopicResponse { return FromTopic(item) })
}
