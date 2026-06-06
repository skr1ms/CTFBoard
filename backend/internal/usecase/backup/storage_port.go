package backup

import (
	"context"
	"io"
)

const backupFilesPrefix = "files/"

type BackupStorage interface {
	Upload(ctx context.Context, path string, reader io.Reader, size int64, contentType string) error
	Download(ctx context.Context, path string) (io.ReadCloser, error)
	Delete(ctx context.Context, path string) error
}
