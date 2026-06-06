package backup

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/samber/lo"
	"github.com/wahrwelt-kit/go-logkit"
	"golang.org/x/sync/errgroup"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/crypto"
)

const maxConcurrentFileUploads = 5

// importFilesToStorage concurrently uploads challenge files from the ZIP archive
// to object storage. It builds a lookup map from the expected ZIP path to the
// corresponding File record, then spawns up to maxConcurrentFileUploads workers
// via an errgroup. Each worker uploads one file and sanitizes the storage
// location path before writing. Individual upload failures are collected as
// warning strings and returned; the overall function always returns nil for the
// error so that the caller can treat file upload failures as partial results
// rather than fatal errors.
func (uc *BackupUseCase) importFilesToStorage(ctx context.Context, zr *zip.Reader, files []domain.File, opts domain.ImportOptions) ([]string, error) {
	fileMap := uc.importFilesBuildFileMap(files)
	tasks := uc.importFilesBuildTasks(zr, fileMap)

	var (
		mu       sync.Mutex
		errs     []string
		uploaded int
	)

	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(maxConcurrentFileUploads)

	for _, t := range tasks {
		g.Go(func() error {
			errMsg := uc.importFileUploadOne(gCtx, t.zf, t.file, opts)

			mu.Lock()
			if errMsg != "" {
				errs = append(errs, errMsg)
			} else {
				uploaded++
			}
			mu.Unlock()

			return nil
		})
	}

	_ = g.Wait()

	if len(errs) > 0 {
		uc.deps.Logger.Warn("BackupUseCase - importFilesToStorage - completed with errors", logkit.Fields{
			"total":    len(files),
			"uploaded": uploaded,
			"errors":   len(errs),
		})
	}

	return errs, nil
}

func (uc *BackupUseCase) importFilesBuildFileMap(files []domain.File) map[string]domain.File {
	return lo.Associate(files, func(f domain.File) (string, domain.File) {
		return fmt.Sprintf("files/challenge-%s/%s", f.ChallengeID, f.Filename), f
	})
}

type importFileTask struct {
	zf   *zip.File
	file domain.File
}

// importFilesBuildTasks matches ZIP entries against fileMap (keyed by the
// canonical "files/challenge-<id>/<filename>" path). Symlinks are skipped to
// prevent path-traversal attacks via crafted ZIP archives. Returns only the
// entries that have a corresponding DB file record.
func (uc *BackupUseCase) importFilesBuildTasks(zr *zip.Reader, fileMap map[string]domain.File) []importFileTask {
	var tasks []importFileTask

	for _, zf := range zr.File {
		if zf.FileInfo().Mode()&os.ModeSymlink != 0 {
			continue
		}

		file, ok := fileMap[zf.Name]
		if !ok {
			continue
		}

		tasks = append(tasks, importFileTask{zf: zf, file: file})
	}

	return tasks
}

func (uc *BackupUseCase) importFileUploadOne(ctx context.Context, zf *zip.File, file domain.File, opts domain.ImportOptions) string {
	if err := ctx.Err(); err != nil {
		return fmt.Sprintf("canceled: %s", zf.Name)
	}

	rc, err := zf.Open()
	if err != nil {
		return fmt.Sprintf("open %s: %v", zf.Name, err)
	}
	defer rc.Close()

	size := zipSizeToInt64(zf.UncompressedSize64)
	file.Location = sanitizeFileLocation(file.Location)

	if opts.ValidateFiles {
		return uc.importFileUploadWithHash(ctx, zf.Name, rc, size, file)
	}

	if err := uc.deps.Storage.Upload(ctx, file.Location, rc, size, "application/octet-stream"); err != nil {
		return fmt.Sprintf("upload %s: %v", zf.Name, err)
	}

	return ""
}

// importFileUploadWithHash uploads a single file to object storage while
// simultaneously computing its SHA-256 digest via a TeeReader. Once the upload
// completes, the computed hash is compared against the expected value stored in
// the File record. If the hashes differ the uploaded object is deleted to avoid
// leaving corrupt data in storage, and a warning string describing the mismatch
// is returned. An empty string is returned on success.
func (uc *BackupUseCase) importFileUploadWithHash(ctx context.Context, name string, rc io.Reader, size int64, file domain.File) string {
	hash := sha256.New()

	tee := io.TeeReader(rc, hash)

	err := uc.deps.Storage.Upload(ctx, file.Location, tee, size, "application/octet-stream")
	if err != nil {
		return fmt.Sprintf("upload %s: %v", name, err)
	}

	hashStr := crypto.HashHex(hash)
	if hashStr != file.SHA256 {
		delErr := uc.deps.Storage.Delete(ctx, file.Location)
		if delErr != nil {
			uc.deps.Logger.WithError(delErr).WithFields(logkit.Fields{"location": file.Location}).Warn("BackupUseCase - importFileUploadWithHash - delete after mismatch")
		}

		return fmt.Sprintf("sha256 mismatch for %s: expected %s, got %s", name, file.SHA256, hashStr)
	}

	return ""
}

func zipSizeToInt64(u uint64) int64 {
	if u > math.MaxInt64 {
		return math.MaxInt64
	}

	return int64(u)
}

// sanitizeFileLocation strips path traversal sequences and normalizes a storage
// location path. It prepends a synthetic leading slash so that filepath.Clean
// resolves all ".." components, then converts backslashes to forward slashes and
// removes the synthetic prefix. If the resulting path does not start with
// "files/" (e.g. an absolute or escape path was supplied) the basename is
// placed directly under "files/" to prevent directory traversal attacks when
// writing to object storage.
func sanitizeFileLocation(location string) string {
	cleaned := filepath.ToSlash(filepath.Clean("/" + location))

	cleaned = strings.TrimPrefix(cleaned, "/")
	if !strings.HasPrefix(cleaned, backupFilesPrefix) {
		cleaned = backupFilesPrefix + filepath.Base(cleaned)
	}

	return cleaned
}
