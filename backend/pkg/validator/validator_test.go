package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidatePassword_Success(t *testing.T) {
	t.Parallel()
	assert.True(t, ValidatePassword("ValidPass1!"))
	assert.True(t, ValidatePassword("Abcd1234"))
}

func TestValidatePassword_Error(t *testing.T) {
	t.Parallel()
	assert.False(t, ValidatePassword("short"))
	assert.False(t, ValidatePassword("Abc123"), "7 chars - too short")
	assert.False(t, ValidatePassword("ab"))
	assert.False(t, ValidatePassword(""))
	assert.False(t, ValidatePassword("abc123"), "no uppercase")
	assert.False(t, ValidatePassword("ABC123"), "no lowercase")
	assert.False(t, ValidatePassword("Abcdef"), "no digit")

	long := make([]byte, 130)
	for i := range long {
		long[i] = 'a'
	}

	assert.False(t, ValidatePassword(string(long)))
}

func TestValidateUsername_Success(t *testing.T) {
	t.Parallel()
	assert.True(t, ValidateUsername("user1"))
	assert.True(t, ValidateUsername("user.name"))
}

func TestValidateUsername_Error(t *testing.T) {
	t.Parallel()
	assert.False(t, ValidateUsername(""))
	assert.False(t, ValidateUsername("user@invalid"))
}

func TestValidateEmail_Success(t *testing.T) {
	t.Parallel()
	assert.True(t, ValidateEmail("a@b.co"))
	assert.True(t, ValidateEmail("user@example.com"))
}

func TestValidateEmail_Error(t *testing.T) {
	t.Parallel()
	assert.False(t, ValidateEmail(""))
	assert.False(t, ValidateEmail("invalid"))
	assert.False(t, ValidateEmail("missing@entity"))
}

func TestValidateTeamName_Success(t *testing.T) {
	t.Parallel()
	assert.True(t, ValidateTeamName("Team 1"))
	assert.True(t, ValidateTeamName("a"))
}

func TestValidateTeamName_Error(t *testing.T) {
	t.Parallel()
	assert.False(t, ValidateTeamName(""))
	assert.False(t, ValidateTeamName("tooooooooooooooooooooooooooooooooooooooooooooooooooo"))
}

func TestValidateChallengeTitle_Success(t *testing.T) {
	t.Parallel()
	assert.True(t, ValidateChallengeTitle("Title"))
}

func TestValidateChallengeTitle_Error(t *testing.T) {
	t.Parallel()
	assert.False(t, ValidateChallengeTitle(""))
}

func TestValidateChallengeDescription_Success(t *testing.T) {
	t.Parallel()
	assert.True(t, ValidateChallengeDescription("desc"))
}

func TestValidateChallengeDescription_Error(t *testing.T) {
	t.Parallel()
	assert.False(t, ValidateChallengeDescription(""))
}

func TestValidateChallengeCategory_Success(t *testing.T) {
	t.Parallel()
	assert.True(t, ValidateChallengeCategory("web"))
}

func TestValidateChallengeCategory_Error(t *testing.T) {
	t.Parallel()
	assert.False(t, ValidateChallengeCategory(""))
	assert.False(t, ValidateChallengeCategory("invalid!@#"))
}

func TestValidateChallengeFlag_Success(t *testing.T) {
	t.Parallel()
	assert.True(t, ValidateChallengeFlag("flag{test}"))
}

func TestValidateChallengeFlag_Error(t *testing.T) {
	t.Parallel()
	assert.False(t, ValidateChallengeFlag(""))
}

func TestValidateHintContent_Success(t *testing.T) {
	t.Parallel()
	assert.True(t, ValidateHintContent("hint"))
}

func TestValidateHintContent_Error(t *testing.T) {
	t.Parallel()
	assert.False(t, ValidateHintContent(""))
}

func TestValidateNotEmpty_Success(t *testing.T) {
	t.Parallel()
	assert.True(t, ValidateNotEmpty("x"))
}

func TestValidateNotEmpty_Error(t *testing.T) {
	t.Parallel()
	assert.False(t, ValidateNotEmpty(""))
}

func TestNew_Success(t *testing.T) {
	t.Parallel()

	v, err := New()
	require.NoError(t, err)
	assert.NotNil(t, v)
}

func TestCustomValidator_Validate_Success(t *testing.T) {
	t.Parallel()

	v, err := New()
	require.NoError(t, err)

	type S struct {
		Field string `validate:"not_empty"`
	}

	err = v.Validate(S{Field: "x"})
	assert.NoError(t, err)
}

func TestCustomValidator_Validate_Error(t *testing.T) {
	t.Parallel()

	v, err := New()
	require.NoError(t, err)

	type S struct {
		Field string `validate:"not_empty"`
	}

	err = v.Validate(S{Field: ""})
	assert.Error(t, err)
}

func TestCustomValidator_ValidateVar_Success(t *testing.T) {
	t.Parallel()

	v, err := New()
	require.NoError(t, err)
	err = v.ValidateVar("user1", "custom_username")
	assert.NoError(t, err)
}

func TestCustomValidator_ValidateStruct_PointerFields(t *testing.T) {
	t.Parallel()

	v, err := New()
	require.NoError(t, err)

	username := "scoreboard_ban_a1b2c3d4"

	type S struct {
		Username *string `validate:"required,custom_username"`
	}

	err = v.Validate(S{Username: &username})
	assert.NoError(t, err)
}

