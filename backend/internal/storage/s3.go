package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type S3Provider struct {
	client         *minio.Client
	bucket         string
	publicEndpoint string
}

func NewS3Provider(endpoint, publicEndpoint, accessKey, secretKey, bucket string, useSSL bool) (*S3Provider, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure:       useSSL,
		Region:       "us-east-1",
		BucketLookup: minio.BucketLookupPath,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
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
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}
	if !exists {
		if err := p.client.MakeBucket(ctx, p.bucket, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
	}
	return nil
}

func (p *S3Provider) Upload(ctx context.Context, path string, reader io.Reader, size int64, contentType string) error {
	_, err := p.client.PutObject(ctx, p.bucket, path, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
		UserMetadata: map[string]string{
			"uploaded-by": "ctfboard",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to upload to S3: %w", err)
	}
	return nil
}

func (p *S3Provider) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	obj, err := p.client.GetObject(ctx, p.bucket, path, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object from S3: %w", err)
	}
	return obj, nil
}

func (p *S3Provider) Delete(ctx context.Context, path string) error {
	err := p.client.RemoveObject(ctx, p.bucket, path, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete from S3: %w", err)
	}
	return nil
}

func (p *S3Provider) GetPresignedURL(ctx context.Context, path string, expiry time.Duration) (string, error) {
	presignedUrl, err := p.client.PresignedGetObject(ctx, p.bucket, path, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	if p.publicEndpoint != "" {
		publicURL, err := url.Parse(p.publicEndpoint)
		if err != nil {
			return "", fmt.Errorf("failed to parse public endpoint: %w", err)
		}

		presignedUrl.Scheme = publicURL.Scheme
		presignedUrl.Host = publicURL.Host
	}

	return presignedUrl.String(), nil
}
