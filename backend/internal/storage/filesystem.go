package storage

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultDirMode  = 0o750
	filePathHashLen = 16

	PrefixUsers = "users/"
	PrefixTeams = "teams/"
	PrefixFiles = "files/"
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
		err := p.root.MkdirAll(dir, defaultDirMode)
		if err != nil {
			return fmt.Errorf("FilesystemProvider - Upload - MkdirAll: %w", err)
		}
	}

	file, err := p.root.Create(path)
	if err != nil {
		return fmt.Errorf("FilesystemProvider - Upload - Create: %w", err)
	}

	if _, err := io.Copy(file, reader); err != nil {
		_ = file.Close()
		_ = p.root.Remove(path)

		return fmt.Errorf("FilesystemProvider - Upload - Copy: %w", err)
	}

	if err := file.Close(); err != nil {
		_ = p.root.Remove(path)

		return fmt.Errorf("FilesystemProvider - Upload - Close: %w", err)
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
	err := p.root.Remove(path)
	if err != nil {
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

// List returns all file paths under prefix relative to the storage root.
// Rejects path traversal attempts (prefixes containing ".." or absolute paths)
// before walking. Context cancellation is checked per-entry during the walk so
// long listings are interruptible.
func (p *FilesystemProvider) List(ctx context.Context, prefix string) ([]string, error) {
	cleanPrefix := filepath.Clean(prefix)
	if strings.HasPrefix(cleanPrefix, "..") || filepath.IsAbs(cleanPrefix) {
		return nil, fmt.Errorf("FilesystemProvider - List: invalid prefix %q", prefix)
	}

	fullPath := filepath.Join(p.basePath, cleanPrefix)

	relCheck, err := filepath.Rel(p.basePath, fullPath)
	if err != nil || strings.HasPrefix(relCheck, "..") {
		return nil, errors.New("FilesystemProvider - List: path traversal attempt blocked")
	}

	var paths []string

	err = filepath.WalkDir(fullPath, func(walkPath string, d os.DirEntry, err error) error {
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

		rel, err := filepath.Rel(p.basePath, walkPath)
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
	if strings.HasPrefix(path, PrefixUsers) || strings.HasPrefix(path, PrefixTeams) {
		return fmt.Sprintf("/api/v1/avatars/%s", path), nil
	}

	return fmt.Sprintf("/api/v1/files/download/%s", path), nil
}

var ErrInvalidStorageFilename = errors.New("invalid storage filename")

// GenerateStoragePath builds a collision-resistant storage path for filename.
// filepath.Base strips any directory components to prevent path injection.
// A cryptographically random 8-byte prefix (16 hex chars) is prepended as a
// subdirectory so files from different uploads never collide on the same path.
func GenerateStoragePath(filename string) (string, error) {
	safeName := filepath.Base(filename)
	if safeName == "" || strings.Contains(safeName, "..") {
		return "", ErrInvalidStorageFilename
	}

	var buf [filePathHashLen / 2]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", fmt.Errorf("GenerateStoragePath - crypto/rand: %w", err)
	}

	hash := hex.EncodeToString(buf[:])

	return path.Join("tasks", hash, safeName), nil
}
