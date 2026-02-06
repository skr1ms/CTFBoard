package helper

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/middleware"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/pkg/httputil"
	"github.com/skr1ms/CTFBoard/pkg/logger"
	"github.com/skr1ms/CTFBoard/pkg/validator"
)

func DecodeAndValidateE[T any](r *http.Request, v validator.Validator) (T, error) {
	var req T
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return req, &entityError.HTTPError{
			Err:        err,
			StatusCode: http.StatusBadRequest,
			Code:       "INVALID_REQUEST",
		}
	}
	if err := v.Validate(req); err != nil {
		return req, &entityError.HTTPError{
			Err:        err,
			StatusCode: http.StatusBadRequest,
			Code:       "VALIDATION_ERROR",
		}
	}
	return req, nil
}

func RequireUserE(r *http.Request) (*entity.User, error) {
	user, ok := middleware.GetUser(r.Context())
	if !ok {
		return nil, entityError.ErrNotAuthenticated
	}
	return user, nil
}

func DecodeAndValidate[T any](
	w http.ResponseWriter,
	r *http.Request,
	v validator.Validator,
	log logger.Logger,
	operation string,
) (T, bool) {
	return httputil.DecodeAndValidate[T](w, r, v, log, operation)
}

func DecodeJSON[T any](r *http.Request, v *T) error {
	return httputil.DecodeJSON(r, v)
}

func ParseAuthUserID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	return httputil.ParseAuthUserID(w, r)
}

func GetClientIP(r *http.Request) string {
	return httputil.GetClientIP(r)
}

func ParseUUID(w http.ResponseWriter, r *http.Request, id string) (uuid.UUID, bool) {
	if id == "" {
		RenderInvalidID(w, r)
		return uuid.Nil, false
	}
	parsed, err := uuid.Parse(id)
	if err != nil {
		RenderInvalidID(w, r)
		return uuid.Nil, false
	}
	return parsed, true
}
