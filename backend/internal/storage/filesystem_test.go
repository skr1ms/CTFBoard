package storage_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/storage"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/storagepath"
)

func TestFilesystemProvider_Workflow(t *testing.T) { //nolint:tparallel // test mutates filesystem state under one temp root
	t.Parallel()

	tmpDir, err := os.MkdirTemp("", "ctf-platform-storage-test")
	require.NoError(t, err)

	defer func() { _ = os.RemoveAll(tmpDir) }()

	provider, err := storage.NewFilesystemProvider(tmpDir)
	require.NoError(t, err)

	defer func() { _ = provider.Close() }()

	ctx := context.Background()
	filename := "test-file.txt"
	content := []byte("hello world")
	path, err := storagepath.Generate(filename)
	require.NoError(t, err)

	t.Run("Upload", func(t *testing.T) {
		err := provider.Upload(ctx, path, bytes.NewReader(content), int64(len(content)), "text/plain")
		require.NoError(t, err)

		fullPath := filepath.Join(tmpDir, path)
		stat, err := os.Stat(fullPath)
		require.NoError(t, err)
		assert.Equal(t, int64(len(content)), stat.Size())
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
		assert.Contains(t, url, "/api/v1/files/download/")
		assert.Contains(t, url, path)
	})

	t.Run("Delete", func(t *testing.T) {
		err := provider.Delete(ctx, path)
		require.NoError(t, err)

		_, err = provider.Download(ctx, path)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "FilesystemProvider - Download")
	})
}

func TestFilesystemProvider_PathTraversal(t *testing.T) {
	t.Parallel()

	tmpDir, err := os.MkdirTemp("", "ctf-platform-storage-traversal-test")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

	provider, err := storage.NewFilesystemProvider(tmpDir)
	require.NoError(t, err)
	t.Cleanup(func() { _ = provider.Close() })

	ctx := context.Background()
	content := []byte("malicious")
	path := "../escape.txt"

	t.Run("Upload Traversal", func(t *testing.T) {
		t.Parallel()

		err := provider.Upload(ctx, path, bytes.NewReader(content), int64(len(content)), "text/plain")
		assert.Error(t, err)
	})
}

func TestStoragePathGenerate_Success(t *testing.T) {
	t.Parallel()

	path, err := storagepath.Generate("file.txt")
	require.NoError(t, err)
	assert.NotEmpty(t, path)
	assert.Contains(t, path, "file.txt")
}

func TestStoragePathGenerate_SanitizesFilename(t *testing.T) {
	t.Parallel()

	path, err := storagepath.Generate("/etc/passwd")
	require.Error(t, err)
	assert.Empty(t, path)
	assert.ErrorIs(t, err, storagepath.ErrInvalidFilename)
}

func TestStoragePathGenerate_RejectsDotDot(t *testing.T) {
	t.Parallel()

	_, err := storagepath.Generate("..")
	require.Error(t, err)
	assert.ErrorIs(t, err, storagepath.ErrInvalidFilename)
	_, err = storagepath.Generate("a..b")
	require.Error(t, err)
	assert.ErrorIs(t, err, storagepath.ErrInvalidFilename)
}

func TestFilesystemProvider_UploadDownload_WithNestedPath(t *testing.T) {
	t.Parallel()

	tmpDir, err := os.MkdirTemp("", "ctf-platform-storage-nested")
	require.NoError(t, err)

	defer func() { _ = os.RemoveAll(tmpDir) }()

	provider, err := storage.NewFilesystemProvider(tmpDir)
	require.NoError(t, err)

	defer func() { _ = provider.Close() }()

	ctx := context.Background()
	nestedPath := "subdir/nested/file.txt"
	content := []byte("nested content")

	err = provider.Upload(ctx, nestedPath, bytes.NewReader(content), int64(len(content)), "text/plain")
	require.NoError(t, err)

	rc, err := provider.Download(ctx, nestedPath)
	require.NoError(t, err)

	defer func() { _ = rc.Close() }()

	data, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, content, data)

	err = provider.Delete(ctx, nestedPath)
	require.NoError(t, err)
}

func TestFilesystemProvider_CanceledContext(t *testing.T) {
	t.Parallel()

	tmpDir, err := os.MkdirTemp("", "ctf-platform-storage-canceled")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

	provider, err := storage.NewFilesystemProvider(tmpDir)
	require.NoError(t, err)
	t.Cleanup(func() { _ = provider.Close() })

	ctx := context.Background()
	path := "cancel/file.txt"
	content := []byte("content")

	require.NoError(t, provider.Upload(ctx, path, bytes.NewReader(content), int64(len(content)), "text/plain"))

	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	err = provider.Upload(canceledCtx, "cancel/new.txt", bytes.NewReader(content), int64(len(content)), "text/plain")
	assert.ErrorIs(t, err, context.Canceled)

	rc, err := provider.Download(canceledCtx, path)
	require.Nil(t, rc)
	assert.ErrorIs(t, err, context.Canceled)

	_, err = provider.List(canceledCtx, "cancel/", 100)
	assert.ErrorIs(t, err, context.Canceled)

	err = provider.Delete(canceledCtx, path)
	assert.ErrorIs(t, err, context.Canceled)

	err = provider.Ping(canceledCtx)
	assert.ErrorIs(t, err, context.Canceled)

	_, err = provider.GetPresignedURL(canceledCtx, path, time.Hour)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestFilesystemProvider_UploadCanceledDuringCopyRemovesPartialFile(t *testing.T) {
	t.Parallel()

	tmpDir, err := os.MkdirTemp("", "ctf-platform-storage-copy-cancel")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

	provider, err := storage.NewFilesystemProvider(tmpDir)
	require.NoError(t, err)
	t.Cleanup(func() { _ = provider.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	path := "cancel/during-copy.txt"
	reader := cancelingReader{
		cancel: cancel,
		data:   []byte("partial"),
	}

	err = provider.Upload(ctx, path, &reader, int64(len(reader.data)), "text/plain")
	require.ErrorIs(t, err, context.Canceled)

	_, statErr := os.Stat(filepath.Join(tmpDir, path))
	require.True(t, os.IsNotExist(statErr))
}

func TestNewFilesystemProvider_InvalidPath(t *testing.T) {
	t.Parallel()

	tmpFile, err := os.CreateTemp(t.TempDir(), "ctf-platform-file-*")
	require.NoError(t, err)

	tmpPath := tmpFile.Name()
	require.NoError(t, tmpFile.Close())

	defer func() { _ = os.Remove(tmpPath) }()

	_, err = storage.NewFilesystemProvider(tmpPath)
	assert.Error(t, err)
}

func TestFilesystemProvider_Download_NotFound(t *testing.T) {
	t.Parallel()

	tmpDir, err := os.MkdirTemp("", "ctf-platform-storage-download-test")
	require.NoError(t, err)

	defer func() { _ = os.RemoveAll(tmpDir) }()

	provider, err := storage.NewFilesystemProvider(tmpDir)
	require.NoError(t, err)

	defer func() { _ = provider.Close() }()

	_, err = provider.Download(context.Background(), "nonexistent/path.txt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "FilesystemProvider - Download")
}

type cancelingReader struct {
	cancel context.CancelFunc
	data   []byte
	read   bool
}

func (r *cancelingReader) Read(p []byte) (int, error) {
	if r.read {
		return 0, io.EOF
	}

	r.read = true
	n := copy(p, r.data)
	r.cancel()

	return n, nil
}
