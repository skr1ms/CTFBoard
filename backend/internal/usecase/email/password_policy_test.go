package email

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
)

type staticConfigGetter struct {
	ints map[string]int
}

func (g staticConfigGetter) GetString(_ context.Context, _, defaultVal string) string {
	return defaultVal
}

func (g staticConfigGetter) GetInt(_ context.Context, key string, defaultVal int) int {
	value, ok := g.ints[key]
	if !ok {
		return defaultVal
	}

	return value
}

func TestEmailUseCase_ResetPassword_ConfiguredPasswordMinLength(t *testing.T) {
	t.Parallel()

	uc := NewEmailUseCase(EmailDeps{
		ConfigUC: staticConfigGetter{ints: map[string]int{"password_min_length": 12}},
	})

	err := uc.ResetPassword(context.Background(), "reset-token", "ValidP1!")

	var validationErr *apperr.ValidationError

	require.Error(t, err)
	assert.ErrorAs(t, err, &validationErr)
}
