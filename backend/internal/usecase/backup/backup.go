package backup

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/storage"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/logger"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

const (
	exportTimeout           = 5 * time.Minute
	defaultCSVExportMaxRows = 100_000
)

type BackupUseCase struct {
	deps BackupDeps
}

type BackupDeps struct {
	CompetitionRepo repo.CompetitionRepository
	ChallengeRepo   repo.ChallengeRepository
	HintRepo        repo.HintRepository
	TeamRepo        repo.TeamRepository
	UserRepo        repo.UserRepository
	AwardRepo       repo.AwardRepository
	SolveRepo       repo.SolveRepository
	SubmissionRepo  repo.SubmissionRepository
	FileRepo        repo.FileRepository
	BackupRepo      repo.BackupRepository
	SettingsRepo    repo.SettingsRepository
	Storage         storage.Provider
	TM              repo.TransactionManager
	Logger          logger.Logger
}

var _ usecase.BackupUseCase = (*BackupUseCase)(nil)

func NewBackupUseCase(deps BackupDeps) *BackupUseCase {
	if deps.Logger == nil {
		deps.Logger = logger.Noop()
	}
	return &BackupUseCase{deps: deps}
}

func (uc *BackupUseCase) Export(ctx context.Context, opts entity.ExportOptions) (*entity.BackupData, error) {
	ctx, cancel := context.WithTimeout(ctx, exportTimeout)
	defer cancel()

	g, gCtx := errgroup.WithContext(ctx)
	backup := &entity.BackupData{
		Version:    entity.BackupVersion,
		ExportedAt: time.Now().UTC(),
	}
	var mu sync.Mutex

	g.Go(func() error {
		comp, err := uc.deps.CompetitionRepo.Get(gCtx)
		if err != nil {
			return fmt.Errorf("BackupUseCase - Export - CompetitionRepo.Get: %w", err)
		}
		mu.Lock()
		backup.Competition = comp
		mu.Unlock()
		return nil
	})

	g.Go(func() error {
		challenges, err := uc.fetchChallengesWithHints(gCtx)
		if err != nil {
			return fmt.Errorf("BackupUseCase - Export - fetchChallengesWithHints: %w", err)
		}
		mu.Lock()
		backup.Challenges = challenges
		mu.Unlock()
		return nil
	})

	uc.exportOptional(gCtx, backup, opts, &mu, g)

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("BackupUseCase - Export - errgroup.Wait: %w", err)
	}
	return backup, nil
}

func (uc *BackupUseCase) exportOptional(
	ctx context.Context,
	backup *entity.BackupData,
	opts entity.ExportOptions,
	mu *sync.Mutex,
	g *errgroup.Group,
) {
	uc.exportOptionalTeams(ctx, backup, opts, mu, g)
	uc.exportOptionalUsers(ctx, backup, opts, mu, g)
	uc.exportOptionalAwards(ctx, backup, opts, mu, g)
	uc.exportOptionalSolves(ctx, backup, opts, mu, g)
	uc.exportOptionalHintUnlocks(ctx, backup, opts, mu, g)
	uc.exportOptionalFiles(ctx, backup, opts, mu, g)
}

func (uc *BackupUseCase) exportOptionalTeams(ctx context.Context, backup *entity.BackupData, opts entity.ExportOptions, mu *sync.Mutex, g *errgroup.Group) {
	if !opts.IncludeTeams {
		return
	}
	g.Go(func() error {
		teams, err := uc.fetchTeamsWithMembers(ctx)
		if err != nil {
			return fmt.Errorf("BackupUseCase - Export - fetchTeamsWithMembers: %w", err)
		}
		mu.Lock()
		backup.Teams = teams
		mu.Unlock()
		return nil
	})
}

func (uc *BackupUseCase) exportOptionalUsers(ctx context.Context, backup *entity.BackupData, opts entity.ExportOptions, mu *sync.Mutex, g *errgroup.Group) {
	if !opts.IncludeUsers {
		return
	}
	g.Go(func() error {
		users, err := uc.fetchUsers(ctx)
		if err != nil {
			return fmt.Errorf("BackupUseCase - Export - fetchUsers: %w", err)
		}
		mu.Lock()
		backup.Users = users
		mu.Unlock()
		return nil
	})
}

