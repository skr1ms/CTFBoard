package storage

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/pkg/crypto"
)

const (
	defaultDirMode = 0o750
	hashPrefixLen  = 16
)

var _ Provider = (*FilesystemProvider)(nil)

type FilesystemProvider struct {
	basePath string
	root     *os.Root
}

func NewFilesystemProvider(basePath string) (*FilesystemProvider, error) {
	if err := os.MkdirAll(basePath, defaultDirMode); err != nil {
		return nil, fmt.Errorf("FilesystemProvider - NewFilesystemProvider: %w", err)
	}

	root, err := os.OpenRoot(basePath)
	if err != nil {
		return nil, fmt.Errorf("FilesystemProvider - NewFilesystemProvider: %w", err)
	}

	return &FilesystemProvider{
		basePath: basePath,
		root:     root,
	}, nil
}

func (p *FilesystemProvider) Upload(_ context.Context, path string, reader io.Reader, _ int64, _ string) error {
	dir := filepath.Dir(path)
	if dir != "." && dir != "/" {
		if err := p.root.MkdirAll(dir, defaultDirMode); err != nil {
			return fmt.Errorf("FilesystemProvider - Upload: %w", err)
		}
	}

	file, err := p.root.Create(path)
	if err != nil {
		return fmt.Errorf("FilesystemProvider - Upload: %w", err)
	}
	defer func() { _ = file.Close() }()

	if _, err := io.Copy(file, reader); err != nil {
		return fmt.Errorf("FilesystemProvider - Upload: %w", err)
	}

	return nil
}

func (p *FilesystemProvider) Download(_ context.Context, path string) (io.ReadCloser, error) {
	file, err := p.root.Open(path)
	if err != nil {
		return nil, fmt.Errorf("FilesystemProvider - Download: %w", err)
	}
	return file, nil
}

func (p *FilesystemProvider) Close() error {
	return p.root.Close()
}

func (p *FilesystemProvider) Delete(_ context.Context, path string) error {
	if err := p.root.Remove(path); err != nil {
		return fmt.Errorf("FilesystemProvider - Delete: %w", err)
	}

	dir := filepath.Dir(path)
	if dir != "." && dir != "/" {
		_ = p.root.Remove(dir) //nolint:errcheck // best-effort: fails if dir still contains other files
	}

	return nil
}

func (p *FilesystemProvider) Ping(_ context.Context) error {
	_, err := os.Stat(p.basePath)
	if err != nil {
		return fmt.Errorf("FilesystemProvider - Ping: %w", err)
	}
	return nil
}

func (p *FilesystemProvider) List(ctx context.Context, prefix string) ([]string, error) {
	dir := filepath.Join(p.basePath, prefix)
	var paths []string
	err := filepath.WalkDir(dir, func(fullPath string, d os.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(p.basePath, fullPath)
		if err != nil {
			return err
		}
		paths = append(paths, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("FilesystemProvider - List: %w", err)
	}
	return paths, nil
}

func (p *FilesystemProvider) GetPresignedURL(_ context.Context, path string, _ time.Duration) (string, error) {
	return fmt.Sprintf("/api/v1/files/download/%s", path), nil
}

var ErrInvalidStorageFilename = errors.New("invalid storage filename")

func GenerateStoragePath(filename string) (string, error) {
	safeName := filepath.Base(filename)
	if safeName == "" || strings.Contains(safeName, "..") {
		return "", ErrInvalidStorageFilename
	}
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "%d-%s", time.Now().UnixNano(), safeName)
	hash := crypto.HashHex(h)[:hashPrefixLen]
	return filepath.Join(hash, safeName), nil
}
