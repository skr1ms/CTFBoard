package storage

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const (
	s3BackoffInitialInterval = 500 * time.Millisecond
	s3BackoffMaxRetries      = 3
)

type S3Provider struct {
	client         *minio.Client
	bucket         string
	publicEndpoint string
}

var _ Provider = (*S3Provider)(nil)

func NewS3Provider(endpoint, publicEndpoint, accessKey, secretKey, bucket, region string, useSSL bool) (*S3Provider, error) {
	if accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("S3Provider - NewS3Provider: S3 credentials (accessKey, secretKey) are required")
	}

	client, err := minio.New(endpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure:       useSSL,
		Region:       region,
		BucketLookup: minio.BucketLookupPath,
	})
	if err != nil {
		return nil, fmt.Errorf("S3Provider - NewS3Provider: %w", err)
	}

	return &S3Provider{
		client:         client,
		bucket:         bucket,
		publicEndpoint: publicEndpoint,
	}, nil
}

func (p *S3Provider) EnsureBucket(ctx context.Context) error {
	exists, err := p.client.BucketExists(ctx, p.bucket)
	if err != nil {
		return fmt.Errorf("S3Provider - EnsureBucket: %w", err)
	}
	if !exists {
		if err := p.client.MakeBucket(ctx, p.bucket, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("S3Provider - EnsureBucket: %w", err)
		}
	}
	return nil
}

func (p *S3Provider) Upload(ctx context.Context, path string, reader io.Reader, size int64, contentType string) error {
	operation := func() error {
		_, err := p.client.PutObject(ctx, p.bucket, path, reader, size, minio.PutObjectOptions{
			ContentType: contentType,
			UserMetadata: map[string]string{
				"uploaded-by": "astroctfb",
			},
		})
		if err != nil {
			if isS3PermanentError(err) {
				return backoff.Permanent(fmt.Errorf("S3Provider - Upload: %w", err))
			}
			return fmt.Errorf("S3Provider - Upload: %w", err)
		}
		return nil
	}

	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = s3BackoffInitialInterval
	if err := backoff.Retry(operation, backoff.WithContext(backoff.WithMaxRetries(bo, s3BackoffMaxRetries), ctx)); err != nil {
		return fmt.Errorf("S3Provider - Upload: %w", err)
	}
	return nil
}

func (p *S3Provider) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	obj, err := p.client.GetObject(ctx, p.bucket, path, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("S3Provider - Download: %w", err)
	}

	if _, err := obj.Stat(); err != nil {
		return nil, fmt.Errorf("S3Provider - Download: %w", err)
	}

	return obj, nil
}

func (p *S3Provider) Delete(ctx context.Context, path string) error {
	err := p.client.RemoveObject(ctx, p.bucket, path, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("S3Provider - Delete: %w", err)
	}
	return nil
}

func (p *S3Provider) List(ctx context.Context, prefix string) ([]string, error) {
	opts := minio.ListObjectsOptions{Prefix: prefix, Recursive: true}
	var paths []string
	for obj := range p.client.ListObjects(ctx, p.bucket, opts) {
		if obj.Err != nil {
			return nil, fmt.Errorf("S3Provider - List: %w", obj.Err)
		}
		paths = append(paths, obj.Key)
	}
	return paths, nil
}

func (p *S3Provider) Ping(ctx context.Context) error {
	_, err := p.client.BucketExists(ctx, p.bucket)
	if err != nil {
		return fmt.Errorf("S3Provider - Ping: %w", err)
	}
	return nil
}

func (p *S3Provider) GetPresignedURL(ctx context.Context, path string, expiry time.Duration) (string, error) {
	var result string

	operation := func() error {
		presignedURL, err := p.client.PresignedGetObject(ctx, p.bucket, path, expiry, nil)
		if err != nil {
			if isS3PermanentError(err) {
				return backoff.Permanent(fmt.Errorf("S3Provider - GetPresignedURL: %w", err))
			}
			return fmt.Errorf("S3Provider - GetPresignedURL: %w", err)
		}

		if p.publicEndpoint != "" {
			publicURL, parseErr := url.Parse(p.publicEndpoint)
			if parseErr != nil {
				return backoff.Permanent(fmt.Errorf("S3Provider - GetPresignedURL: %w", parseErr))
			}
			presignedURL.Scheme = publicURL.Scheme
			presignedURL.Host = publicURL.Host
		}

		result = presignedURL.String()
		return nil
	}

	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = s3BackoffInitialInterval
	if err := backoff.Retry(operation, backoff.WithContext(backoff.WithMaxRetries(bo, s3BackoffMaxRetries), ctx)); err != nil {
		return "", fmt.Errorf("S3Provider - GetPresignedURL: %w", err)
	}
	return result, nil
}

func isS3PermanentError(err error) bool {
	if err == nil {
		return false
	}
	resp := minio.ToErrorResponse(err)
	switch resp.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden,
		http.StatusNotFound, http.StatusBadRequest,
		http.StatusMethodNotAllowed:
		return true
	}
	return false
}
