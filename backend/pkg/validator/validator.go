package validator

import (
	"fmt"
	"reflect"
	"regexp"

	"github.com/go-playground/validator/v10"
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
	emailRegex    = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	passwordRegex = regexp.MustCompile(`^[a-zA-Z0-9!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]+$`)
	teamNameRegex = regexp.MustCompile(`^[a-zA-Z0-9\s._\-]+$`)
	categoryRegex = regexp.MustCompile(`^[a-zA-Z0-9\s\-]+$`)
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
	}

	for name, fn := range validations {
		if err := v.RegisterValidation(name, fn); err != nil {
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
//nolint:gocyclo // character class checks
func ValidateStrongPasswordField(fl validator.FieldLevel) bool {
	password := fieldString(fl.Field())
	if len(password) < 6 {
		return false
	}
	if len(password) > 72 {
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

//nolint:gocyclo // character class checks
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

func ValidateUsernameField(fl validator.FieldLevel) bool {
	username := fieldString(fl.Field())
	if username == "" || len(username) > 32 {
		return false
	}
	return usernameRegex.MatchString(username)
}

func ValidateUsername(username string) bool {
	return usernameRegex.MatchString(username)
}

func validateUsername(fl validator.FieldLevel) bool {
	return ValidateUsernameField(fl)
}

func ValidateEmailField(fl validator.FieldLevel) bool {
	email := fieldString(fl.Field())
	if email == "" {
		return false
	}
	return emailRegex.MatchString(email)
}

func ValidateEmail(email string) bool {
	return emailRegex.MatchString(email)
}

func validateEmail(fl validator.FieldLevel) bool {
	return ValidateEmailField(fl)
}

func validateTeamName(fl validator.FieldLevel) bool {
	name := fieldString(fl.Field())
	if len(name) == 0 || len(name) > 50 {
		return false
	}
	return teamNameRegex.MatchString(name)
}

func ValidateTeamName(name string) bool {
	if len(name) == 0 || len(name) > 50 {
		return false
	}
	return teamNameRegex.MatchString(name)
}

func validateChallengeTitle(fl validator.FieldLevel) bool {
	title := fieldString(fl.Field())
	return len(title) > 0 && len(title) <= 100
}

func validateChallengeDescription(fl validator.FieldLevel) bool {
	desc := fieldString(fl.Field())
	return len(desc) > 0 && len(desc) <= 2000
}

func validateChallengeCategory(fl validator.FieldLevel) bool {
	category := fieldString(fl.Field())
	if len(category) == 0 || len(category) > 50 {
		return false
	}
	return categoryRegex.MatchString(category)
}

func validateChallengeFlag(fl validator.FieldLevel) bool {
	flag := fieldString(fl.Field())
	return len(flag) > 0 && len(flag) <= 200
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
	content := fieldString(fl.Field())
	return len(content) > 0 && len(content) <= 500
}

func ValidateHintContent(content string) bool {
	return len(content) > 0 && len(content) <= 500
}

func validateNotEmpty(fl validator.FieldLevel) bool {
	s := fieldString(fl.Field())
	return len(s) > 0
}

func ValidateNotEmpty(s string) bool {
	return len(s) > 0
}
