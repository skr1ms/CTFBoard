package httputil

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/render"
	"github.com/skr1ms/CTFBoard/pkg/logger"
	"github.com/skr1ms/CTFBoard/pkg/validator"
)

func DecodeAndValidate[T any](
	w http.ResponseWriter,
	r *http.Request,
	v validator.Validator,
	log logger.Logger,
	operation string,
) (T, bool) {
	var req T

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.WithError(err).Error("httputil - " + operation + " - Decode")
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{
			"error": "invalid JSON format",
			"code":  "INVALID_JSON",
		})
		return req, false
	}

	if err := v.Validate(req); err != nil {
		log.WithError(err).Error("httputil - " + operation + " - Validate")
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{
			"error": err.Error(),
			"code":  "VALIDATION_ERROR",
		})
		return req, false
	}

	return req, true
}
