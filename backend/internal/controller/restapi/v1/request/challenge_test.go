package request

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/validator"
)

func TestCreateChallengeRequestToParams_InvalidState_ReturnsError(t *testing.T) {
	t.Parallel()

	invalid := openapi.CreateChallengeRequestState("invalid")
	req := &openapi.CreateChallengeRequest{
		Title:       "Test",
		Description: "Desc",
		Category:    "misc",
		Points:      100,
		Flag:        "CTF{flag}",
		State:       &invalid,
	}
	_, err := CreateChallengeRequestToParams(req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "state")
}

func TestUpdateChallengeRequestToParams_InvalidState_ReturnsError(t *testing.T) {
	t.Parallel()

	invalid := openapi.UpdateChallengeRequestState("invalid")
	req := &openapi.UpdateChallengeRequest{
		Title:       "Test",
		Description: "Desc",
		Category:    "misc",
		Points:      100,
		State:       &invalid,
	}
	_, err := UpdateChallengeRequestToParams(req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "state")
}

func TestCreateChallengeRequestToParams_StateNil_DefaultsToVisible(t *testing.T) {
	t.Parallel()

	req := &openapi.CreateChallengeRequest{
		Title:       "Test",
		Description: "Desc",
		Category:    "misc",
		Points:      100,
		Flag:        "CTF{flag}",
		State:       nil,
	}
	params, err := CreateChallengeRequestToParams(req)
	assert.NoError(t, err)
	assert.Equal(t, "visible", params.State)
}

func TestCreateChallengeRequestToParams_InvalidNumericParams_ReturnsError(t *testing.T) {
	t.Parallel()

	state := openapi.CreateChallengeRequestStateVisible
	req := &openapi.CreateChallengeRequest{
		Title:       "Test",
		Description: "Desc",
		Category:    "misc",
		Points:      -1,
		Flag:        "CTF{flag}",
		State:       &state,
	}
	_, err := CreateChallengeRequestToParams(req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "points")
}

func TestCreateChallengeRequestToParams_InitialValueLessThanMinValue_ReturnsError(t *testing.T) {
	t.Parallel()

	iv, mv := 50, 100
	state := openapi.CreateChallengeRequestStateVisible
	req := &openapi.CreateChallengeRequest{
		Title:        "Test",
		Description:  "Desc",
		Category:     "misc",
		Points:       100,
		Flag:         "CTF{flag}",
		State:        &state,
		InitialValue: &iv,
		MinValue:     &mv,
	}
	_, err := CreateChallengeRequestToParams(req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "initial_value")
}

func TestSubmitFlagRequestToParams_FlagTooLong_ReturnsError(t *testing.T) {
	t.Parallel()

	v, err := validator.New()
	require.NoError(t, err)

	req := &openapi.SubmitFlagRequest{Flag: string(make([]byte, 201))}
	err = v.Validate(req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Flag")
}

func TestCreateChallengeRequestToParams_InvalidTagID_ReturnsError(t *testing.T) {
	t.Parallel()

	badID := "not-a-uuid"
	state := openapi.CreateChallengeRequestStateVisible
	req := &openapi.CreateChallengeRequest{
		Title:       "Test",
		Description: "Desc",
		Category:    "misc",
		Points:      100,
		Flag:        "CTF{flag}",
		State:       &state,
		TagIds:      &[]string{badID},
	}
	_, err := CreateChallengeRequestToParams(req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tag")
}

func TestCreateChallengeRequestToParams_ConnectionInfoMaxAttemptsPositionDefaults(t *testing.T) {
	t.Parallel()

	req := &openapi.CreateChallengeRequest{
		Title:       "Test",
		Description: "Desc",
		Category:    "misc",
		Points:      100,
		Flag:        "CTF{flag}",
		State:       nil,
	}
	params, err := CreateChallengeRequestToParams(req)
	assert.NoError(t, err)
	assert.Empty(t, params.ConnectionInfo)
	assert.Equal(t, 0, params.MaxAttempts)
	assert.Equal(t, 0, params.Position)
}

func TestUpdateChallengeRequestToParams_StateNil_LeavesEmpty(t *testing.T) {
	t.Parallel()

	req := &openapi.UpdateChallengeRequest{
		Title:       "Test",
		Description: "Desc",
		Category:    "misc",
		Points:      100,
		State:       nil,
	}
	params, err := UpdateChallengeRequestToParams(req)
	assert.NoError(t, err)
	assert.Empty(t, params.State)
}