func (uc *BackupUseCase) exportOptionalAwards(ctx context.Context, backup *entity.BackupData, opts entity.ExportOptions, mu *sync.Mutex, g *errgroup.Group) {
	if !opts.IncludeAwards {
		return
	}
	g.Go(func() error {
		awards, err := uc.fetchAwards(ctx)
		if err != nil {
			return fmt.Errorf("BackupUseCase - Export - fetchAwards: %w", err)
		}
		mu.Lock()
		backup.Awards = awards
		mu.Unlock()
		return nil
	})
}

func (uc *BackupUseCase) exportOptionalSolves(ctx context.Context, backup *entity.BackupData, opts entity.ExportOptions, mu *sync.Mutex, g *errgroup.Group) {
	if !opts.IncludeSolves {
		return
	}
	g.Go(func() error {
		solves, err := uc.fetchSolves(ctx)
		if err != nil {
			return fmt.Errorf("BackupUseCase - Export - fetchSolves: %w", err)
		}
		mu.Lock()
		backup.Solves = solves
		mu.Unlock()
		return nil
	})
}

func (uc *BackupUseCase) exportOptionalHintUnlocks(ctx context.Context, backup *entity.BackupData, opts entity.ExportOptions, mu *sync.Mutex, g *errgroup.Group) {
	if !opts.IncludeSolves {
		return
	}
	g.Go(func() error {
		unlocks, err := uc.deps.HintRepo.GetAllUnlocks(ctx)
		if err != nil {
			return fmt.Errorf("BackupUseCase - Export - HintRepo.GetAllUnlocks: %w", err)
		}
		result := make([]entity.HintUnlock, len(unlocks))
		for i, u := range unlocks {
			result[i] = *u
		}
		mu.Lock()
		backup.HintUnlocks = result
		mu.Unlock()
		return nil
	})
}

func (uc *BackupUseCase) exportOptionalFiles(ctx context.Context, backup *entity.BackupData, opts entity.ExportOptions, mu *sync.Mutex, g *errgroup.Group) {
	if !opts.IncludeFiles {
		return
	}
	g.Go(func() error {
		files, err := uc.fetchFiles(ctx)
		if err != nil {
			return fmt.Errorf("BackupUseCase - Export - fetchFiles: %w", err)
		}
		mu.Lock()
		backup.Files = files
		mu.Unlock()
		return nil
	})
}

const backupFetchConcurrency = 20

