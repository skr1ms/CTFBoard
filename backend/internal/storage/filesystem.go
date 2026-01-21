package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

type FilesystemProvider struct {
	basePath string
	root     *os.Root
}

func NewFilesystemProvider(basePath string) (*FilesystemProvider, error) {
	if err := os.MkdirAll(basePath, 0750); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	root, err := os.OpenRoot(basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open root: %w", err)
	}

	return &FilesystemProvider{
		basePath: basePath,
		root:     root,
	}, nil
}

func (p *FilesystemProvider) Upload(ctx context.Context, path string, reader io.Reader, size int64, contentType string) error {
	dir := filepath.Dir(path)
	if dir != "." && dir != "/" {
		if err := p.root.MkdirAll(dir, 0750); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	file, err := p.root.Create(path)
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
	file, err := p.root.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", path)
		}
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	return file, nil
}

func (p *FilesystemProvider) Close() error {
	return p.root.Close()
}

func (p *FilesystemProvider) Delete(ctx context.Context, path string) error {
	if err := p.root.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to delete file: %w", err)
	}

	dir := filepath.Dir(path)
	if dir != "." && dir != "/" {
		_ = p.root.Remove(dir)
	}

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
