package storage_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"
	"time"

	"github.com/skr1ms/CTFBoard/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewS3Provider_EmptyCredentials_Error(t *testing.T) {
	_, err := storage.NewS3Provider("http://localhost:9000", "http://localhost:9000", "", "", "bucket", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "credentials")
}

func TestNewS3Provider_EmptyAccessKey_Error(t *testing.T) {
	_, err := storage.NewS3Provider("http://localhost:9000", "", "", "secret", "bucket", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "credentials")
}

func TestNewS3Provider_EmptySecretKey_Error(t *testing.T) {
	_, err := storage.NewS3Provider("http://localhost:9000", "", "access", "", "bucket", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "credentials")
}

func TestS3Provider_Workflow(t *testing.T) {
	endpoint := os.Getenv("S3_ENDPOINT")
	accessKey := os.Getenv("S3_ACCESS_KEY")
	secretKey := os.Getenv("S3_SECRET_KEY")
	bucket := os.Getenv("S3_BUCKET")

	if endpoint == "" || accessKey == "" || secretKey == "" || bucket == "" {
		t.Skip("Skipping S3 integration test: missing credentials")
	}

	provider, err := storage.NewS3Provider(
		endpoint,
		"http://localhost:9000",
		accessKey,
		secretKey,
		bucket,
		false,
	)
	require.NoError(t, err)

	ctx := context.Background()

	err = provider.EnsureBucket(ctx)
	require.NoError(t, err)

	filename := "test-s3-file.txt"
	content := []byte("hello s3")
	path := storage.GenerateStoragePath(filename)

	t.Run("Upload", func(t *testing.T) {
		err := provider.Upload(ctx, path, bytes.NewReader(content), int64(len(content)), "text/plain")
		require.NoError(t, err)
	})

	t.Run("Download", func(t *testing.T) {
		rc, err := provider.Download(ctx, path)
		require.NoError(t, err)
		defer func() { _ = rc.Close() }()

		data, err := io.ReadAll(rc)
		require.NoError(t, err)
		assert.Equal(t, content, data)
	})

	t.Run("GetPresignedURL", func(t *testing.T) {
		url, err := provider.GetPresignedURL(ctx, path, time.Hour)
		require.NoError(t, err)
		assert.NotEmpty(t, url)
		assert.Contains(t, url, path)
	})

	t.Run("Delete", func(t *testing.T) {
		err := provider.Delete(ctx, path)
		require.NoError(t, err)

		_, err = provider.Download(ctx, path)
		assert.Error(t, err)
	})
}
