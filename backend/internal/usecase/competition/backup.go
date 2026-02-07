package competition

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/repo"
	"github.com/skr1ms/CTFBoard/internal/storage"
	"github.com/skr1ms/CTFBoard/pkg/logger"
	"github.com/skr1ms/CTFBoard/pkg/usecaseutil"
	"golang.org/x/sync/errgroup"
)

type BackupDeps struct {
	CompetitionRepo repo.CompetitionRepository
	ChallengeRepo   repo.ChallengeRepository
	HintRepo        repo.HintRepository
	TeamRepo        repo.TeamRepository
	UserRepo        repo.UserRepository
	AwardRepo       repo.AwardRepository
	SolveRepo       repo.SolveRepository
	FileRepo        repo.FileRepository
	BackupRepo      repo.BackupRepository
	Storage         storage.Provider
	TxRepo          repo.TxRepository
	Logger          logger.Logger
}

type BackupUseCase struct {
	deps BackupDeps
}

func NewBackupUseCase(deps BackupDeps) *BackupUseCase {
	return &BackupUseCase{deps: deps}
}

func (uc *BackupUseCase) Export(ctx context.Context, opts entity.ExportOptions) (*entity.BackupData, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
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
			return usecaseutil.Wrap(err, "BackupUseCase - Export - Get")
		}
		mu.Lock()
		backup.Competition = comp
		mu.Unlock()
		return nil
	})

	g.Go(func() error {
		challenges, err := uc.fetchChallengesWithHints(gCtx)
		if err != nil {
			return usecaseutil.Wrap(err, "BackupUseCase - Export - fetchChallengesWithHints")
		}
		mu.Lock()
		backup.Challenges = challenges
		mu.Unlock()
		return nil
	})

	uc.exportOptional(gCtx, backup, opts, &mu, g)

	if err := g.Wait(); err != nil {
		return nil, err
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
	uc.exportOptionalFiles(ctx, backup, opts, mu, g)
}

func (uc *BackupUseCase) exportOptionalTeams(ctx context.Context, backup *entity.BackupData, opts entity.ExportOptions, mu *sync.Mutex, g *errgroup.Group) {
	if !opts.IncludeTeams {
		return
	}
	g.Go(func() error {
		teams, err := uc.fetchTeamsWithMembers(ctx)
		if err != nil {
			return usecaseutil.Wrap(err, "BackupUseCase - Export - fetchTeamsWithMembers")
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
			return usecaseutil.Wrap(err, "BackupUseCase - Export - fetchUsers")
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
			return usecaseutil.Wrap(err, "BackupUseCase - Export - fetchAwards")
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
			return usecaseutil.Wrap(err, "BackupUseCase - Export - fetchSolves")
		}
		mu.Lock()
		backup.Solves = solves
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
			return usecaseutil.Wrap(err, "BackupUseCase - Export - fetchFiles")
		}
		mu.Lock()
		backup.Files = files
		mu.Unlock()
		return nil
	})
}

func (uc *BackupUseCase) fetchChallengesWithHints(ctx context.Context) ([]entity.ChallengeExport, error) {
	challengesWithSolved, err := uc.deps.ChallengeRepo.GetAll(ctx, nil, nil)
	if err != nil {
		return nil, usecaseutil.Wrap(err, "BackupUseCase - fetchChallengesWithHints - GetAll")
	}

	result := make([]entity.ChallengeExport, 0, len(challengesWithSolved))
	for _, cws := range challengesWithSolved {
		hints, err := uc.deps.HintRepo.GetByChallengeID(ctx, cws.Challenge.ID)
		if err != nil {
			return nil, usecaseutil.Wrap(err, "BackupUseCase - fetchChallengesWithHints - GetByChallengeID")
		}

		hintsCopy := make([]entity.Hint, len(hints))
		for i, h := range hints {
			hintsCopy[i] = *h
		}

		result = append(result, entity.ChallengeExport{
			Challenge: *cws.Challenge,
			Hints:     hintsCopy,
		})
	}

	return result, nil
}

func (uc *BackupUseCase) fetchTeamsWithMembers(ctx context.Context) ([]entity.TeamExport, error) {
	teams, err := uc.deps.TeamRepo.GetAll(ctx)
	if err != nil {
		return nil, usecaseutil.Wrap(err, "BackupUseCase - fetchTeamsWithMembers - GetAll")
	}

	result := make([]entity.TeamExport, 0, len(teams))
	for _, team := range teams {
		members, err := uc.deps.UserRepo.GetByTeamID(ctx, team.ID)
		if err != nil {
			return nil, usecaseutil.Wrap(err, "BackupUseCase - fetchTeamsWithMembers - GetByTeamID")
		}

		memberIDs := make([]uuid.UUID, len(members))
		for i, m := range members {
			memberIDs[i] = m.ID
		}

		result = append(result, entity.TeamExport{
			Team:      *team,
			MemberIDs: memberIDs,
		})
	}

	return result, nil
}