func (uc *BackupUseCase) fetchChallengesWithHints(ctx context.Context) ([]entity.ChallengeExport, error) {
	challengesWithSolved, err := uc.deps.ChallengeRepo.GetAll(ctx, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("BackupUseCase - fetchChallengesWithHints - ChallengeRepo.GetAll: %w", err)
	}

	result := make([]entity.ChallengeExport, len(challengesWithSolved))
	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(backupFetchConcurrency)
	for i, cws := range challengesWithSolved {
		g.Go(func() error {
			hints, err := uc.deps.HintRepo.GetByChallengeID(gCtx, cws.Challenge.ID)
			if err != nil {
				return fmt.Errorf("BackupUseCase - fetchChallengesWithHints - HintRepo.GetByChallengeID: %w", err)
			}
			hintsCopy := make([]entity.Hint, len(hints))
			for j, h := range hints {
				hintsCopy[j] = *h
			}
			result[i] = entity.ChallengeExport{
				Challenge: *cws.Challenge,
				Hints:     hintsCopy,
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("BackupUseCase - fetchChallengesWithHints - errgroup.Wait: %w", err)
	}
	return result, nil
}

func (uc *BackupUseCase) fetchTeamsWithMembers(ctx context.Context) ([]entity.TeamExport, error) {
	teams, err := uc.deps.TeamRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("BackupUseCase - fetchTeamsWithMembers - TeamRepo.GetAll: %w", err)
	}

	result := make([]entity.TeamExport, len(teams))
	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(backupFetchConcurrency)
	for i, team := range teams {
		g.Go(func() error {
			members, err := uc.deps.UserRepo.GetByTeamID(gCtx, team.ID)
			if err != nil {
				return fmt.Errorf("BackupUseCase - fetchTeamsWithMembers - UserRepo.GetByTeamID: %w", err)
			}
			memberIDs := make([]uuid.UUID, len(members))
			for j, m := range members {
				memberIDs[j] = m.ID
			}
			result[i] = entity.TeamExport{
				Team:      *team,
				MemberIDs: memberIDs,
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("BackupUseCase - fetchTeamsWithMembers - errgroup.Wait: %w", err)
	}
	return result, nil
}

func (uc *BackupUseCase) fetchUsers(ctx context.Context) ([]entity.UserExport, error) {
	users, err := uc.deps.UserRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("BackupUseCase - fetchUsers - UserRepo.GetAll: %w", err)
	}

	result := make([]entity.UserExport, 0, len(users))
	for _, u := range users {
		result = append(result, entity.UserExport{
			ID:       u.ID,
			Username: u.Username,
			Email:    u.Email,
			Role:     u.Role,
			TeamID:   u.TeamID,
		})
	}
	return result, nil
}

func (uc *BackupUseCase) fetchAwards(ctx context.Context) ([]entity.Award, error) {
	awards, err := uc.deps.AwardRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("BackupUseCase - fetchAwards - AwardRepo.GetAll: %w", err)
	}

	result := make([]entity.Award, len(awards))
	for i, a := range awards {
		result[i] = *a
	}

	return result, nil
}

func (uc *BackupUseCase) fetchSolves(ctx context.Context) ([]entity.Solve, error) {
	solves, err := uc.deps.SolveRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("BackupUseCase - fetchSolves - SolveRepo.GetAll: %w", err)
	}

	result := make([]entity.Solve, len(solves))
	for i, s := range solves {
		result[i] = *s
	}

	return result, nil
}

func (uc *BackupUseCase) fetchFiles(ctx context.Context) ([]entity.File, error) {
	files, err := uc.deps.FileRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("BackupUseCase - fetchFiles - FileRepo.GetAll: %w", err)
	}

	result := make([]entity.File, len(files))
	for i, f := range files {
		result[i] = *f
	}

	return result, nil
}

func (uc *BackupUseCase) ExportZIP(ctx context.Context, opts entity.ExportOptions) (io.ReadCloser, error) {
	pr, pw := io.Pipe()
	go uc.exportZIPWorker(ctx, pw, opts)
	return pr, nil
}

func (uc *BackupUseCase) exportZIPWorker(ctx context.Context, pw *io.PipeWriter, opts entity.ExportOptions) {
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
		uc.deps.Logger.Info("BackupUseCase - ExportZIP - completed", logger.Fields{
			"challenges": len(data.Challenges),
			"teams":      len(data.Teams),
			"files":      len(data.Files),
			"skipped":    skipped,
		})
	} else {
		uc.deps.Logger.Info("BackupUseCase - ExportZIP - completed", logger.Fields{
			"challenges": len(data.Challenges),
			"teams":      len(data.Teams),
			"files":      0,
		})
	}
}

func (uc *BackupUseCase) writeBackupJSON(zw *zip.Writer, data *entity.BackupData) error {
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
	if _, err := fmt.Fprintf(readme, "# AstroCTFb Backup\n\nBackup created: %s\nVersion: %s\n", data.ExportedAt.Format(time.RFC3339), data.Version); err != nil {
		return fmt.Errorf("BackupUseCase - ExportZIP - write README: %w", err)
	}
	return nil
}

func (uc *BackupUseCase) streamFilesToZip(ctx context.Context, zw *zip.Writer, files []entity.File) int {
	var skipped int
	for _, file := range files {
		if ctx.Err() != nil {
			break
		}
		path := fmt.Sprintf("files/challenge-%s/%s", file.ChallengeID, file.Filename)
		f, err := zw.Create(path)
		if err != nil {
			uc.deps.Logger.WithError(err).WithFields(logger.Fields{"file": file.Filename}).Warn("BackupUseCase - streamFilesToZip - create")
			skipped++
			continue
		}

		rc, err := uc.deps.Storage.Download(ctx, file.Location)
		if err != nil {
			uc.deps.Logger.WithError(err).WithFields(logger.Fields{"file": file.Filename, "location": file.Location}).Warn("BackupUseCase - streamFilesToZip - download")
			skipped++
			continue
		}

		if _, err := io.Copy(f, rc); err != nil {
			_ = rc.Close()
			uc.deps.Logger.WithError(err).WithFields(logger.Fields{"file": file.Filename}).Warn("BackupUseCase - streamFilesToZip - copy")
			skipped++
			continue
		}
		_ = rc.Close()
	}

	if skipped > 0 {
		uc.deps.Logger.Warn("BackupUseCase - streamFilesToZip - completed with skipped files", logger.Fields{
			"total":   len(files),
			"skipped": skipped,
		})
	}

	return skipped
}

