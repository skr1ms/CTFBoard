package response

import (
	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func FromUploadedFile(f *entity.File) openapi.UploadFileResponse {
	return openapi.UploadFileResponse{
		ID:       f.ID.String(),
		Filename: f.Filename,
		Size:     f.Size,
		Sha256:   f.SHA256,
	}
}

func FromFile(f *entity.File) openapi.FileItem {
	return openapi.FileItem{
		ID:        ptr(f.ID.String()),
		Filename:  ptr(f.Filename),
		Size:      ptr(int(f.Size)),
		Sha256:    ptr(f.SHA256),
		CreatedAt: ptr(f.CreatedAt),
	}
}

func FromFileList(files []*entity.File) []openapi.FileItem {
	res := make([]openapi.FileItem, len(files))
	for i, f := range files {
		res[i] = FromFile(f)
	}
	return res
}

func FromFileDownloadURL(url string) openapi.FileDownloadURLResponse {
	return openapi.FileDownloadURLResponse{URL: url}
}
