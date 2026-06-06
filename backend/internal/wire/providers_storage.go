package wire

import (
	"context"
	"fmt"

	"github.com/wahrwelt-kit/go-logkit"

	"github.com/TakuyaYagam1/AstroCTFb/config"
	"github.com/TakuyaYagam1/AstroCTFb/internal/storage"
)

const storageProviderS3Value = "s3"

func ProvideStorage(ctx context.Context, cfg *config.Config, l logkit.Logger) (storage.Provider, error) {
	if cfg.Provider == storageProviderS3Value {
		s3Provider, err := storage.NewS3Provider(
			cfg.S3Endpoint,
			cfg.S3PublicEndpoint,
			cfg.S3AccessKey,
			cfg.S3SecretKey,
			cfg.S3Bucket,
			cfg.S3Region,
			cfg.S3UseSSL,
		)
		if err != nil {
			return nil, fmt.Errorf("ProvideStorage - NewS3Provider: %w", err)
		}

		if err := s3Provider.EnsureBucket(ctx); err != nil {
			return nil, fmt.Errorf("ProvideStorage - EnsureBucket: %w", err)
		}

		l.Info("Using S3 storage provider", logkit.Fields{"endpoint": cfg.S3Endpoint, "bucket": cfg.S3Bucket})

		return s3Provider, nil
	}

	fsProvider, err := storage.NewFilesystemProvider(cfg.LocalPath)
	if err != nil {
		return nil, fmt.Errorf("ProvideStorage - NewFilesystemProvider: %w", err)
	}

	l.Info("Using filesystem storage provider", logkit.Fields{"path": cfg.LocalPath})

	return fsProvider, nil
}
