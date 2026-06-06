package validator

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"

	"github.com/TakuyaYagam1/AstroCTFb/pkg/slug"
)

// fieldString extracts the underlying string value from a reflect.Value,
// dereferencing any number of pointer indirections. Returns "" for nil pointers.
func fieldString(f reflect.Value) string {
	for f.Kind() == reflect.Pointer {
		if f.IsNil() {
			return ""
		}

		f = f.Elem()
	}

	return f.String()
}

// Validator is the minimal interface consumed by HTTP handlers for struct and variable validation.
type Validator interface {
	ValidateVar(field any, tag string) error
	Validate(i any) error
}

var (
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+$`)
	// EmailRegex is the compiled regular expression used to validate email address format.
	EmailRegex    = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	passwordRegex = regexp.MustCompile(`^[a-zA-Z0-9!@#$%^&*()_+\-=\[\]{};':"|,.<>/?]+$`)
	teamNameRegex = regexp.MustCompile(`^[a-zA-Z0-9\s._\-]+$`)
	categoryRegex = regexp.MustCompile(`^[a-zA-Z0-9\s\-]+$`)
	hexColorRe    = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
)

// CustomValidator wraps go-playground/validator with project-specific validation rules
// registered as named tags (e.g. "strong_password", "custom_email").
type CustomValidator struct {
	validator *validator.Validate
}

// New constructs a CustomValidator with all project-specific validation rules registered.
func New() (*CustomValidator, error) {
	v := validator.New()

	validations := map[string]validator.Func{
		"strong_password":       validateStrongPassword,
		"custom_email":          validateEmail,
		"custom_username":       validateUsername,
		"team_name":             validateTeamName,
		"challenge_title":       validateChallengeTitle,
		"challenge_description": validateChallengeDescription,
		"challenge_category":    validateChallengeCategory,
		"challenge_flag":        validateChallengeFlag,
		"hint_content":          validateHintContent,
		"not_empty":             validateNotEmpty,
		"page_slug":             validatePageSlug,
		"hex_color":             validateHexColor,
	}

	for name, fn := range validations {
		err := v.RegisterValidation(name, fn)
		if err != nil {
			return nil, fmt.Errorf("validator.New: register %s: %w", name, err)
		}
	}

	return &CustomValidator{validator: v}, nil
}

// Validate validates the fields of a struct according to its registered tags.
func (cv *CustomValidator) Validate(i any) error {
	return cv.validator.Struct(i)
}

// ValidateVar validates a single value against a tag expression (e.g. "required,custom_email").
func (cv *CustomValidator) ValidateVar(field any, tag string) error {
	return cv.validator.Var(field, tag)
}

// ValidateStrongPasswordField is the go-playground/validator field-level adapter for ValidatePassword.
func ValidateStrongPasswordField(fl validator.FieldLevel) bool {
	return ValidatePassword(fieldString(fl.Field()))
}

// ValidatePassword reports whether password satisfies the strength policy
// length 8–72, characters from the allowed set, and at least one lowercase letter,
// one uppercase letter, and one digit.
func ValidatePassword(password string) bool {
	if len(password) < minPasswordLen || len(password) > maxPasswordLen {
		return false
	}

	if !passwordRegex.MatchString(password) {
		return false
	}

	var hasLower, hasUpper, hasDigit bool

	for _, c := range password {
		switch {
		case c >= 'a' && c <= 'z':
			hasLower = true
		case c >= 'A' && c <= 'Z':
			hasUpper = true
		case c >= '0' && c <= '9':
			hasDigit = true
		}
	}

	return hasLower && hasUpper && hasDigit
}

func validateStrongPassword(fl validator.FieldLevel) bool {
	return ValidateStrongPasswordField(fl)
}

const (
	minPasswordLen                = 8
	maxPasswordLen                = 72
	maxUsernameLen                = 50
	maxEmailLen                   = 254
	maxTeamNameLen                = 50
	maxChallengeTitleLen          = 100
	maxChallengeDescriptionLen    = 2000
	maxChallengeCategoryLen       = 50
	maxChallengeFlagLen           = 200
	maxHintContentLen             = 500
	minPersistableCustomFieldRune = 32
	deleteRune                    = 127
)

// ValidateUsernameField is the go-playground/validator field-level adapter for ValidateUsername.
func ValidateUsernameField(fl validator.FieldLevel) bool {
	return ValidateUsername(fieldString(fl.Field()))
}

// ValidateUsername reports whether username is non-empty, at most 50 characters, and contains only
// alphanumeric characters, dots, underscores, percent signs, plus signs, or hyphens.
func ValidateUsername(username string) bool {
	if username == "" || len(username) > maxUsernameLen {
		return false
	}

	return usernameRegex.MatchString(username)
}

func validateUsername(fl validator.FieldLevel) bool {
	return ValidateUsernameField(fl)
}

// ValidateEmailField is the go-playground/validator field-level adapter for ValidateEmail.
func ValidateEmailField(fl validator.FieldLevel) bool {
	return ValidateEmail(fieldString(fl.Field()))
}

// ValidateEmail reports whether email is non-empty, at most 254 characters, and matches EmailRegex.
func ValidateEmail(email string) bool {
	if email == "" || len(email) > maxEmailLen {
		return false
	}

	return EmailRegex.MatchString(email)
}

func validateEmail(fl validator.FieldLevel) bool {
	return ValidateEmailField(fl)
}

func validateTeamName(fl validator.FieldLevel) bool {
	return ValidateTeamName(fieldString(fl.Field()))
}

// ValidateTeamName reports whether name is 1–50 characters containing only alphanumeric characters,
// spaces, dots, underscores, or hyphens.
func ValidateTeamName(name string) bool {
	if len(name) == 0 || len(name) > maxTeamNameLen {
		return false
	}

	return teamNameRegex.MatchString(name)
}

func validateChallengeTitle(fl validator.FieldLevel) bool {
	return ValidateChallengeTitle(fieldString(fl.Field()))
}

func validateChallengeDescription(fl validator.FieldLevel) bool {
	return ValidateChallengeDescription(fieldString(fl.Field()))
}

func validateChallengeCategory(fl validator.FieldLevel) bool {
	return ValidateChallengeCategory(fieldString(fl.Field()))
}

func validateChallengeFlag(fl validator.FieldLevel) bool {
	return ValidateChallengeFlag(fieldString(fl.Field()))
}

// ValidateChallengeTitle reports whether title is 1–100 characters.
func ValidateChallengeTitle(title string) bool {
	return len(title) > 0 && len(title) <= maxChallengeTitleLen
}

// ValidateChallengeDescription reports whether desc is 1–2000 characters.
func ValidateChallengeDescription(desc string) bool {
	return len(desc) > 0 && len(desc) <= maxChallengeDescriptionLen
}

// ValidateChallengeCategory reports whether category is 1–50 characters matching the alphanumeric-and-hyphen pattern.
func ValidateChallengeCategory(category string) bool {
	if len(category) == 0 || len(category) > maxChallengeCategoryLen {
		return false
	}

	return categoryRegex.MatchString(category)
}

// ValidateChallengeFlag reports whether flag is 1–200 characters.
func ValidateChallengeFlag(flag string) bool {
	return len(flag) > 0 && len(flag) <= maxChallengeFlagLen
}

func validateHintContent(fl validator.FieldLevel) bool {
	return ValidateHintContent(fieldString(fl.Field()))
}

// ValidateHintContent reports whether content is 1–500 characters.
func ValidateHintContent(content string) bool {
	return len(content) > 0 && len(content) <= maxHintContentLen
}

func validateNotEmpty(fl validator.FieldLevel) bool {
	return ValidateNotEmpty(fieldString(fl.Field()))
}

// ValidateNotEmpty reports whether s has at least one character.
func ValidateNotEmpty(s string) bool {
	return len(s) > 0
}

func validatePageSlug(fl validator.FieldLevel) bool {
	return slug.MatchPageSlug(fieldString(fl.Field()))
}

func validateHexColor(fl validator.FieldLevel) bool {
	s := fieldString(fl.Field())

	return s == "" || hexColorRe.MatchString(s)
}

const (
	// MaxCustomFields is the maximum number of custom profile fields accepted per request.
	MaxCustomFields = 10
	// MaxCustomFieldKeyLen is the maximum byte length allowed for a custom profile field key.
	MaxCustomFieldKeyLen = 50
	// MaxCustomFieldValueLen is the maximum byte length allowed for a custom profile field value.
	MaxCustomFieldValueLen = 500
)

// ValidateCustomFieldEnvelope checks request-level custom field limits before
// dynamic schema validation resolves field IDs and per-field rules.
func ValidateCustomFieldEnvelope(fields map[string]string) error {
	if len(fields) > MaxCustomFields {
		return fmt.Errorf("validator.ValidateCustomFieldEnvelope: too many custom fields (max %d)", MaxCustomFields)
	}

	for k, v := range fields {
		if len(k) > MaxCustomFieldKeyLen {
			return fmt.Errorf("validator.ValidateCustomFieldEnvelope: custom field key too long: %s", k)
		}

		if len(v) > MaxCustomFieldValueLen {
			return fmt.Errorf("validator.ValidateCustomFieldEnvelope: custom field value too long for %s", k)
		}
	}

	return nil
}

func sanitizeCustomFieldValue(s string) string {
	var b strings.Builder

	for _, r := range s {
		if r >= minPersistableCustomFieldRune && r != deleteRune {
			b.WriteRune(r)
		}
	}

	return strings.TrimSpace(b.String())
}

// SanitizeCustomFieldValues trims values and removes control characters that
// should not be persisted in dynamic profile fields.
func SanitizeCustomFieldValues(fields map[string]string) map[string]string {
	if len(fields) == 0 {
		return fields
	}

	out := make(map[string]string, len(fields))
	for k, v := range fields {
		out[k] = sanitizeCustomFieldValue(v)
	}

	return out
}
