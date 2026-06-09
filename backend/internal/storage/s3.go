package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
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
	s3DialTimeout            = 5 * time.Second
	s3DialKeepAlive          = 30 * time.Second
	s3TLSHandshakeTimeout    = 5 * time.Second
	s3ResponseHeaderTimeout  = 15 * time.Second
	s3IdleConnTimeout        = 90 * time.Second
	s3ExpectContinueTimeout  = 1 * time.Second
	s3MaxIdleConns           = 100
	s3MaxIdleConnsPerHost    = 10
)

type S3Provider struct {
	client        *minio.Client
	presignClient *minio.Client
	bucket        string
}

var _ Provider = (*S3Provider)(nil)

func NewS3Provider(endpoint, publicEndpoint, accessKey, secretKey, bucket, region string, useSSL bool) (*S3Provider, error) {
	if accessKey == "" || secretKey == "" {
		return nil, errors.New("S3Provider - NewS3Provider: S3 credentials (accessKey, secretKey) are required")
	}

	creds := credentials.NewStaticV4(accessKey, secretKey, "")

	client, err := newS3Client(endpoint, creds, region, useSSL)
	if err != nil {
		return nil, fmt.Errorf("S3Provider - NewS3Provider: %w", err)
	}

	presignClient := client

	if publicEndpoint != "" {
		presignClient, err = newS3PublicClient(publicEndpoint, creds, region)
		if err != nil {
			return nil, fmt.Errorf("S3Provider - NewS3Provider - public endpoint: %w", err)
		}
	}

	return &S3Provider{
		client:        client,
		presignClient: presignClient,
		bucket:        bucket,
	}, nil
}

func newS3Client(endpoint string, creds *credentials.Credentials, region string, useSSL bool) (*minio.Client, error) {
	return minio.New(endpoint, &minio.Options{
		Creds:        creds,
		Secure:       useSSL,
		Region:       region,
		BucketLookup: minio.BucketLookupPath,
		Transport:    newS3HTTPTransport(),
	})
}

func newS3PublicClient(publicEndpoint string, creds *credentials.Credentials, region string) (*minio.Client, error) {
	publicURL, err := url.Parse(publicEndpoint)
	if err != nil {
		return nil, err
	}

	if publicURL.Host == "" || (publicURL.Scheme != "http" && publicURL.Scheme != "https") {
		return nil, fmt.Errorf("invalid public endpoint %q", publicEndpoint)
	}

	if publicURL.Path != "" && publicURL.Path != "/" {
		return nil, fmt.Errorf("invalid public endpoint %q", publicEndpoint)
	}

	if publicURL.RawQuery != "" || publicURL.Fragment != "" {
		return nil, fmt.Errorf("invalid public endpoint %q", publicEndpoint)
	}

	return newS3Client(publicURL.Host, creds, region, publicURL.Scheme == "https")
}

func newS3HTTPTransport() *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   s3DialTimeout,
			KeepAlive: s3DialKeepAlive,
		}).DialContext,
		MaxIdleConns:          s3MaxIdleConns,
		MaxIdleConnsPerHost:   s3MaxIdleConnsPerHost,
		TLSHandshakeTimeout:   s3TLSHandshakeTimeout,
		ResponseHeaderTimeout: s3ResponseHeaderTimeout,
		IdleConnTimeout:       s3IdleConnTimeout,
		ExpectContinueTimeout: s3ExpectContinueTimeout,
	}
}

func (p *S3Provider) EnsureBucket(ctx context.Context) error {
	exists, err := p.client.BucketExists(ctx, p.bucket)
	if err != nil {
		return fmt.Errorf("S3Provider - EnsureBucket - BucketExists: %w", err)
	}

	if !exists {
		err := p.client.MakeBucket(ctx, p.bucket, minio.MakeBucketOptions{})
		if err != nil {
			return fmt.Errorf("S3Provider - EnsureBucket - MakeBucket: %w", err)
		}
	}

	return nil
}

