package response

import "github.com/TakuyaYagam1/AstroCTFb/internal/openapi"

func FromStorageList(paths []string) openapi.StorageListResponse {
	objects := make([]openapi.StorageObjectResponse, len(paths))
	for i, path := range paths {
		objects[i] = openapi.StorageObjectResponse{Path: &path}
	}

	total := len(objects)

	return openapi.StorageListResponse{Objects: &objects, Total: &total}
}
