package response

import "github.com/TakuyaYagam1/AstroCTFb/internal/openapi"

func FromAvatarUpload(fullURL, thumbURL string) openapi.AvatarUploadResponse {
	return openapi.AvatarUploadResponse{
		FullURL:  fullURL,
		ThumbURL: thumbURL,
	}
}
