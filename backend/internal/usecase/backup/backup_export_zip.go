package backup

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
// archive, and then streams backup file payloads via streamFilesToZip when
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

		if err := uc.streamFilesToZip(ctx, zw, data.Files); err != nil {
			pw.CloseWithError(err)

			return
		}

		uc.deps.Logger.Info("BackupUseCase - ExportZIP - completed", logkit.Fields{
			"challenges": len(data.Challenges),
			"teams":      len(data.Teams),
			"files":      len(data.Files),
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
// the ZIP archive under the canonical backup file path for its type. Files are
// processed sequentially. Any path, storage, or copy error fails the streaming
// export through the io.Pipe so operators do not receive an apparently complete
// archive with silently missing payloads.
func (uc *BackupUseCase) streamFilesToZip(ctx context.Context, zw *zip.Writer, files []domain.File) error {
	for _, file := range files {
		if err := ctx.Err(); err != nil {
			return err
		}

		path, err := backupFileZIPPath(file)
		if err != nil {
			return fmt.Errorf("BackupUseCase - streamFilesToZip - path %s: %w", file.Filename, err)
		}

		f, err := zw.Create(path)
		if err != nil {
			return fmt.Errorf("BackupUseCase - streamFilesToZip - create %s: %w", file.Filename, err)
		}

		rc, err := uc.deps.Storage.Download(ctx, file.Location)
		if err != nil {
			return fmt.Errorf("BackupUseCase - streamFilesToZip - download %s: %w", file.Location, err)
		}

		if err := func() error {
			defer func() { _ = rc.Close() }()

			if _, err := io.Copy(f, rc); err != nil {
				return fmt.Errorf("BackupUseCase - streamFilesToZip - copy %s: %w", file.Filename, err)
			}

			return nil
		}(); err != nil {
			return err
		}
	}

	return nil
}