func TestCustomValidator_ValidateVar_Error(t *testing.T) {
	t.Parallel()

	v, err := New()
	require.NoError(t, err)
	err = v.ValidateVar("", "custom_username")
	assert.Error(t, err)
}

func TestCustomValidator_ValidateVar_StrongPassword_Success(t *testing.T) {
	t.Parallel()

	v, err := New()
	require.NoError(t, err)
	assert.NoError(t, v.ValidateVar("ValidPass1", "strong_password"))
}

func TestCustomValidator_ValidateVar_StrongPassword_Error(t *testing.T) {
	t.Parallel()

	v, err := New()
	require.NoError(t, err)
	assert.Error(t, v.ValidateVar("short", "strong_password"))
}

func TestValidateCustomFieldEnvelope_Success(t *testing.T) {
	t.Parallel()

	err := ValidateCustomFieldEnvelope(map[string]string{"field_id": "value"})
	assert.NoError(t, err)
}

func TestValidateCustomFieldEnvelope_Error(t *testing.T) {
	t.Parallel()

	longKey := string(make([]byte, MaxCustomFieldKeyLen+1))
	longValue := string(make([]byte, MaxCustomFieldValueLen+1))

	assert.Error(t, ValidateCustomFieldEnvelope(map[string]string{longKey: "value"}))
	assert.Error(t, ValidateCustomFieldEnvelope(map[string]string{"field_id": longValue}))

	fields := make(map[string]string, MaxCustomFields+1)
	for i := range MaxCustomFields + 1 {
		fields[string(rune('a'+i))] = "value"
	}

	assert.Error(t, ValidateCustomFieldEnvelope(fields))
}

func TestSanitizeCustomFieldValues(t *testing.T) {
	t.Parallel()

	assert.Nil(t, SanitizeCustomFieldValues(nil))
	assert.Empty(t, SanitizeCustomFieldValues(map[string]string{}))

	out := SanitizeCustomFieldValues(map[string]string{
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

func TestCustomValidator_ValidateVar_CustomEmail_Success(t *testing.T) {
	t.Parallel()

	v, err := New()
	require.NoError(t, err)
	assert.NoError(t, v.ValidateVar("a@b.co", "custom_email"))
}

func TestCustomValidator_ValidateVar_CustomEmail_Error(t *testing.T) {
	t.Parallel()

	v, err := New()
	require.NoError(t, err)
	assert.Error(t, v.ValidateVar("invalid", "custom_email"))
}

func TestCustomValidator_ValidateVar_TeamName_Success(t *testing.T) {
	t.Parallel()

	v, err := New()
	require.NoError(t, err)
	assert.NoError(t, v.ValidateVar("Team 1", "team_name"))
}

func TestCustomValidator_ValidateVar_TeamName_Error(t *testing.T) {
	t.Parallel()

	v, err := New()
	require.NoError(t, err)
	assert.Error(t, v.ValidateVar("", "team_name"))
}

func TestCustomValidator_ValidateVar_ChallengeTitle_Success(t *testing.T) {
	t.Parallel()

	v, err := New()
	require.NoError(t, err)
	assert.NoError(t, v.ValidateVar("Title", "challenge_title"))
}

func TestCustomValidator_ValidateVar_ChallengeTitle_Error(t *testing.T) {
	t.Parallel()

	v, err := New()
	require.NoError(t, err)
	assert.Error(t, v.ValidateVar("", "challenge_title"))
}

func TestCustomValidator_ValidateVar_ChallengeDescription_Success(t *testing.T) {
	t.Parallel()

	v, err := New()
	require.NoError(t, err)
	assert.NoError(t, v.ValidateVar("desc", "challenge_description"))
}

func TestCustomValidator_ValidateVar_ChallengeDescription_Error(t *testing.T) {
	t.Parallel()

	v, err := New()
	require.NoError(t, err)
	assert.Error(t, v.ValidateVar("", "challenge_description"))
}

func TestCustomValidator_ValidateVar_ChallengeCategory_Success(t *testing.T) {
	t.Parallel()

	v, err := New()
	require.NoError(t, err)
	assert.NoError(t, v.ValidateVar("web", "challenge_category"))
}

func TestCustomValidator_ValidateVar_ChallengeCategory_Error(t *testing.T) {
	t.Parallel()

	v, err := New()
	require.NoError(t, err)
	assert.Error(t, v.ValidateVar("", "challenge_category"))
}

func TestCustomValidator_ValidateVar_ChallengeFlag_Success(t *testing.T) {
	t.Parallel()

	v, err := New()
	require.NoError(t, err)
	assert.NoError(t, v.ValidateVar("flag{test}", "challenge_flag"))
}

func TestCustomValidator_ValidateVar_ChallengeFlag_Error(t *testing.T) {
	t.Parallel()

	v, err := New()
	require.NoError(t, err)
	assert.Error(t, v.ValidateVar("", "challenge_flag"))
}

func TestCustomValidator_ValidateVar_HintContent_Success(t *testing.T) {
	t.Parallel()

	v, err := New()
	require.NoError(t, err)
	assert.NoError(t, v.ValidateVar("hint", "hint_content"))
}

func TestCustomValidator_ValidateVar_HintContent_Error(t *testing.T) {
	t.Parallel()

	v, err := New()
	require.NoError(t, err)
	assert.Error(t, v.ValidateVar("", "hint_content"))
}
