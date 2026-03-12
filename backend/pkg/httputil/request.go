package httputil

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/render"
	playvalidator "github.com/go-playground/validator/v10"

	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/logger"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/validator"
)

const MaxRequestBodySize = 1 << 20 // 1 MB

func rejectTrailingJSON(limited io.Reader, dec *json.Decoder) bool {
	buf := make([]byte, 1)
	n, err := dec.Buffered().Read(buf)
	if n > 0 {
		return true
	}
	_ = err
	_, err = limited.Read(buf)
	return err != io.EOF
}

func sanitizeValidationField(field string) string {
	if i := strings.LastIndex(field, "."); i >= 0 && i+1 < len(field) {
		field = field[i+1:]
	}
	return field
}

func sanitizeValidationMessage(e playvalidator.FieldError) string {
	switch e.Tag() {
	case "required", "not_empty":
		return "Required"
	case "email", "custom_email":
		return "Invalid format"
	case "min", "max", "len", "gte", "lte", "gt", "lt":
		return "Invalid value"
	default:
		return "Invalid value"
	}
}

func validationErrorsToItems(valErr playvalidator.ValidationErrors) []ValidationErrorItem {
	items := make([]ValidationErrorItem, len(valErr))
	for i, e := range valErr {
		items[i] = ValidationErrorItem{
			Field:   sanitizeValidationField(e.Field()),
			Message: sanitizeValidationMessage(e),
		}
	}
	return items
}

func DecodeAndValidate[T any](
	w http.ResponseWriter,
	r *http.Request,
	v validator.Validator,
	log logger.Logger,
	operation string,
) (T, bool) {
	var req T
	limited := io.LimitReader(r.Body, MaxRequestBodySize)
	dec := json.NewDecoder(limited)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		log.WithError(err).Error("httputil - " + operation + " - Decode")
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, ErrorResponse{Code: "INVALID_JSON", Message: "invalid JSON format"})
		return req, false
	}
	if rejectTrailingJSON(limited, dec) {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, ErrorResponse{Code: "INVALID_JSON", Message: "invalid JSON format"})
		return req, false
	}

	if err := v.Validate(req); err != nil {
		log.WithError(err).Error("httputil - " + operation + " - Validate")
		render.Status(r, http.StatusBadRequest)
		var valErr playvalidator.ValidationErrors
		if errors.As(err, &valErr) {
			items := validationErrorsToItems(valErr)
			render.JSON(w, r, ValidationErrorResponse{Code: "VALIDATION_ERROR", Message: "Validation failed", Errors: items})
		} else {
			render.JSON(w, r, ErrorResponse{Code: "VALIDATION_ERROR", Message: "Validation failed"})
		}
		return req, false
	}

	return req, true
}

func DecodeAndValidateE[T any](r *http.Request, v validator.Validator) (T, error) {
	var req T
	limited := io.LimitReader(r.Body, MaxRequestBodySize)
	dec := json.NewDecoder(limited)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		return req, &httperr.HTTPError{
			Err:        errors.New("invalid JSON in request body"),
			StatusCode: http.StatusBadRequest,
			Code:       "INVALID_JSON",
		}
	}
	if rejectTrailingJSON(limited, dec) {
		return req, &httperr.HTTPError{
			Err:        errors.New("invalid JSON in request body"),
			StatusCode: http.StatusBadRequest,
			Code:       "INVALID_JSON",
		}
	}
	if err := v.Validate(req); err != nil {
		return req, &httperr.HTTPError{
			Err:        errors.New("validation failed"),
			StatusCode: http.StatusBadRequest,
			Code:       "VALIDATION_ERROR",
		}
	}
	return req, nil
}

func DecodeJSON[T any](r *http.Request, v *T) error {
	limited := io.LimitReader(r.Body, MaxRequestBodySize)
	dec := json.NewDecoder(limited)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	if rejectTrailingJSON(limited, dec) {
		return errors.New("trailing data after JSON")
	}
	return nil
}
