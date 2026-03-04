package response

import "github.com/TakuyaYagam1/AstroCTFb/internal/openapi"

// Message returns an openapi.MessageResponse for success operations.
func Message(msg string) openapi.MessageResponse {
	return openapi.MessageResponse{Message: msg}
}
