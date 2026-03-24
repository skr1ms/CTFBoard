package backup

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/json"
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
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

func (uc *BackupUseCase) ImportZIP(ctx context.Context, r io.ReaderAt, size int64, opts domain.ImportOptions) (*domain.ImportResult, error) {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return nil, fmt.Errorf("BackupUseCase - ImportZIP - NewReader: %w", err)
	}
	var totalUncompressed uint64
	for _, f := range zr.File {
		totalUncompressed += f.UncompressedSize64
	}
	if size < 0 {
		return nil, fmt.Errorf("BackupUseCase - ImportZIP: negative size")
	}
	maxAllowed := uint64(size) * maxUncompressedRatio
	if maxAllowed > maxUncompressedAbsolute {
		maxAllowed = maxUncompressedAbsolute
	}
	if totalUncompressed > maxAllowed {
		return nil, fmt.Errorf("BackupUseCase - ImportZIP: uncompressed size %d exceeds limit %d (zip bomb protection)", totalUncompressed, maxAllowed)
	}
	backupData, err := uc.importZIPReadBackup(zr)
	if err != nil {
		return nil, fmt.Errorf("BackupUseCase - ImportZIP - importZIPReadBackup: %w", err)
	}
	if err := uc.importZIPValidateVersion(backupData); err != nil {
		return nil, fmt.Errorf("BackupUseCase - ImportZIP - importZIPValidateVersion: %w", err)
	}
	result := &domain.ImportResult{Success: true}
	if err := uc.importZIPRunTx(ctx, backupData, opts); err != nil {
		return nil, fmt.Errorf("BackupUseCase - ImportZIP - importZIPRunTx: %w", err)
	}
	// File upload to storage happens after the DB transaction commits intentionally:
	// storage uploads are not transactional. If uploads fail, the DB records are kept
	// and the caller receives a partial result with SkippedCount > 0 so the issue is visible.
	// A full rollback would require compensating deletes in DB, which adds complexity with
	// no meaningful safety gain since files can be re-uploaded manually.
	if len(backupData.Files) > 0 {
		fileErrors, err := uc.importFilesToStorage(ctx, zr, backupData.Files, opts)
		if err != nil && uc.deps.Logger != nil {
			uc.deps.Logger.WithError(err).Warn("BackupUseCase - ImportZIP - importFilesToStorage")
		}
		if len(fileErrors) > 0 {
			result.Errors = fileErrors
			result.SkippedCount = len(fileErrors)
		}
	}
	uc.deps.Logger.Info("BackupUseCase - ImportZIP - completed", logkit.Fields{
		"challenges": len(backupData.Challenges),
		"teams":      len(backupData.Teams),
		"users":      len(backupData.Users),
		"files":      len(backupData.Files),
		"skipped":    result.SkippedCount,
	})
	return result, nil
}

func (uc *BackupUseCase) importZIPReadBackup(zr *zip.Reader) (*domain.BackupData, error) {
	for _, f := range zr.File {
		if f.Name != "backup.json" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("BackupUseCase - ImportZIP - open backup.json: %w", err)
		}
		limited := io.LimitReader(rc, maxBackupJSONSize)
		backupData := &domain.BackupData{}
		if err := json.NewDecoder(limited).Decode(backupData); err != nil {
			_ = rc.Close()
			return nil, fmt.Errorf("BackupUseCase - ImportZIP - decode backup.json: %w", err)
		}
		_ = rc.Close()
		return backupData, nil
	}
	return nil, httperr.ErrBackupJSONNotFound
}

func (uc *BackupUseCase) importZIPValidateVersion(backupData *domain.BackupData) error {
	if backupData.Version != domain.BackupVersion {
		return httperr.ErrBackupVersionUnsupported
	}
	return nil
}

