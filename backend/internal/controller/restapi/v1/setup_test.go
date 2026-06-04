package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func validSetupRequest() setupRequest {
	return setupRequest{
		CTFName:                "Astro CTF",
		Mode:                   "flexible",
		ChallengeVisibility:    "private",
		ScoreVisibility:        "public",
		AccountVisibility:      "public",
		RegistrationVisibility: "public",
		AdminUsername:          "admin",
		AdminEmail:             "admin@example.com",
		AdminPassword:          "password123",
		Timezone:               "UTC",
	}
}

func TestSetupRequestValidate_InvalidVisibility(t *testing.T) {
	t.Parallel()

	req := validSetupRequest()
	req.ScoreVisibility = "garbage"

	assert.Contains(t, req.validate(), "score_visibility")
}

func TestSetupRequestValidate_ValidVisibility(t *testing.T) {
	t.Parallel()

	req := validSetupRequest()

	assert.Empty(t, req.validate())
}

func TestSetupHandlerValidSetupToken(t *testing.T) {
	t.Parallel()

	h := NewSetupHandler(nil, nil, "12345678901234567890123456789012", false, 0)

	assert.True(t, h.validSetupToken("12345678901234567890123456789012"))
	assert.False(t, h.validSetupToken("wrong"))
	assert.False(t, h.validSetupToken(""))
	assert.False(t, NewSetupHandler(nil, nil, "", false, 0).validSetupToken("12345678901234567890123456789012"))
}
