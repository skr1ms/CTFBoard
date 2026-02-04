package request

import (
	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/openapi"
)

func CreateChallengeRequestToParams(req *openapi.RequestCreateChallengeRequest) (title, description, category string, points, initialValue, minValue, decay int, flag string, isHidden, isRegex, isCaseInsensitive bool, flagFormatRegex *string, tagIDs []uuid.UUID) {
	initialValue, minValue, decay = 500, 100, 20
	if req.InitialValue != nil {
		initialValue = *req.InitialValue
	}
	if req.MinValue != nil {
		minValue = *req.MinValue
	}
	if req.Decay != nil {
		decay = *req.Decay
	}
	if req.IsHidden != nil {
		isHidden = *req.IsHidden
	}
	if req.IsRegex != nil {
		isRegex = *req.IsRegex
	}
	if req.IsCaseInsensitive != nil {
		isCaseInsensitive = *req.IsCaseInsensitive
	}
	if req.TagIds != nil {
		tagIDs = make([]uuid.UUID, 0, len(*req.TagIds))
		for _, s := range *req.TagIds {
			if id, err := uuid.Parse(s); err == nil {
				tagIDs = append(tagIDs, id)
			}
		}
	}
	return req.Title, req.Description, req.Category, req.Points, initialValue, minValue, decay, req.Flag, isHidden, isRegex, isCaseInsensitive, req.FlagFormatRegex, tagIDs
}

func SubmitFlagRequestToFlag(req *openapi.RequestSubmitFlagRequest) string {
	return req.Flag
}

func UpdateChallengeRequestToParams(req *openapi.RequestUpdateChallengeRequest) (title, description, category string, points, initialValue, minValue, decay int, flag string, isHidden, isRegex, isCaseInsensitive bool, flagFormatRegex *string, tagIDs []uuid.UUID) {
	initialValue, minValue, decay = 500, 100, 20
	if req.InitialValue != nil {
		initialValue = *req.InitialValue
	}
	if req.MinValue != nil {
		minValue = *req.MinValue
	}
	if req.Decay != nil {
		decay = *req.Decay
	}
	if req.Flag != nil {
		flag = *req.Flag
	}
	if req.IsHidden != nil {
		isHidden = *req.IsHidden
	}
	if req.IsRegex != nil {
		isRegex = *req.IsRegex
	}
	if req.IsCaseInsensitive != nil {
		isCaseInsensitive = *req.IsCaseInsensitive
	}
	if req.TagIds != nil {
		tagIDs = make([]uuid.UUID, 0, len(*req.TagIds))
		for _, s := range *req.TagIds {
			if id, err := uuid.Parse(s); err == nil {
				tagIDs = append(tagIDs, id)
			}
		}
	}
	return req.Title, req.Description, req.Category, req.Points, initialValue, minValue, decay, flag, isHidden, isRegex, isCaseInsensitive, req.FlagFormatRegex, tagIDs
}
