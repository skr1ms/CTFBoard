package backup

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/wahrwelt-kit/go-logkit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

type exportZIPReadCloser struct {
	reader *io.PipeReader
	cancel context.CancelFunc
	done   <-chan struct{}
}

// ExportZIP returns a streaming ReadCloser that delivers the backup as a ZIP
// archive. It creates an io.Pipe and launches exportZIPWorker in a background
// goroutine that writes into the pipe's write end; the caller reads from the
// read end. Errors produced by the worker are propagated through the pipe so
// that the reader receives them as read errors. The caller must close the
// returned ReadCloser to release resources.
func (uc *BackupUseCase) ExportZIP(ctx context.Context, opts domain.ExportOptions) (io.ReadCloser, error) {
	pr, pw := io.Pipe()
	workerCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})

	go func() {
		defer close(done)

		uc.exportZIPWorker(workerCtx, pw, opts)
	}()

	return &exportZIPReadCloser{
		reader: pr,
		cancel: cancel,
		done:   done,
	}, nil
}

func (r *exportZIPReadCloser) Read(p []byte) (int, error) {
	return r.reader.Read(p)
}

func (r *exportZIPReadCloser) Close() error {
	r.cancel()

	err := r.reader.Close()
	select {
	case <-r.done:
		return err
	case <-time.After(exportZIPCloseTimeout):
		return errors.Join(err, context.DeadlineExceeded)
	}
}

// exportZIPWorker is the background goroutine that drives ZIP archive creation
// It calls Export to fetch all data, writes backup.json and README.md into the
// archive, and then streams challenge files via streamFilesToZip when
// IncludeFiles is set. Any error at any step is forwarded to the pipe writer via
// CloseWithError so that the reader on the other end of the io.Pipe receives it
// as an I/O error. The pipe is always closed (with or without an error) before
// the goroutine returns.
func (uc *BackupUseCase) exportZIPWorker(ctx context.Context, pw *io.PipeWriter, opts domain.ExportOptions) {
	defer pw.Close()

	select {
	case <-ctx.Done():
		pw.CloseWithError(ctx.Err())

		return
	default:
	}

	zw := zip.NewWriter(pw)
	defer zw.Close()

	data, err := uc.Export(ctx, opts)
	if err != nil {
		pw.CloseWithError(err)

		return
	}

	if ctx.Err() != nil {
		pw.CloseWithError(ctx.Err())

		return
	}

	if err := uc.writeBackupJSON(zw, data); err != nil {
		pw.CloseWithError(err)

		return
	}

	if opts.IncludeFiles && len(data.Files) > 0 {
		if ctx.Err() != nil {
			pw.CloseWithError(ctx.Err())

			return
		}

		skipped := uc.streamFilesToZip(ctx, zw, data.Files)
		uc.deps.Logger.Info("BackupUseCase - ExportZIP - completed", logkit.Fields{
			"challenges": len(data.Challenges),
			"teams":      len(data.Teams),
			"files":      len(data.Files),
			"skipped":    skipped,
		})
	} else {
		uc.deps.Logger.Info("BackupUseCase - ExportZIP - completed", logkit.Fields{
			"challenges": len(data.Challenges),
			"teams":      len(data.Teams),
			"files":      0,
		})
	}
}

func (uc *BackupUseCase) writeBackupJSON(zw *zip.Writer, data *domain.BackupData) error {
	jsonFile, err := zw.Create("backup.json")
	if err != nil {
		return fmt.Errorf("BackupUseCase - ExportZIP - create backup.json: %w", err)
	}

	if err := json.NewEncoder(jsonFile).Encode(data); err != nil {
		return fmt.Errorf("BackupUseCase - ExportZIP - encode backup.json: %w", err)
	}

	readme, err := zw.Create("README.md")
	if err != nil {
		return fmt.Errorf("BackupUseCase - ExportZIP - create README.md: %w", err)
	}

	if _, err := fmt.Fprintf(readme, "# CTF Platform Backup\n\ncreated: %s\nVersion: %s", data.ExportedAt.Format(time.RFC3339), data.Version); err != nil {
		return fmt.Errorf("BackupUseCase - ExportZIP - write README: %w", err)
	}

	return nil
}

// streamFilesToZip downloads each file from object storage and writes it into
// the ZIP archive under the path "files/challenge-<uuid>/<basename>". Files are
// processed sequentially; when a download or copy fails the file is skipped, a
// warning is logged, and the skip counter is incremented. Context cancellation
// stops the loop early. The total number of skipped files is returned so the
// caller can surface it in the completion log entry.
func (uc *BackupUseCase) streamFilesToZip(ctx context.Context, zw *zip.Writer, files []domain.File) int {
	var skipped int

	for _, file := range files {
		if ctx.Err() != nil {
			break
		}

		path := fmt.Sprintf("files/challenge-%s/%s", file.ChallengeID, filepath.Base(file.Filename))

		f, err := zw.Create(path)
		if err != nil {
			uc.deps.Logger.WithError(err).WithFields(logkit.Fields{"file": file.Filename}).Warn("BackupUseCase - streamFilesToZip - create")

			skipped++

			continue
		}

		rc, err := uc.deps.Storage.Download(ctx, file.Location)
		if err != nil {
			uc.deps.Logger.WithError(err).WithFields(logkit.Fields{"file": file.Filename, "location": file.Location}).Warn("BackupUseCase - streamFilesToZip - download")

			skipped++

			continue
		}

		func() {
			defer func() { _ = rc.Close() }()

			if _, err := io.Copy(f, rc); err != nil {
				uc.deps.Logger.WithError(err).WithFields(logkit.Fields{"file": file.Filename}).Warn("BackupUseCase - streamFilesToZip - copy")

				skipped++
			}
		}()
	}

	if skipped > 0 {
		uc.deps.Logger.Warn("BackupUseCase - streamFilesToZip - completed with skipped files", logkit.Fields{
			"total":   len(files),
			"skipped": skipped,
		})
	}

	return skipped
}
