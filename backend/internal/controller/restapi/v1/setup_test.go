package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/validator"
)

func validSetupRequest() openapi.SetupRequest {
	timezone := "UTC"

	return openapi.SetupRequest{
		CtfName:                "Astro CTF",
		Mode:                   openapi.Flexible,
		ChallengeVisibility:    openapi.SetupRequestChallengeVisibilityPrivate,
		ScoreVisibility:        openapi.SetupRequestScoreVisibilityPublic,
		AccountVisibility:      openapi.SetupRequestAccountVisibilityPublic,
		RegistrationVisibility: openapi.SetupRequestRegistrationVisibilityPublic,
		AdminUsername:          "admin",
		AdminEmail:             "admin@example.com",
		AdminPassword:          "Password12345",
		Timezone:               &timezone,
	}
}

func validateSetupRequest(t *testing.T, req openapi.SetupRequest) error {
	t.Helper()

	v, err := validator.New()
	require.NoError(t, err)

	return v.Validate(req)
}

func TestSetupRequestValidation_InvalidVisibility(t *testing.T) {
	t.Parallel()

	req := validSetupRequest()
	req.ScoreVisibility = "garbage"

	assert.Error(t, validateSetupRequest(t, req))
}

func TestSetupRequestValidation_ValidVisibility(t *testing.T) {
	t.Parallel()

	req := validSetupRequest()

	assert.NoError(t, validateSetupRequest(t, req))
}

func TestSetupRequestValidation_ScoreVisibilityAdminsOnly(t *testing.T) {
	t.Parallel()

	req := validSetupRequest()
	req.ScoreVisibility = openapi.SetupRequestScoreVisibilityAdminsOnly

	assert.NoError(t, validateSetupRequest(t, req))
}

func TestSetupRequestValidation_WeakAdminPassword(t *testing.T) {
	t.Parallel()

	cases := []string{
		"password12345",
		"PASSWORD12345",
		"PasswordOnly",
		"Pass123",
	}

	for _, password := range cases {
		req := validSetupRequest()
		req.AdminPassword = password

		assert.Error(t, validateSetupRequest(t, req))
	}
}

func TestSetupHandlerValidSetupToken(t *testing.T) {
	t.Parallel()

	h := NewSetupHandler(nil, nil, nil, "12345678901234567890123456789012", false, 0)

	assert.True(t, h.validSetupToken("12345678901234567890123456789012"))
	assert.False(t, h.validSetupToken("wrong"))
	assert.False(t, h.validSetupToken(""))
	assert.False(t, NewSetupHandler(nil, nil, nil, "", false, 0).validSetupToken("12345678901234567890123456789012"))
}
