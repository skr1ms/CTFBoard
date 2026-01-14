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
)

type CustomValidator struct {
	validator *validator.Validate
}

func New() *CustomValidator {
	v := validator.New()

	_ = v.RegisterValidation("strong_password", validateStrongPassword)
	_ = v.RegisterValidation("custom_email", validateEmail)
	_ = v.RegisterValidation("custom_username", validateUsername)

	return &CustomValidator{validator: v}
}

func (cv *CustomValidator) Validate(i any) error {
	return cv.validator.Struct(i)
}

func (cv *CustomValidator) ValidateVar(field any, tag string) error {
	return cv.validator.Var(field, tag)
}

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

func ValidateEmailField(fl validator.FieldLevel) bool {
	email := fl.Field().String()
	if email == "" {
		return false
	}
	return emailRegex.MatchString(email)
}

func ValidateUsernameField(fl validator.FieldLevel) bool {
	username := fl.Field().String()
	if username == "" || len(username) > 20 {
		return false
	}
	return usernameRegex.MatchString(username)
}

func validateStrongPassword(fl validator.FieldLevel) bool {
	return ValidateStrongPasswordField(fl)
}

func validateEmail(fl validator.FieldLevel) bool {
	return ValidateEmailField(fl)
}

func validateUsername(fl validator.FieldLevel) bool {
	return ValidateUsernameField(fl)
}

func ValidatePassword(password string) bool {
	if len(password) < 6 || len(password) > 128 {
		return false
	}
	return passwordRegex.MatchString(password)
}

func ValidateEmail(email string) bool {
	return emailRegex.MatchString(email)
}

func ValidateUsername(username string) bool {
	return usernameRegex.MatchString(username)
}
