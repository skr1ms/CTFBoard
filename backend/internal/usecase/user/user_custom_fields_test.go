package user

import (
	"testing"

	"github.com/stretchr/testify/assert"

	validation "github.com/TakuyaYagam1/AstroCTFb/pkg/validator"
)

func TestSanitizeCustomFieldValues(t *testing.T) {
	t.Parallel()
	assert.Nil(t, validation.SanitizeCustomFieldValues(nil))
	assert.Empty(t, validation.SanitizeCustomFieldValues(map[string]string{}))
	out := validation.SanitizeCustomFieldValues(map[string]string{
		"k1": " v1 \x00 ",
		"k2": "v2",
		"k3": "a \x00\x1b b",
		"k4": "\x00\x1f\x7f",
	})
	assert.Equal(t, "v1", out["k1"])
	assert.Equal(t, "v2", out["k2"])
	assert.Equal(t, "a  b", out["k3"])
	assert.Empty(t, out["k4"])
}
