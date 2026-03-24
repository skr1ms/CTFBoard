package response

import (
	"github.com/samber/lo"
	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func FromUploadedFile(f *domain.File) openapi.UploadFileResponse {
	return openapi.UploadFileResponse{
		ID:       f.ID.String(),
		Filename: f.Filename,
		Size:     f.Size,
		Sha256:   f.SHA256,
	}
}

func FromFile(f *domain.File) openapi.FileItem {
	return openapi.FileItem{
		ID:        httputil.Ptr(f.ID.String()),
		Filename:  httputil.Ptr(f.Filename),
		Size:      httputil.Ptr(int(f.Size)),
		Sha256:    httputil.Ptr(f.SHA256),
		CreatedAt: httputil.Ptr(f.CreatedAt),
	}
}

func FromFileList(files []*domain.File) []openapi.FileItem {
	return lo.Map(files, func(f *domain.File, _ int) openapi.FileItem { return FromFile(f) })
}

func FromFileDownloadURL(url string) openapi.FileDownloadURLResponse {
	return openapi.FileDownloadURLResponse{URL: url}
}
