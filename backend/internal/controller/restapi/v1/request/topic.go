package request

import (
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func CreateTopicRequestToParams(req *openapi.CreateTopicRequest) (name string, err error) {
	return req.Name, nil
}

func UpdateTopicRequestToParams(req *openapi.UpdateTopicRequest) (name string, err error) {
	return req.Name, nil
}

func SetChallengeTopicsRequestToParams(req *openapi.SetChallengeTopicsRequest) ([]string, error) {
	if req.TopicIds == nil {
		return nil, nil
	}

	return *req.TopicIds, nil
}