func (uc *BackupUseCase) ImportZIP(ctx context.Context, r io.ReaderAt, size int64, opts entity.ImportOptions) (*entity.ImportResult, error) {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return nil, fmt.Errorf("BackupUseCase - ImportZIP - NewReader: %w", err)
	}
	backupData, err := uc.importZIPReadBackup(zr)
	if err != nil {
		return nil, fmt.Errorf("BackupUseCase - ImportZIP - importZIPReadBackup: %w", err)
	}
	if err := uc.importZIPValidateVersion(backupData); err != nil {
		return nil, fmt.Errorf("BackupUseCase - ImportZIP - importZIPValidateVersion: %w", err)
	}
	result := &entity.ImportResult{Success: true}
	if err := uc.importZIPRunTx(ctx, backupData, opts); err != nil {
		return nil, fmt.Errorf("BackupUseCase - ImportZIP - importZIPRunTx: %w", err)
	}
	// File upload to storage happens after the DB transaction commits intentionally:
	// storage uploads are not transactional. If uploads fail, the DB records are kept
	// and the caller receives a partial result with SkippedCount > 0 so the issue is visible.
	// A full rollback would require compensating deletes in DB, which adds complexity with
	// no meaningful safety gain since files can be re-uploaded manually.
	if len(backupData.Files) > 0 {
		fileErrors := uc.importFilesToStorage(ctx, zr, backupData.Files, opts)
		if len(fileErrors) > 0 {
			result.Errors = fileErrors
			result.SkippedCount = len(fileErrors)
		}
	}
	uc.deps.Logger.Info("BackupUseCase - ImportZIP - completed", logger.Fields{
		"challenges": len(backupData.Challenges),
		"teams":      len(backupData.Teams),
		"users":      len(backupData.Users),
		"files":      len(backupData.Files),
		"skipped":    result.SkippedCount,
	})
	return result, nil
}

func (uc *BackupUseCase) importZIPReadBackup(zr *zip.Reader) (*entity.BackupData, error) {
	for _, f := range zr.File {
		if f.Name != "backup.json" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("BackupUseCase - ImportZIP - open backup.json: %w", err)
		}
		backupData := &entity.BackupData{}
		if err := json.NewDecoder(rc).Decode(backupData); err != nil {
			_ = rc.Close()
			return nil, fmt.Errorf("BackupUseCase - ImportZIP - decode backup.json: %w", err)
		}
		_ = rc.Close()
		return backupData, nil
	}
	return nil, httperr.ErrBackupJSONNotFound
}

func (uc *BackupUseCase) importZIPValidateVersion(backupData *entity.BackupData) error {
	if backupData.Version != entity.BackupVersion {
		return httperr.ErrBackupVersionUnsupported
	}
	return nil
}

func (uc *BackupUseCase) importZIPRunTx(ctx context.Context, backupData *entity.BackupData, opts entity.ImportOptions) error {
	return uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		if opts.EraseExisting {
			if err := uc.deps.BackupRepo.EraseAllTables(ctx); err != nil {
				return fmt.Errorf("BackupUseCase - ImportZIP - BackupRepo.EraseAllTables: %w", err)
			}
		}
		return uc.importZIPRunTxImports(ctx, backupData, opts)
	})
}

