package validator

import (
	"regexp"

	"github.com/go-playground/validator/v10"
)

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

func New() *CustomValidator {
	v := validator.New()

	_ = v.RegisterValidation("strong_password", validateStrongPassword)
	_ = v.RegisterValidation("custom_email", validateEmail)
	_ = v.RegisterValidation("custom_username", validateUsername)
	_ = v.RegisterValidation("team_name", validateTeamName)
	_ = v.RegisterValidation("challenge_title", validateChallengeTitle)
	_ = v.RegisterValidation("challenge_description", validateChallengeDescription)
	_ = v.RegisterValidation("challenge_category", validateChallengeCategory)
	_ = v.RegisterValidation("challenge_flag", validateChallengeFlag)
	_ = v.RegisterValidation("hint_content", validateHintContent)
	_ = v.RegisterValidation("not_empty", validateNotEmpty)

	return &CustomValidator{validator: v}
}

func (cv *CustomValidator) Validate(i any) error {
	return cv.validator.Struct(i)
}

func (cv *CustomValidator) ValidateVar(field any, tag string) error {
	return cv.validator.Var(field, tag)
}

// Password validation
func ValidateStrongPasswordField(fl validator.FieldLevel) bool {
	password := fl.Field().String()
	if len(password) < 6 {
		return false
	}
	if len(password) > 128 {
		return false
	}
	return passwordRegex.MatchString(password)
}

func ValidatePassword(password string) bool {
	if len(password) < 6 || len(password) > 128 {
		return false
	}
	return passwordRegex.MatchString(password)
}

func validateStrongPassword(fl validator.FieldLevel) bool {
	return ValidateStrongPasswordField(fl)
}

// Username validation
func ValidateUsernameField(fl validator.FieldLevel) bool {
	username := fl.Field().String()
	if username == "" || len(username) > 20 {
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

// Email validation
func ValidateEmailField(fl validator.FieldLevel) bool {
	email := fl.Field().String()
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

// Team name validation
func validateTeamName(fl validator.FieldLevel) bool {
	name := fl.Field().String()
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

// Challenge validations
func validateChallengeTitle(fl validator.FieldLevel) bool {
	title := fl.Field().String()
	return len(title) > 0 && len(title) <= 100
}

func validateChallengeDescription(fl validator.FieldLevel) bool {
	desc := fl.Field().String()
	return len(desc) > 0 && len(desc) <= 2000
}

func validateChallengeCategory(fl validator.FieldLevel) bool {
	category := fl.Field().String()
	if len(category) == 0 || len(category) > 50 {
		return false
	}
	return categoryRegex.MatchString(category)
}

func validateChallengeFlag(fl validator.FieldLevel) bool {
	flag := fl.Field().String()
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

// Hint validation
func validateHintContent(fl validator.FieldLevel) bool {
	content := fl.Field().String()
	return len(content) > 0 && len(content) <= 500
}

func ValidateHintContent(content string) bool {
	return len(content) > 0 && len(content) <= 500
}

// Generic not empty validation
func validateNotEmpty(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	return len(s) > 0
}

func ValidateNotEmpty(s string) bool {
	return len(s) > 0
}