func (uc *BackupUseCase) fetchUsers(ctx context.Context) ([]entity.UserExport, error) {
	users, err := uc.deps.UserRepo.GetAll(ctx)
	if err != nil {
		return nil, usecaseutil.Wrap(err, "BackupUseCase - fetchUsers - GetAll")
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
		return nil, usecaseutil.Wrap(err, "BackupUseCase - fetchAwards - GetAll")
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
		return nil, usecaseutil.Wrap(err, "BackupUseCase - fetchSolves - GetAll")
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
		return nil, usecaseutil.Wrap(err, "BackupUseCase - fetchFiles - GetAll")
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
		uc.deps.Logger.Info("BackupUseCase - ExportZIP - completed", map[string]any{
			"challenges": len(data.Challenges),
			"teams":      len(data.Teams),
			"files":      len(data.Files),
			"skipped":    skipped,
		})
	} else {
		uc.deps.Logger.Info("BackupUseCase - ExportZIP - completed", map[string]any{
			"challenges": len(data.Challenges),
			"teams":      len(data.Teams),
			"files":      0,
		})
	}
}

func (uc *BackupUseCase) writeBackupJSON(zw *zip.Writer, data *entity.BackupData) error {
	jsonFile, err := zw.Create("backup.json")
	if err != nil {
		return usecaseutil.Wrap(err, "BackupUseCase - ExportZIP - create backup.json")
	}
	if err := json.NewEncoder(jsonFile).Encode(data); err != nil {
		return usecaseutil.Wrap(err, "BackupUseCase - ExportZIP - encode backup.json")
	}
	readme, err := zw.Create("README.md")
	if err != nil {
		return usecaseutil.Wrap(err, "BackupUseCase - ExportZIP - create README.md")
	}
	if _, err := fmt.Fprintf(readme, "# CTFBoard Backup\n\nBackup created: %s\nVersion: %s\n", data.ExportedAt.Format(time.RFC3339), data.Version); err != nil {
		return usecaseutil.Wrap(err, "BackupUseCase - ExportZIP - write README")
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
			uc.deps.Logger.Warn("BackupUseCase - streamFilesToZip - create", map[string]any{
				"file":  file.Filename,
				"error": err.Error(),
			})
			skipped++
			continue
		}

		rc, err := uc.deps.Storage.Download(ctx, file.Location)
		if err != nil {
			uc.deps.Logger.Warn("BackupUseCase - streamFilesToZip - download", map[string]any{
				"file":     file.Filename,
				"location": file.Location,
				"error":    err.Error(),
			})
			skipped++
			continue
		}

		if _, err := io.Copy(f, rc); err != nil {
			uc.deps.Logger.Warn("BackupUseCase - streamFilesToZip - copy", map[string]any{
				"file":  file.Filename,
				"error": err.Error(),
			})
			skipped++
		}
		_ = rc.Close()
	}

	if skipped > 0 {
		uc.deps.Logger.Warn("BackupUseCase - streamFilesToZip - completed with skipped files", map[string]any{
			"total":   len(files),
			"skipped": skipped,
		})
	}

	return skipped
}

func (uc *BackupUseCase) ImportZIP(ctx context.Context, r io.ReaderAt, size int64, opts entity.ImportOptions) (*entity.ImportResult, error) {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return nil, usecaseutil.Wrap(err, "BackupUseCase - ImportZIP - NewReader")
	}
	backupData, err := uc.importZIPReadBackup(zr)
	if err != nil {
		return nil, err
	}
	if err := uc.importZIPValidateVersion(backupData); err != nil {
		return nil, err
	}
	result := &entity.ImportResult{Success: true}
	if err := uc.importZIPRunTx(ctx, backupData, opts); err != nil {
		return nil, err
	}
	if len(backupData.Files) > 0 {
		fileErrors := uc.importFilesToStorage(ctx, zr, backupData.Files, opts)
		if len(fileErrors) > 0 {
			result.Errors = fileErrors
			result.SkippedCount = len(fileErrors)
		}
	}
	uc.deps.Logger.Info("BackupUseCase - ImportZIP - completed", map[string]any{
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
			return nil, usecaseutil.Wrap(err, "BackupUseCase - ImportZIP - open backup.json")
		}
		backupData := &entity.BackupData{}
		if err := json.NewDecoder(rc).Decode(backupData); err != nil {
			_ = rc.Close()
			return nil, usecaseutil.Wrap(err, "BackupUseCase - ImportZIP - decode backup.json")
		}
		_ = rc.Close()
		return backupData, nil
	}
	return nil, fmt.Errorf("BackupUseCase - ImportZIP: backup.json not found in zip")
}