func (uc *BackupUseCase) importZIPRunTxImports(ctx context.Context, backupData *entity.BackupData, opts entity.ImportOptions) error {
	if err := uc.deps.BackupRepo.ImportCompetition(ctx, backupData.Competition); err != nil {
		return fmt.Errorf("BackupUseCase - ImportZIP - BackupRepo.ImportCompetition: %w", err)
	}
	if err := uc.deps.BackupRepo.ImportChallenges(ctx, backupData); err != nil {
		return fmt.Errorf("BackupUseCase - ImportZIP - BackupRepo.ImportChallenges: %w", err)
	}
	if err := uc.deps.BackupRepo.ImportTeams(ctx, backupData, opts); err != nil {
		return fmt.Errorf("BackupUseCase - ImportZIP - BackupRepo.ImportTeams: %w", err)
	}
	uc.importNormalizeUserRoles(backupData, opts)
	if err := uc.deps.BackupRepo.ImportUsers(ctx, backupData, opts); err != nil {
		return fmt.Errorf("BackupUseCase - ImportZIP - BackupRepo.ImportUsers: %w", err)
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
	return nil
}

// importNormalizeUserRoles ensures every imported user has a valid role.
// Unless PreserveAdminRoles is explicitly set, all admin roles are downgraded
// to RoleUser, preventing a crafted backup from injecting admin accounts.
func (uc *BackupUseCase) importNormalizeUserRoles(backupData *entity.BackupData, opts entity.ImportOptions) {
	for i := range backupData.Users {
		switch backupData.Users[i].Role {
		case entity.RoleAdmin:
			if !opts.PreserveAdminRoles {
				backupData.Users[i].Role = entity.RoleUser
			}
		case entity.RoleUser:
			// valid
		default:
			backupData.Users[i].Role = entity.RoleUser
		}
	}
}

const maxConcurrentFileUploads = 5

func (uc *BackupUseCase) importFilesToStorage(ctx context.Context, zr *zip.Reader, files []entity.File, opts entity.ImportOptions) []string {
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
			} else {
				uploaded++
			}
			mu.Unlock()
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		errs = append(errs, err.Error())
	}

	if len(errs) > 0 {
		uc.deps.Logger.Warn("BackupUseCase - importFilesToStorage - completed with errors", logger.Fields{
			"total":    len(files),
			"uploaded": uploaded,
			"errors":   len(errs),
		})
	}
	return errs
}

func (uc *BackupUseCase) importFilesBuildFileMap(files []entity.File) map[string]entity.File {
	m := make(map[string]entity.File)
	for _, f := range files {
		path := fmt.Sprintf("files/challenge-%s/%s", f.ChallengeID, f.Filename)
		m[path] = f
	}
	return m
}

type importFileTask struct {
	zf   *zip.File
	file entity.File
}

func (uc *BackupUseCase) importFilesBuildTasks(zr *zip.Reader, fileMap map[string]entity.File) []importFileTask {
	var tasks []importFileTask
	for _, zf := range zr.File {
		file, ok := fileMap[zf.Name]
		if !ok {
			continue
		}
		tasks = append(tasks, importFileTask{zf: zf, file: file})
	}
	return tasks
}

func (uc *BackupUseCase) importFileUploadOne(ctx context.Context, zf *zip.File, file entity.File, opts entity.ImportOptions) string {
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

func (uc *BackupUseCase) importFileUploadWithHash(ctx context.Context, name string, rc io.Reader, size int64, file entity.File) string {
	hash := sha256.New()
	tee := io.TeeReader(rc, hash)
	if err := uc.deps.Storage.Upload(ctx, file.Location, tee, size, "application/octet-stream"); err != nil {
		return fmt.Sprintf("upload %s: %v", name, err)
	}
	hashStr := hex.EncodeToString(hash.Sum(nil))
	if hashStr != file.SHA256 {
		if delErr := uc.deps.Storage.Delete(ctx, file.Location); delErr != nil {
			uc.deps.Logger.WithError(delErr).WithFields(logger.Fields{"location": file.Location}).Warn("BackupUseCase - importFileUploadWithHash - delete after mismatch")
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

func (uc *BackupUseCase) Reset(ctx context.Context, opts entity.AdminResetOptions) error {
	tables := make([]string, 0)
	if opts.Submissions {
		tables = append(tables, "solves", "submissions")
	}
	if opts.Challenges {
		tables = append(tables, "hint_unlocks", "files", "hints", "challenges")
	}
	if opts.Accounts {
		tables = append(tables, "awards", "users", "teams")
	}
	if opts.Notifications {
		tables = append(tables, "notifications")
	}
	if opts.Pages {
		tables = append(tables, "pages")
	}
	if len(tables) == 0 {
		return nil
	}
	return uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		if err := uc.deps.BackupRepo.EraseTables(ctx, tables); err != nil {
			return fmt.Errorf("BackupUseCase - Reset - BackupRepo.EraseTables: %w", err)
		}
		return nil
	})
}

func (uc *BackupUseCase) ExportCSV(ctx context.Context, tableName string) ([]byte, error) {
	if !isAllowedCSVTable(tableName) {
		return nil, httperr.ErrBackupTableUnsupported
	}

	exporters := map[string]func(context.Context) ([]byte, error){
		"users":       uc.exportCSVUsers,
		"teams":       uc.exportCSVTeams,
		"challenges":  uc.exportCSVChallenges,
		"submissions": uc.exportCSVSubmissions,
		"solves":      uc.exportCSVSolves,
		"awards":      uc.exportCSVAwards,
	}

	exporter, ok := exporters[tableName]
	if !ok {
		return nil, httperr.ErrBackupTableUnsupported
	}

	return exporter(ctx)
}

func (uc *BackupUseCase) exportCSVUsers(ctx context.Context) ([]byte, error) {
	users, err := uc.deps.UserRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("BackupUseCase - ExportCSV - UserRepo.GetAll: %w", err)
	}
	return csvExportUsers(users)
}

func (uc *BackupUseCase) exportCSVTeams(ctx context.Context) ([]byte, error) {
	teams, err := uc.deps.TeamRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("BackupUseCase - ExportCSV - TeamRepo.GetAll: %w", err)
	}
	return csvExportTeams(teams)
}

