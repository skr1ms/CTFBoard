package httputil

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/logger"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/validator"
	"github.com/go-chi/render"
)

const MaxRequestBodySize = 1 << 20 // 1 MB

func DecodeAndValidate[T any](
	w http.ResponseWriter,
	r *http.Request,
	v validator.Validator,
	log logger.Logger,
	operation string,
) (T, bool) {
	var req T
	limited := io.LimitReader(r.Body, MaxRequestBodySize)
	if err := json.NewDecoder(limited).Decode(&req); err != nil {
		log.WithError(err).Error("httputil - " + operation + " - Decode")
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, ErrorResponse{Code: "INVALID_JSON", Message: "invalid JSON format"})
		return req, false
	}

	if err := v.Validate(req); err != nil {
		log.WithError(err).Error("httputil - " + operation + " - Validate")
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, ErrorResponse{Code: "VALIDATION_ERROR", Message: "Validation failed"})
		return req, false
	}

	return req, true
}

func DecodeAndValidateE[T any](r *http.Request, v validator.Validator) (T, error) {
	var req T
	limited := io.LimitReader(r.Body, MaxRequestBodySize)
	if err := json.NewDecoder(limited).Decode(&req); err != nil {
		return req, &httperr.HTTPError{
			Err:        err,
			StatusCode: http.StatusBadRequest,
			Code:       "INVALID_JSON",
		}
	}
	if err := v.Validate(req); err != nil {
		return req, &httperr.HTTPError{
			Err:        err,
			StatusCode: http.StatusBadRequest,
			Code:       "VALIDATION_ERROR",
		}
	}
	return req, nil
}

func DecodeJSON[T any](r *http.Request, v *T) error {
	return json.NewDecoder(io.LimitReader(r.Body, MaxRequestBodySize)).Decode(v)
}
