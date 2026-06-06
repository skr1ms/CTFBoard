package avatar

import (
	"context"
	"io"
	"time"
)

type AvatarStorage interface {
	Upload(ctx context.Context, path string, reader io.Reader, size int64, contentType string) error
	Delete(ctx context.Context, path string) error
	GetPresignedURL(ctx context.Context, path string, expiry time.Duration) (string, error)
}