func (uc *BackupUseCase) exportCSVChallenges(ctx context.Context) ([]byte, error) {
	cws, err := uc.deps.ChallengeRepo.GetAll(ctx, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("BackupUseCase - ExportCSV - ChallengeRepo.GetAll: %w", err)
	}
	challenges := make([]*entity.Challenge, len(cws))
	for i, cw := range cws {
		challenges[i] = cw.Challenge
	}
	return csvExportChallenges(challenges)
}

func (uc *BackupUseCase) exportCSVSubmissions(ctx context.Context) ([]byte, error) {
	settings, err := uc.deps.SettingsRepo.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("BackupUseCase - ExportCSV - SettingsRepo.Get: %w", err)
	}
	maxRows := settings.CSVExportMaxRows
	if maxRows <= 0 {
		maxRows = defaultCSVExportMaxRows
	}
	subs, err := uc.deps.SubmissionRepo.GetAll(ctx, maxRows, 0)
	if err != nil {
		return nil, fmt.Errorf("BackupUseCase - ExportCSV - SubmissionRepo.GetAll: %w", err)
	}
	return csvExportSubmissions(subs)
}

func (uc *BackupUseCase) exportCSVSolves(ctx context.Context) ([]byte, error) {
	solves, err := uc.deps.SolveRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("BackupUseCase - ExportCSV - SolveRepo.GetAll: %w", err)
	}
	return csvExportSolves(solves)
}

func (uc *BackupUseCase) exportCSVAwards(ctx context.Context) ([]byte, error) {
	awards, err := uc.deps.AwardRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("BackupUseCase - ExportCSV - AwardRepo.GetAll: %w", err)
	}
	return csvExportAwards(awards)
}

func (uc *BackupUseCase) ImportCSV(ctx context.Context, tableName string, data []byte) (*usecase.CSVImportResult, error) {
	if !isAllowedCSVTable(tableName) {
		return nil, httperr.ErrBackupTableUnsupported
	}

	header, rows, err := parseCSV(data)
	if err != nil {
		return nil, fmt.Errorf("BackupUseCase - ImportCSV - parseCSV: %w", err)
	}
	if len(rows) == 0 {
		return &usecase.CSVImportResult{Success: true}, nil
	}

	var imported int
	var csvErrors []string

	txErr := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		var txErr error
		imported, csvErrors, txErr = uc.deps.BackupRepo.ImportCSV(ctx, tableName, header, rows)
		return txErr
	})
	if txErr != nil {
		return nil, fmt.Errorf("BackupUseCase - ImportCSV - TM.Run: %w", txErr)
	}

	return &usecase.CSVImportResult{
		Success:       len(csvErrors) == 0,
		ImportedCount: imported,
		Errors:        csvErrors,
		SkippedCount:  len(rows) - imported,
	}, nil
}