func (uc *BackupUseCase) importZIPRunTx(ctx context.Context, backupData *domain.BackupData, opts domain.ImportOptions) error {
	return uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		if opts.EraseExisting {
			if uc.deps.AuditLogRepo != nil && opts.AdminUserID != nil {
				auditEntry := &domain.AuditLog{
					UserID:     opts.AdminUserID,
					Action:     domain.AuditActionImportErase,
					EntityType: domain.AuditEntityBackup,
					EntityID:   "import",
					IP:         opts.AdminIP,
					Details:    map[string]any{"erase_existing": true},
				}
				if err := uc.deps.AuditLogRepo.Create(ctx, auditEntry); err != nil {
					return fmt.Errorf("BackupUseCase - ImportZIP - AuditLogRepo.Create: %w", err)
				}
			}
			if err := uc.deps.BackupRepo.EraseAllTables(ctx); err != nil {
				return fmt.Errorf("BackupUseCase - ImportZIP - BackupRepo.EraseAllTables: %w", err)
			}
		}
		return uc.importZIPRunTxImports(ctx, backupData, opts)
	})
}

func (uc *BackupUseCase) importZIPRunTxImports(ctx context.Context, backupData *domain.BackupData, opts domain.ImportOptions) error {
	if err := uc.deps.BackupRepo.ImportCompetition(ctx, backupData.Competition); err != nil {
		return fmt.Errorf("BackupUseCase - ImportZIP - BackupRepo.ImportCompetition: %w", err)
	}
	if err := uc.deps.BackupRepo.ImportTags(ctx, backupData); err != nil {
		return fmt.Errorf("BackupUseCase - ImportZIP - BackupRepo.ImportTags: %w", err)
	}
	if err := uc.deps.BackupRepo.ImportChallenges(ctx, backupData); err != nil {
		return fmt.Errorf("BackupUseCase - ImportZIP - BackupRepo.ImportChallenges: %w", err)
	}
	if err := uc.deps.BackupRepo.ImportChallengeTags(ctx, backupData); err != nil {
		return fmt.Errorf("BackupUseCase - ImportZIP - BackupRepo.ImportChallengeTags: %w", err)
	}
	if err := uc.deps.BackupRepo.ImportBrackets(ctx, backupData); err != nil {
		return fmt.Errorf("BackupUseCase - ImportZIP - BackupRepo.ImportBrackets: %w", err)
	}
	uc.importNormalizeUserRoles(backupData, opts)
	if err := uc.deps.BackupRepo.ImportUsers(ctx, backupData, opts); err != nil {
		return fmt.Errorf("BackupUseCase - ImportZIP - BackupRepo.ImportUsers: %w", err)
	}
	if err := uc.deps.BackupRepo.ImportTeams(ctx, backupData, opts); err != nil {
		return fmt.Errorf("BackupUseCase - ImportZIP - BackupRepo.ImportTeams: %w", err)
	}
	if err := uc.deps.BackupRepo.UpdateUserTeamIDs(ctx, backupData); err != nil {
		return fmt.Errorf("BackupUseCase - ImportZIP - BackupRepo.UpdateUserTeamIDs: %w", err)
	}
	if err := uc.deps.BackupRepo.ImportAwards(ctx, backupData); err != nil {
		return fmt.Errorf("BackupUseCase - ImportZIP - BackupRepo.ImportAwards: %w", err)
	}
	if err := uc.deps.BackupRepo.ImportSolves(ctx, backupData); err != nil {
		return fmt.Errorf("BackupUseCase - ImportZIP - BackupRepo.ImportSolves: %w", err)
	}
	if err := uc.deps.BackupRepo.ImportHintUnlocks(ctx, backupData); err != nil {
		return fmt.Errorf("BackupUseCase - ImportZIP - BackupRepo.ImportHintUnlocks: %w", err)
	}
	if err := uc.deps.BackupRepo.ImportFileMetadata(ctx, backupData); err != nil {
		return fmt.Errorf("BackupUseCase - ImportZIP - BackupRepo.ImportFileMetadata: %w", err)
	}
	if err := uc.deps.BackupRepo.ImportChallengeRequirements(ctx, backupData); err != nil {
		return fmt.Errorf("BackupUseCase - ImportZIP - BackupRepo.ImportChallengeRequirements: %w", err)
	}
	if err := uc.deps.BackupRepo.ImportSolutions(ctx, backupData); err != nil {
		return fmt.Errorf("BackupUseCase - ImportZIP - BackupRepo.ImportSolutions: %w", err)
	}
	if err := uc.deps.BackupRepo.ImportRatings(ctx, backupData); err != nil {
		return fmt.Errorf("BackupUseCase - ImportZIP - BackupRepo.ImportRatings: %w", err)
	}
	if err := uc.deps.BackupRepo.ImportComments(ctx, backupData); err != nil {
		return fmt.Errorf("BackupUseCase - ImportZIP - BackupRepo.ImportComments: %w", err)
	}
	if err := uc.deps.BackupRepo.ImportFields(ctx, backupData); err != nil {
		return fmt.Errorf("BackupUseCase - ImportZIP - BackupRepo.ImportFields: %w", err)
	}
	if err := uc.deps.BackupRepo.ImportFieldValues(ctx, backupData); err != nil {
		return fmt.Errorf("BackupUseCase - ImportZIP - BackupRepo.ImportFieldValues: %w", err)
	}
	return nil
}

