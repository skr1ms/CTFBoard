package integration_test

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/skr1ms/CTFBoard/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStorageS3_Workflow(t *testing.T) {
	endpoint, accessKey, secretKey, bucket := SetupSeaweedFS(t)

	provider, err := storage.NewS3Provider(endpoint, "http://"+endpoint, accessKey, secretKey, bucket, false)
	require.NoError(t, err)

	ctx := context.Background()

	err = provider.EnsureBucket(ctx)
	require.NoError(t, err)

	filename := "integration-test-file.txt"
	content := []byte("hello seaweedfs")
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
