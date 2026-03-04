package helper

import (
	"net/http"

	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httputil"
)

type ErrorResponse = openapi.ErrorResponse

func HandleError(w http.ResponseWriter, r *http.Request, err error) {
	httputil.HandleError(w, r, err)
}
