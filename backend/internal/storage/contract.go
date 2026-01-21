package storage

import (
	"context"
	"io"
	"time"
)

type Provider interface {
	Upload(ctx context.Context, path string, reader io.Reader, size int64, contentType string) error
	Download(ctx context.Context, path string) (io.ReadCloser, error)
	Delete(ctx context.Context, path string) error
	GetPresignedURL(ctx context.Context, path string, expiry time.Duration) (string, error)
}