// Upload writes the reader content to S3 at the given path. No retry is
// performed: io.Reader is not seekable, so a failed attempt would leave the
// reader in a partially-consumed state, causing a subsequent retry to upload
// truncated data silently. Callers that need retry semantics should pass an
// io.ReadSeeker and seek to the start before each attempt.
func (p *S3Provider) Upload(ctx context.Context, path string, reader io.Reader, size int64, contentType string) error {
	_, err := p.client.PutObject(ctx, p.bucket, path, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
		UserMetadata: map[string]string{
			"uploaded-by": "ctf-platform",
		},
	})
	if err != nil {
		return fmt.Errorf("S3Provider - Upload: %w", err)
	}

	return nil
}

func (p *S3Provider) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	obj, err := p.client.GetObject(ctx, p.bucket, path, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("S3Provider - Download - GetObject: %w", err)
	}

	if _, err := obj.Stat(); err != nil {
		_ = obj.Close()

		return nil, fmt.Errorf("S3Provider - Download - Stat: %w", err)
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

func (p *S3Provider) List(ctx context.Context, prefix string, limit int) ([]string, error) {
	paths, _, err := p.ListPage(ctx, prefix, "", limit)

	return paths, err
}

func (p *S3Provider) ListPage(ctx context.Context, prefix, cursor string, limit int) ([]string, string, error) {
	if limit <= 0 {
		return nil, "", errors.New("S3Provider - ListPage: limit must be positive")
	}

	opts := minio.ListObjectsOptions{
		Prefix:     prefix,
		Recursive:  true,
		MaxKeys:    limit,
		StartAfter: cursor,
	}

	var paths []string

	for obj := range p.client.ListObjects(ctx, p.bucket, opts) {
		if obj.Err != nil {
			return nil, "", fmt.Errorf("S3Provider - ListPage: %w", obj.Err)
		}

		paths = append(paths, obj.Key)

		if len(paths) >= limit {
			break
		}
	}

	nextCursor := ""

	if len(paths) >= limit {
		nextCursor = paths[len(paths)-1]
	}

	return paths, nextCursor, nil
}

func (p *S3Provider) Ping(ctx context.Context) error {
	_, err := p.client.BucketExists(ctx, p.bucket)
	if err != nil {
		return fmt.Errorf("S3Provider - Ping: %w", err)
	}

	return nil
}

// GetPresignedURL generates a short-lived presigned GET URL for the object at
// path. When publicEndpoint is configured the provider signs against that public
// origin directly; rewriting the host after SigV4 signing would invalidate URLs
// on S3-compatible backends that include Host in the canonical request.
func (p *S3Provider) GetPresignedURL(ctx context.Context, path string, expiry time.Duration) (string, error) {
	var result string

	operation := func() error {
		presignedURL, err := p.presignClient.PresignedGetObject(ctx, p.bucket, path, expiry, nil)
		if err != nil {
			if isS3PermanentError(err) {
				return backoff.Permanent(fmt.Errorf("S3Provider - GetPresignedURL: %w", err))
			}

			return fmt.Errorf("S3Provider - GetPresignedURL: %w", err)
		}

		result = presignedURL.String()

		return nil
	}

	bo := backoff.NewExponentialBackOff()

	bo.InitialInterval = s3BackoffInitialInterval

	err := backoff.Retry(operation, backoff.WithContext(backoff.WithMaxRetries(bo, s3BackoffMaxRetries), ctx))
	if err != nil {
		return "", fmt.Errorf("S3Provider - GetPresignedURL: %w", err)
	}

	return result, nil
}

// isS3PermanentError classifies a MinIO error as non-retryable by inspecting the HTTP status
// via minio.ToErrorResponse. 4xx responses (except 429) indicate client errors that will not
// succeed on retry, so they are wrapped as backoff.Permanent in Upload and GetPresignedURL.
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
