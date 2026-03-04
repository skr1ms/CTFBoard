package helper

import (
	"net/http"

	"github.com/TakuyaYagam1/AstroCTFb/pkg/httputil"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/logger"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/validator"
	"github.com/google/uuid"
)

func DecodeAndValidateE[T any](r *http.Request, v validator.Validator) (T, error) {
	return httputil.DecodeAndValidateE[T](r, v)
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

func GetClientIP(r *http.Request, trustedProxyCIDRs []string) string {
	return httputil.GetClientIP(r, trustedProxyCIDRs)
}

func ParseUUID(w http.ResponseWriter, r *http.Request, id string) (uuid.UUID, bool) {
	return httputil.ParseUUID(w, r, id)
}

func ParseUUIDField(w http.ResponseWriter, r *http.Request, value, field string) (uuid.UUID, bool) {
	return httputil.ParseUUIDField(w, r, value, field)
}
