package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type FilesystemProvider struct {
	basePath string
}

func NewFilesystemProvider(basePath string) (*FilesystemProvider, error) {
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}
	return &FilesystemProvider{basePath: basePath}, nil
}

func validatePath(path string) error {
	if filepath.IsAbs(path) {
		return fmt.Errorf("absolute paths not allowed")
	}

	cleaned := filepath.Clean(path)
	if cleaned != path || cleaned == "." || cleaned == ".." {
		return fmt.Errorf("path traversal detected")
	}

	if filepath.Base(path)[0] == '.' {
		return fmt.Errorf("hidden files not allowed")
	}

	return nil
}

func (p *FilesystemProvider) Upload(ctx context.Context, path string, reader io.Reader, size int64, contentType string) error {
	if err := validatePath(path); err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	fullPath := filepath.Join(p.basePath, filepath.Clean(path))
	cleanBase := filepath.Clean(p.basePath)

	if !strings.HasPrefix(fullPath, cleanBase) {
		return fmt.Errorf("path traversal attempt detected")
	}
	if len(fullPath) > len(cleanBase) {
		if !strings.HasSuffix(cleanBase, string(filepath.Separator)) && fullPath[len(cleanBase)] != filepath.Separator {
			return fmt.Errorf("path traversal attempt detected")
		}
	}

	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	file, err := os.Create(fullPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() { _ = file.Close() }()

	if _, err := io.Copy(file, reader); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func (p *FilesystemProvider) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	fullPath := filepath.Join(p.basePath, path)
	file, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", path)
		}
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	return file, nil
}

func (p *FilesystemProvider) Delete(ctx context.Context, path string) error {
	fullPath := filepath.Join(p.basePath, path)
	if err := os.Remove(fullPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to delete file: %w", err)
	}

	dir := filepath.Dir(fullPath)
	_ = os.Remove(dir)

	return nil
}

func (p *FilesystemProvider) GetPresignedURL(ctx context.Context, path string, expiry time.Duration) (string, error) {
	return fmt.Sprintf("/api/v1/files/download/%s", path), nil
}

func GenerateStoragePath(filename string) string {
	safeName := filepath.Base(filename)
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "%d-%s", time.Now().UnixNano(), safeName)
	hash := hex.EncodeToString(h.Sum(nil))[:16]
	return filepath.Join(hash, safeName)
}
