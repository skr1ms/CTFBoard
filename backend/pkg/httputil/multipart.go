package httputil

import (
	"io"
	"mime/multipart"
	"net/http"

	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

func ParseMultipartFormLimit(w http.ResponseWriter, r *http.Request, maxMemory int64) bool {
	if err := r.ParseMultipartForm(maxMemory); err != nil {
		HandleError(w, r, httperr.NewValidationErrorf("failed to parse form"))
		return false
	}
	return true
}

func ReadFormFile(w http.ResponseWriter, r *http.Request, key string, maxSize int64) ([]byte, *multipart.FileHeader, bool) {
	file, header, err := r.FormFile(key)
	if err != nil {
		HandleError(w, r, httperr.NewValidationErrorf("%s is required", key))
		return nil, nil, false
	}
	defer file.Close()

	limited := io.LimitReader(file, maxSize+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		HandleError(w, r, httperr.NewValidationErrorf("failed to read file"))
		return nil, nil, false
	}
	if int64(len(data)) > maxSize {
		HandleError(w, r, httperr.NewValidationErrorf("file exceeds maximum allowed size"))
		return nil, nil, false
	}
	return data, header, true
}