// importNormalizeUserRoles ensures every imported user has a valid role.
// Unless PreserveAdminRoles is explicitly set, all admin roles are downgraded
// to RoleUser, preventing a crafted backup from injecting admin accounts.
func (uc *BackupUseCase) importNormalizeUserRoles(backupData *domain.BackupData, opts domain.ImportOptions) {
	for i := range backupData.Users {
		switch domain.Role(backupData.Users[i].Role) {
		case domain.RoleAdmin:
			if !opts.PreserveAdminRoles {
				backupData.Users[i].Role = string(domain.RoleUser)
			}
		case domain.RoleUser:
		default:
			backupData.Users[i].Role = string(domain.RoleUser)
		}
	}
}

const (
	maxConcurrentFileUploads = 5
	maxUncompressedRatio     = 10
	maxUncompressedAbsolute  = 2 * 1024 * 1024 * 1024
	maxBackupJSONSize        = 100 * 1024 * 1024
)

func (uc *BackupUseCase) importFilesToStorage(ctx context.Context, zr *zip.Reader, files []domain.File, opts domain.ImportOptions) ([]string, error) {
	fileMap := uc.importFilesBuildFileMap(files)
	tasks := uc.importFilesBuildTasks(zr, fileMap)

	var mu sync.Mutex
	var errs []string
	var uploaded int

	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(maxConcurrentFileUploads)

	for _, t := range tasks {
		g.Go(func() error {
			errMsg := uc.importFileUploadOne(gCtx, t.zf, t.file, opts)
			mu.Lock()
			if errMsg != "" {
				errs = append(errs, errMsg)
				mu.Unlock()
				return fmt.Errorf("%s", errMsg)
			}
			uploaded++
			mu.Unlock()
			return nil
		})
	}
	waitErr := g.Wait()
	if waitErr != nil {
		uc.deps.Logger.Warn("BackupUseCase - importFilesToStorage - first error canceled rest", logkit.Error(waitErr))
	}
	if len(errs) > 0 {
		uc.deps.Logger.Warn("BackupUseCase - importFilesToStorage - completed with errors", logkit.Fields{
			"total":    len(files),
			"uploaded": uploaded,
			"errors":   len(errs),
		})
	}
	return errs, waitErr
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

func (uc *BackupUseCase) importFileUploadWithHash(ctx context.Context, name string, rc io.Reader, size int64, file domain.File) string {
	hash := sha256.New()
	tee := io.TeeReader(rc, hash)
	if err := uc.deps.Storage.Upload(ctx, file.Location, tee, size, "application/octet-stream"); err != nil {
		return fmt.Sprintf("upload %s: %v", name, err)
	}
	hashStr := crypto.HashHex(hash)
	if hashStr != file.SHA256 {
		if delErr := uc.deps.Storage.Delete(ctx, file.Location); delErr != nil {
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

// sanitizeFileLocation prevents path traversal by cleaning the location and
// ensuring it always lives under the "files/" prefix.
func sanitizeFileLocation(location string) string {
	cleaned := filepath.ToSlash(filepath.Clean("/" + location))
	cleaned = strings.TrimPrefix(cleaned, "/")
	if !strings.HasPrefix(cleaned, "files/") {
		cleaned = "files/" + filepath.Base(cleaned)
	}
	return cleaned
}