func (uc *BackupUseCase) importZIPValidateVersion(backupData *entity.BackupData) error {
	if backupData.Version != entity.BackupVersion {
		return fmt.Errorf("BackupUseCase - ImportZIP: unsupported backup version %s (expected %s)", backupData.Version, entity.BackupVersion)
	}
	return nil
}

func (uc *BackupUseCase) importZIPRunTx(ctx context.Context, backupData *entity.BackupData, opts entity.ImportOptions) error {
	return uc.deps.TxRepo.RunTransaction(ctx, func(ctx context.Context, tx repo.Transaction) error {
		if opts.EraseExisting {
			if err := uc.deps.BackupRepo.EraseAllTablesTx(ctx, tx); err != nil {
				return usecaseutil.Wrap(err, "BackupUseCase - ImportZIP - EraseAllTablesTx")
			}
		}
		return uc.importZIPRunTxImports(ctx, tx, backupData, opts)
	})
}

func (uc *BackupUseCase) importZIPRunTxImports(ctx context.Context, tx repo.Transaction, backupData *entity.BackupData, opts entity.ImportOptions) error {
	if err := uc.deps.BackupRepo.ImportCompetitionTx(ctx, tx, backupData.Competition); err != nil {
		return usecaseutil.Wrap(err, "BackupUseCase - ImportZIP - ImportCompetitionTx")
	}
	if err := uc.deps.BackupRepo.ImportChallengesTx(ctx, tx, backupData); err != nil {
		return usecaseutil.Wrap(err, "BackupUseCase - ImportZIP - ImportChallengesTx")
	}
	if err := uc.deps.BackupRepo.ImportTeamsTx(ctx, tx, backupData, opts); err != nil {
		return usecaseutil.Wrap(err, "BackupUseCase - ImportZIP - ImportTeamsTx")
	}
	if err := uc.deps.BackupRepo.ImportUsersTx(ctx, tx, backupData, opts); err != nil {
		return usecaseutil.Wrap(err, "BackupUseCase - ImportZIP - ImportUsersTx")
	}
	if err := uc.deps.BackupRepo.ImportAwardsTx(ctx, tx, backupData); err != nil {
		return usecaseutil.Wrap(err, "BackupUseCase - ImportZIP - ImportAwardsTx")
	}
	if err := uc.deps.BackupRepo.ImportSolvesTx(ctx, tx, backupData); err != nil {
		return usecaseutil.Wrap(err, "BackupUseCase - ImportZIP - ImportSolvesTx")
	}
	if err := uc.deps.BackupRepo.ImportFileMetadataTx(ctx, tx, backupData); err != nil {
		return usecaseutil.Wrap(err, "BackupUseCase - ImportZIP - ImportFileMetadataTx")
	}
	return nil
}

const maxConcurrentFileUploads = 5

func (uc *BackupUseCase) importFilesToStorage(ctx context.Context, zr *zip.Reader, files []entity.File, opts entity.ImportOptions) []string {
	fileMap := uc.importFilesBuildFileMap(files)
	tasks := uc.importFilesBuildTasks(zr, fileMap)
	var mu sync.Mutex
	var errors []string
	var uploaded int
	sem := make(chan struct{}, maxConcurrentFileUploads)
	var wg sync.WaitGroup
	for _, t := range tasks {
		wg.Add(1)
		go func(zf *zip.File, file entity.File) {
			defer wg.Done()
			errMsg := uc.importFileUploadOne(ctx, zf, file, opts, sem)
			if errMsg != "" {
				mu.Lock()
				errors = append(errors, errMsg)
				mu.Unlock()
			} else {
				mu.Lock()
				uploaded++
				mu.Unlock()
			}
		}(t.zf, t.file)
	}
	wg.Wait()
	if len(errors) > 0 {
		uc.deps.Logger.Warn("BackupUseCase - importFilesToStorage - completed with errors", map[string]any{
			"total":    len(files),
			"uploaded": uploaded,
			"errors":   len(errors),
		})
	}
	return errors
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

func (uc *BackupUseCase) importFileUploadOne(ctx context.Context, zf *zip.File, file entity.File, opts entity.ImportOptions, sem chan struct{}) string {
	select {
	case sem <- struct{}{}:
		defer func() { <-sem }()
	case <-ctx.Done():
		return fmt.Sprintf("canceled: %s", zf.Name)
	}
	rc, err := zf.Open()
	if err != nil {
		return fmt.Sprintf("open %s: %v", zf.Name, err)
	}
	defer rc.Close()
	size := zipSizeToInt64(zf.UncompressedSize64)
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
			uc.deps.Logger.Warn("BackupUseCase - importFileUploadWithHash - delete after mismatch", map[string]any{"location": file.Location, "error": delErr.Error()})
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
