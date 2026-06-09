package response

import (
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

func FromStorageList(result *usecase.StorageAdminListResult) openapi.StorageListResponse {
	objects := make([]openapi.StorageObjectResponse, len(result.Paths))
	for i, path := range result.Paths {
		objects[i] = openapi.StorageObjectResponse{Path: &path}
	}

	total := len(objects)

	resp := openapi.StorageListResponse{Objects: &objects, Total: &total}

	if result.NextCursor != "" {
		nextCursor := result.NextCursor
		resp.NextCursor = &nextCursor
	}

	return resp
}
