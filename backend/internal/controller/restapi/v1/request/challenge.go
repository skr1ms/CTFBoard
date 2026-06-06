package request

import (
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

const (
	staticScoringInitialValue = 0
	staticScoringMinValue     = 0
	staticScoringDecay        = 0
)

type ChallengeParams = usecase.ChallengeCreateParams

type UpdateChallengeParams = usecase.ChallengeUpdateParams

func validateChallengeNumericParams(points, initialValue, minValue, decay int) error {
	if points < 0 {
		return apperr.NewValidationErrorf("points must be >= 0")
	}

	if initialValue < 0 || minValue < 0 || decay < 0 {
		return apperr.NewValidationErrorf("initial_value, min_value and decay must be >= 0")
	}

	if initialValue < minValue {
		return apperr.NewValidationErrorf("initial_value must be >= min_value")
	}

	return nil
}

func invalidChallengeStateError() error {
	return apperr.NewValidationErrorf("state must be one of: visible, hidden, locked")
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

	state, err := challengeStateFromReq(req.State)
	if err != nil {
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
		MaxAttemptsWindow: time.Duration(lo.FromPtrOr(req.MaxAttemptsWindow, 0)) * time.Second,
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

func challengeStateFromReq(s *openapi.CreateChallengeRequestState) (string, error) {
	if s == nil {
		return domain.ChallengeStateVisible, nil
	}

	if !s.Valid() {
		return "", invalidChallengeStateError()
	}

	switch *s {
	case openapi.CreateChallengeRequestStateVisible:
		return domain.ChallengeStateVisible, nil
	case openapi.CreateChallengeRequestStateHidden:
		return domain.ChallengeStateHidden, nil
	case openapi.CreateChallengeRequestStateLocked:
		return domain.ChallengeStateLocked, nil
	default:
		return "", invalidChallengeStateError()
	}
}

func updateChallengeStateFromReq(s *openapi.UpdateChallengeRequestState) (string, error) {
	if s == nil {
		return "", nil
	}

	if !s.Valid() {
		return "", invalidChallengeStateError()
	}

	switch *s {
	case openapi.UpdateChallengeRequestStateVisible:
		return domain.ChallengeStateVisible, nil
	case openapi.UpdateChallengeRequestStateHidden:
		return domain.ChallengeStateHidden, nil
	case openapi.UpdateChallengeRequestStateLocked:
		return domain.ChallengeStateLocked, nil
	default:
		return "", invalidChallengeStateError()
	}
}

func SubmitFlagRequestToParams(req *openapi.SubmitFlagRequest) (string, error) {
	return req.Flag, nil
}

func ChallengeSubmitParams(challengeID uuid.UUID, flag string, userID uuid.UUID, teamID *uuid.UUID, clientIP string) usecase.ChallengeSubmitParams {
	return usecase.ChallengeSubmitParams{
		ChallengeID: challengeID,
		Flag:        flag,
		UserID:      userID,
		TeamID:      teamID,
		ClientIP:    clientIP,
	}
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

	state, err := updateChallengeStateFromReq(req.State)
	if err != nil {
		return UpdateChallengeParams{}, err
	}

	var maxAttemptsWindow *time.Duration

	if req.MaxAttemptsWindow != nil {
		d := time.Duration(*req.MaxAttemptsWindow) * time.Second
		maxAttemptsWindow = &d
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
		MaxAttemptsWindow: maxAttemptsWindow,
		Position:          req.Position,
		State:             state,
		IsRegex:           req.IsRegex,
		IsCaseInsensitive: req.IsCaseInsensitive,
	}, nil
}
