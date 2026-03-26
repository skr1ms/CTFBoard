package validator

import (
	"fmt"
	"reflect"
	"regexp"

	"github.com/go-playground/validator/v10"

	"github.com/TakuyaYagam1/AstroCTFb/pkg/slug"
)

func fieldString(f reflect.Value) string {
	for f.Kind() == reflect.Ptr {
		if f.IsNil() {
			return ""
		}

		f = f.Elem()
	}

	return f.String()
}

type Validator interface {
	ValidateVar(field any, tag string) error
	Validate(i any) error
}

var (
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+$`)
	EmailRegex    = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	passwordRegex = regexp.MustCompile(`^[a-zA-Z0-9!@#$%^&*()_+\-=\[\]{};':"|,.<>/?]+$`)
	teamNameRegex = regexp.MustCompile(`^[a-zA-Z0-9\s._\-]+$`)
	categoryRegex = regexp.MustCompile(`^[a-zA-Z0-9\s\-]+$`)
	hexColorRe    = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
)

type CustomValidator struct {
	validator *validator.Validate
}

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
			return nil, fmt.Errorf("register validation %s: %w", name, err)
		}
	}

	return &CustomValidator{validator: v}, nil
}

func (cv *CustomValidator) Validate(i any) error {
	return cv.validator.Struct(i)
}

func (cv *CustomValidator) ValidateVar(field any, tag string) error {
	return cv.validator.Var(field, tag)
}

// Password validation: length 6–72, allowlist, and at least one lowercase, one uppercase, one digit.
//

func ValidateStrongPasswordField(fl validator.FieldLevel) bool {
	return ValidatePassword(fieldString(fl.Field()))
}

func ValidatePassword(password string) bool {
	if len(password) < 6 || len(password) > 72 {
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
	maxUsernameLen = 50
	maxEmailLen    = 254
)

func ValidateUsernameField(fl validator.FieldLevel) bool {
	return ValidateUsername(fieldString(fl.Field()))
}

func ValidateUsername(username string) bool {
	if username == "" || len(username) > maxUsernameLen {
		return false
	}

	return usernameRegex.MatchString(username)
}

func validateUsername(fl validator.FieldLevel) bool {
	return ValidateUsernameField(fl)
}

func ValidateEmailField(fl validator.FieldLevel) bool {
	return ValidateEmail(fieldString(fl.Field()))
}

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

func ValidateTeamName(name string) bool {
	if len(name) == 0 || len(name) > 50 {
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

func ValidateChallengeTitle(title string) bool {
	return len(title) > 0 && len(title) <= 100
}

func ValidateChallengeDescription(desc string) bool {
	return len(desc) > 0 && len(desc) <= 2000
}

func ValidateChallengeCategory(category string) bool {
	if len(category) == 0 || len(category) > 50 {
		return false
	}

	return categoryRegex.MatchString(category)
}

func ValidateChallengeFlag(flag string) bool {
	return len(flag) > 0 && len(flag) <= 200
}

func validateHintContent(fl validator.FieldLevel) bool {
	return ValidateHintContent(fieldString(fl.Field()))
}

func ValidateHintContent(content string) bool {
	return len(content) > 0 && len(content) <= 500
}

func validateNotEmpty(fl validator.FieldLevel) bool {
	return ValidateNotEmpty(fieldString(fl.Field()))
}

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
	MaxCustomFieldKeyLen   = 50
	MaxCustomFieldValueLen = 500
)

var AllowedCustomFields = map[string]bool{
	"affiliation": true,
	"country":     true,
	"discord":     true,
	"telegram":    true,
	"twitter":     true,
	"github":      true,
}

func ValidateCustomFields(fields map[string]string) error {
	if fields == nil {
		return nil
	}

	for k, v := range fields {
		if !AllowedCustomFields[k] {
			return fmt.Errorf("invalid custom field: %s", k)
		}

		if len(k) > MaxCustomFieldKeyLen {
			return fmt.Errorf("custom field key too long: %s", k)
		}

		if len(v) > MaxCustomFieldValueLen {
			return fmt.Errorf("custom field value too long for %s", k)
		}
	}

	return nil
}
