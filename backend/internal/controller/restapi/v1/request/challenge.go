package request

import (
	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

const (
	staticScoringInitialValue = 0
	staticScoringMinValue     = 0
	staticScoringDecay        = 0
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

type UpdateChallengeParams struct {
	Title             string
	Description       string
	Category          string
	Points            int
	InitialValue      *int
	MinValue          *int
	Decay             *int
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

func validateChallengeNumericParams(points, initialValue, minValue, decay int) error {
	if points < 0 {
		return helper.NewValidationErrorf("points must be >= 0")
	}
	if initialValue < 0 || minValue < 0 || decay < 0 {
		return helper.NewValidationErrorf("initial_value, min_value and decay must be >= 0")
	}
	if initialValue < minValue {
		return helper.NewValidationErrorf("initial_value must be >= min_value")
	}
	return nil
}

func CreateChallengeRequestToParams(req *openapi.CreateChallengeRequest) (ChallengeParams, error) {
	tagIDs, err := parseTagIDs(req.TagIds)
	if err != nil {
		return ChallengeParams{}, err
	}
	initialValue := derefOr(req.InitialValue, staticScoringInitialValue)
	minValue := derefOr(req.MinValue, staticScoringMinValue)
	decay := derefOr(req.Decay, staticScoringDecay)
	if err := validateChallengeNumericParams(req.Points, initialValue, minValue, decay); err != nil {
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
		InitialValue:      initialValue,
		MinValue:          minValue,
		Decay:             decay,
		IsHidden:          derefOr(req.IsHidden, false),
		IsRegex:           derefOr(req.IsRegex, false),
		IsCaseInsensitive: derefOr(req.IsCaseInsensitive, false),
	}, nil
}

const maxSubmittedFlagLen = 200

func SubmitFlagRequestToParams(req *openapi.SubmitFlagRequest) (string, error) {
	if len(req.Flag) > maxSubmittedFlagLen {
		return "", helper.NewValidationErrorf("flag too long")
	}
	return req.Flag, nil
}

func AdminUpsertSolutionRequestToParams(req *openapi.AdminUpsertSolutionRequest) string {
	return req.Content
}

func UpdateChallengeRequestToParams(req *openapi.UpdateChallengeRequest) (UpdateChallengeParams, error) {
	tagIDs, err := parseTagIDs(req.TagIds)
	if err != nil {
		return UpdateChallengeParams{}, err
	}
	iv, mv, dc := req.InitialValue, req.MinValue, req.Decay
	if iv != nil || mv != nil || dc != nil {
		effectiveIV := staticScoringInitialValue
		if iv != nil {
			effectiveIV = *iv
		}
		effectiveMV := staticScoringMinValue
		if mv != nil {
			effectiveMV = *mv
		}
		effectiveDC := staticScoringDecay
		if dc != nil {
			effectiveDC = *dc
		}
		if err := validateChallengeNumericParams(req.Points, effectiveIV, effectiveMV, effectiveDC); err != nil {
			return UpdateChallengeParams{}, err
		}
	}
	return UpdateChallengeParams{
		Title:             req.Title,
		Description:       req.Description,
		Category:          req.Category,
		Points:            req.Points,
		InitialValue:      req.InitialValue,
		MinValue:          req.MinValue,
		Decay:             req.Decay,
		FlagFormatRegex:   req.FlagFormatRegex,
		TagIDs:            tagIDs,
		Flag:              derefOr(req.Flag, ""),
		IsHidden:          derefOr(req.IsHidden, false),
		IsRegex:           derefOr(req.IsRegex, false),
		IsCaseInsensitive: derefOr(req.IsCaseInsensitive, false),
	}, nil
}
