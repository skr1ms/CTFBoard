package v1

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/pkg/httputil"
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