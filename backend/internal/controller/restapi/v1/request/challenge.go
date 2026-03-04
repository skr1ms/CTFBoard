package request

import (
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/google/uuid"
)

const (
	defaultInitialValue = 500
	defaultMinValue     = 100
	defaultDecay        = 20
)

type ChallengeParams struct {
	Title             string
	Description       string
	Category          string
	Points            int
	InitialValue      int
	MinValue          int
	Decay             int
	Flag              string
	IsHidden          bool
	IsRegex           bool
	IsCaseInsensitive bool
	FlagFormatRegex   *string
	TagIDs            []uuid.UUID
}

func parseTagIDs(rawIDs *[]string) ([]uuid.UUID, error) {
	if rawIDs == nil {
		return nil, nil
	}
	ids := make([]uuid.UUID, 0, len(*rawIDs))
	for _, s := range *rawIDs {
		id, err := uuid.Parse(s)
		if err != nil {
			return nil, helper.NewValidationErrorf("invalid tag_id")
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func ParseRequirementIDs(rawIDs *[]string) ([]uuid.UUID, error) {
	if rawIDs == nil {
		return nil, nil
	}
	ids := make([]uuid.UUID, 0, len(*rawIDs))
	for _, s := range *rawIDs {
		id, err := uuid.Parse(s)
		if err != nil {
			return nil, helper.NewValidationErrorf("invalid requirement_id")
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func CreateChallengeRequestToParams(req *openapi.CreateChallengeRequest) (ChallengeParams, error) {
	tagIDs, err := parseTagIDs(req.TagIds)
	if err != nil {
		return ChallengeParams{}, err
	}
	return ChallengeParams{
		Title:             req.Title,
		Description:       req.Description,
		Category:          req.Category,
		Points:            req.Points,
		Flag:              req.Flag,
		FlagFormatRegex:   req.FlagFormatRegex,
		TagIDs:            tagIDs,
		InitialValue:      derefOr(req.InitialValue, defaultInitialValue),
		MinValue:          derefOr(req.MinValue, defaultMinValue),
		Decay:             derefOr(req.Decay, defaultDecay),
		IsHidden:          derefOr(req.IsHidden, false),
		IsRegex:           derefOr(req.IsRegex, false),
		IsCaseInsensitive: derefOr(req.IsCaseInsensitive, false),
	}, nil
}

func SubmitFlagRequestToParams(req *openapi.SubmitFlagRequest) string {
	return req.Flag
}

func AdminUpsertSolutionRequestToParams(req *openapi.AdminUpsertSolutionRequest) string {
	return req.Content
}

func UpdateChallengeRequestToParams(req *openapi.UpdateChallengeRequest) (ChallengeParams, error) {
	tagIDs, err := parseTagIDs(req.TagIds)
	if err != nil {
		return ChallengeParams{}, err
	}
	return ChallengeParams{
		Title:             req.Title,
		Description:       req.Description,
		Category:          req.Category,
		Points:            req.Points,
		FlagFormatRegex:   req.FlagFormatRegex,
		TagIDs:            tagIDs,
		InitialValue:      derefOr(req.InitialValue, defaultInitialValue),
		MinValue:          derefOr(req.MinValue, defaultMinValue),
		Decay:             derefOr(req.Decay, defaultDecay),
		Flag:              derefOr(req.Flag, ""),
		IsHidden:          derefOr(req.IsHidden, false),
		IsRegex:           derefOr(req.IsRegex, false),
		IsCaseInsensitive: derefOr(req.IsCaseInsensitive, false),
	}, nil
}
