package request

import (
	"slices"

	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/validator"
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
	ConnectionInfo    string
	MaxAttempts       int
	Position          int
	State             string
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
	ConnectionInfo    *string
	MaxAttempts       *int
	Position          *int
	State             string
	IsRegex           *bool
	IsCaseInsensitive *bool
	FlagFormatRegex   *string
	TagIDs            []uuid.UUID
}

func validateChallengeNumericParams(points, initialValue, minValue, decay int) error {
	if points < 0 {
		return httperr.NewValidationErrorf("points must be >= 0")
	}

	if initialValue < 0 || minValue < 0 || decay < 0 {
		return httperr.NewValidationErrorf("initial_value, min_value and decay must be >= 0")
	}

	if initialValue < minValue {
		return httperr.NewValidationErrorf("initial_value must be >= min_value")
	}

	return nil
}

var allowedChallengeStates = []string{"visible", "hidden", "locked"}

func validateChallengeState(state string) error {
	if slices.Contains(allowedChallengeStates, state) {
		return nil
	}

	return httperr.NewValidationErrorf("state must be one of: visible, hidden, locked")
}

func CreateChallengeRequestToParams(req *openapi.CreateChallengeRequest) (ChallengeParams, error) {
	tagIDs, err := ParseUUIDSlice(req.TagIds, "tag_id")
	if err != nil {
		return ChallengeParams{}, err
	}

	initialValue := lo.FromPtrOr(req.InitialValue, staticScoringInitialValue)
	minValue := lo.FromPtrOr(req.MinValue, staticScoringMinValue)

	decay := lo.FromPtrOr(req.Decay, staticScoringDecay)
	if err := validateChallengeNumericParams(req.Points, initialValue, minValue, decay); err != nil {
		return ChallengeParams{}, err
	}

	state := challengeStateFromReq(req.State)
	if err := validateChallengeState(state); err != nil {
		return ChallengeParams{}, err
	}

	return ChallengeParams{
		Title:             req.Title,
		Description:       req.Description,
		Category:          req.Category,
		Points:            req.Points,
		Flag:              req.Flag,
		ConnectionInfo:    lo.FromPtrOr(req.ConnectionInfo, ""),
		MaxAttempts:       lo.FromPtrOr(req.MaxAttempts, 0),
		Position:          lo.FromPtrOr(req.Position, 0),
		State:             state,
		FlagFormatRegex:   req.FlagFormatRegex,
		TagIDs:            tagIDs,
		InitialValue:      initialValue,
		MinValue:          minValue,
		Decay:             decay,
		IsRegex:           lo.FromPtrOr(req.IsRegex, false),
		IsCaseInsensitive: lo.FromPtrOr(req.IsCaseInsensitive, false),
	}, nil
}

func challengeStateFromReq(s *openapi.CreateChallengeRequestState) string {
	if s == nil {
		return string(openapi.CreateChallengeRequestStateVisible)
	}

	return string(*s)
}

func updateChallengeStateFromReq(s *openapi.UpdateChallengeRequestState) string {
	if s == nil {
		return ""
	}

	return string(*s)
}

type submitFlagConstraints struct {
	Flag string `validate:"required,max=200"`
}

func ValidateSubmitFlagRequest(req *openapi.SubmitFlagRequest, v validator.Validator) error {
	c := submitFlagConstraints{Flag: req.Flag}

	return ValidateConstraints(v, &c)
}

func SubmitFlagRequestToParams(req *openapi.SubmitFlagRequest) (string, error) {
	return req.Flag, nil
}

func AdminUpsertSolutionRequestToParams(req *openapi.AdminUpsertSolutionRequest) string {
	return req.Content
}

func UpdateChallengeRequestToParams(req *openapi.UpdateChallengeRequest) (UpdateChallengeParams, error) {
	tagIDs, err := ParseUUIDSlice(req.TagIds, "tag_id")
	if err != nil {
		return UpdateChallengeParams{}, err
	}

	iv, mv, dc := req.InitialValue, req.MinValue, req.Decay
	if iv != nil && mv != nil && dc != nil {
		err := validateChallengeNumericParams(req.Points, *iv, *mv, *dc)
		if err != nil {
			return UpdateChallengeParams{}, err
		}
	}

	state := updateChallengeStateFromReq(req.State)
	if state != "" {
		err := validateChallengeState(state)
		if err != nil {
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
		Flag:              lo.FromPtrOr(req.Flag, ""),
		ConnectionInfo:    req.ConnectionInfo,
		MaxAttempts:       req.MaxAttempts,
		Position:          req.Position,
		State:             state,
		IsRegex:           req.IsRegex,
		IsCaseInsensitive: req.IsCaseInsensitive,
	}, nil
}
