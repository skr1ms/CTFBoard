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
	"github.com/jackc/pgx/v5"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/repo"
	"github.com/skr1ms/CTFBoard/internal/storage"
	"github.com/skr1ms/CTFBoard/pkg/logger"
	"golang.org/x/sync/errgroup"
)

type BackupUseCase struct {
	competitionRepo repo.CompetitionRepository
	challengeRepo   repo.ChallengeRepository
	hintRepo        repo.HintRepository
	teamRepo        repo.TeamRepository
	userRepo        repo.UserRepository
	awardRepo       repo.AwardRepository
	solveRepo       repo.SolveRepository
	fileRepo        repo.FileRepository
	backupRepo      repo.BackupRepository
	storage         storage.Provider
	txRepo          repo.TxRepository
	logger          logger.Logger
}

func NewBackupUseCase(
	competitionRepo repo.CompetitionRepository,
	challengeRepo repo.ChallengeRepository,
	hintRepo repo.HintRepository,
	teamRepo repo.TeamRepository,
	userRepo repo.UserRepository,
	awardRepo repo.AwardRepository,
	solveRepo repo.SolveRepository,
	fileRepo repo.FileRepository,
	backupRepo repo.BackupRepository,
	storageProvider storage.Provider,
	txRepo repo.TxRepository,
	l logger.Logger,
) *BackupUseCase {
	return &BackupUseCase{
		competitionRepo: competitionRepo,
		challengeRepo:   challengeRepo,
		hintRepo:        hintRepo,
		teamRepo:        teamRepo,
		userRepo:        userRepo,
		awardRepo:       awardRepo,
		solveRepo:       solveRepo,
		fileRepo:        fileRepo,
		backupRepo:      backupRepo,
		storage:         storageProvider,
		txRepo:          txRepo,
		logger:          l,
	}
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
		comp, err := uc.competitionRepo.Get(gCtx)
		if err != nil {
			return fmt.Errorf("BackupUseCase - Export - Get: %w", err)
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
		return nil, err
	}
	return backup, nil
}

//nolint:gocognit
func (uc *BackupUseCase) exportOptional(
	ctx context.Context,
	backup *entity.BackupData,
	opts entity.ExportOptions,
	mu *sync.Mutex,
	g *errgroup.Group,
) {
	if opts.IncludeTeams {
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
	if opts.IncludeUsers {
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
	if opts.IncludeAwards {
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
	if opts.IncludeSolves {
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
	if opts.IncludeFiles {
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
}

func (uc *BackupUseCase) fetchChallengesWithHints(ctx context.Context) ([]entity.ChallengeExport, error) {
	challengesWithSolved, err := uc.challengeRepo.GetAll(ctx, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("BackupUseCase - fetchChallengesWithHints - GetAll: %w", err)
	}

	result := make([]entity.ChallengeExport, 0, len(challengesWithSolved))
	for _, cws := range challengesWithSolved {
		hints, err := uc.hintRepo.GetByChallengeID(ctx, cws.Challenge.ID)
		if err != nil {
			return nil, fmt.Errorf("BackupUseCase - fetchChallengesWithHints - GetByChallengeID: %w", err)
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
	teams, err := uc.teamRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("BackupUseCase - fetchTeamsWithMembers - GetAll: %w", err)
	}

	result := make([]entity.TeamExport, 0, len(teams))
	for _, team := range teams {
		members, err := uc.userRepo.GetByTeamID(ctx, team.ID)
		if err != nil {
			return nil, fmt.Errorf("BackupUseCase - fetchTeamsWithMembers - GetByTeamID: %w", err)
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
	users, err := uc.userRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("BackupUseCase - fetchUsers - GetAll: %w", err)
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
	awards, err := uc.awardRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("BackupUseCase - fetchAwards - GetAll: %w", err)
	}

	result := make([]entity.Award, len(awards))
	for i, a := range awards {
		result[i] = *a
	}

	return result, nil
}

func (uc *BackupUseCase) fetchSolves(ctx context.Context) ([]entity.Solve, error) {
	solves, err := uc.solveRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("BackupUseCase - fetchSolves - GetAll: %w", err)
	}

	result := make([]entity.Solve, len(solves))
	for i, s := range solves {
		result[i] = *s
	}

	return result, nil
}

func (uc *BackupUseCase) fetchFiles(ctx context.Context) ([]entity.File, error) {
	files, err := uc.fileRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("BackupUseCase - fetchFiles - GetAll: %w", err)
	}

	result := make([]entity.File, len(files))
	for i, f := range files {
		result[i] = *f
	}

	return result, nil
}

func (uc *BackupUseCase) ExportZIP(ctx context.Context, opts entity.ExportOptions) (io.ReadCloser, error) {
	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()
		zw := zip.NewWriter(pw)
		defer zw.Close()

		data, err := uc.Export(ctx, opts)
		if err != nil {
			pw.CloseWithError(err)
			return
		}

		jsonFile, err := zw.Create("backup.json")
		if err != nil {
			pw.CloseWithError(fmt.Errorf("BackupUseCase - ExportZIP - create backup.json: %w", err))
			return
		}
		if err := json.NewEncoder(jsonFile).Encode(data); err != nil {
			pw.CloseWithError(fmt.Errorf("BackupUseCase - ExportZIP - encode backup.json: %w", err))
			return
		}

		readme, err := zw.Create("README.md")
		if err != nil {
			pw.CloseWithError(fmt.Errorf("BackupUseCase - ExportZIP - create README.md: %w", err))
			return
		}
		_, err = fmt.Fprintf(readme, "# CTFBoard Backup\n\nBackup created: %s\nVersion: %s\n", data.ExportedAt.Format(time.RFC3339), data.Version)
		if err != nil {
			pw.CloseWithError(fmt.Errorf("BackupUseCase - ExportZIP - write README: %w", err))
			return
		}

		if opts.IncludeFiles && len(data.Files) > 0 {
			skipped := uc.streamFilesToZip(ctx, zw, data.Files)
			uc.logger.Info("BackupUseCase - ExportZIP - completed", map[string]any{
				"challenges": len(data.Challenges),
				"teams":      len(data.Teams),
				"files":      len(data.Files),
				"skipped":    skipped,
			})
		} else {
			uc.logger.Info("BackupUseCase - ExportZIP - completed", map[string]any{
				"challenges": len(data.Challenges),
				"teams":      len(data.Teams),
				"files":      0,
			})
		}
	}()

	return pr, nil
}

func (uc *BackupUseCase) streamFilesToZip(ctx context.Context, zw *zip.Writer, files []entity.File) int {
	var skipped int
	for _, file := range files {
		path := fmt.Sprintf("files/challenge-%s/%s", file.ChallengeID, file.Filename)
		f, err := zw.Create(path)
		if err != nil {
			uc.logger.Warn("BackupUseCase - streamFilesToZip - create", map[string]any{
				"file":  file.Filename,
				"error": err.Error(),
			})
			skipped++
			continue
		}

		rc, err := uc.storage.Download(ctx, file.Location)
		if err != nil {
			uc.logger.Warn("BackupUseCase - streamFilesToZip - download", map[string]any{
				"file":     file.Filename,
				"location": file.Location,
				"error":    err.Error(),
			})
			skipped++
			continue
		}

		if _, err := io.Copy(f, rc); err != nil {
			uc.logger.Warn("BackupUseCase - streamFilesToZip - copy", map[string]any{
				"file":  file.Filename,
				"error": err.Error(),
			})
			skipped++
		}
		_ = rc.Close()
	}

	if skipped > 0 {
		uc.logger.Warn("BackupUseCase - streamFilesToZip - completed with skipped files", map[string]any{
			"total":   len(files),
			"skipped": skipped,
		})
	}

	return skipped
}

//nolint:gocognit,gocyclo,funlen
func (uc *BackupUseCase) ImportZIP(ctx context.Context, r io.ReaderAt, size int64, opts entity.ImportOptions) (*entity.ImportResult, error) {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return nil, fmt.Errorf("BackupUseCase - ImportZIP - NewReader: %w", err)
	}

	var backupData *entity.BackupData
	for _, f := range zr.File {
		if f.Name == "backup.json" {
			rc, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("BackupUseCase - ImportZIP - open backup.json: %w", err)
			}
			backupData = &entity.BackupData{}
			if err := json.NewDecoder(rc).Decode(backupData); err != nil {
				_ = rc.Close()
				return nil, fmt.Errorf("BackupUseCase - ImportZIP - decode backup.json: %w", err)
			}
			_ = rc.Close()
			break
		}
	}
	if backupData == nil {
		return nil, fmt.Errorf("BackupUseCase - ImportZIP: backup.json not found in zip")
	}

	if backupData.Version != entity.BackupVersion {
		return nil, fmt.Errorf("BackupUseCase - ImportZIP: unsupported backup version %s (expected %s)", backupData.Version, entity.BackupVersion)
	}

	result := &entity.ImportResult{Success: true}

	err = uc.txRepo.RunTransaction(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if opts.EraseExisting {
			if err := uc.backupRepo.EraseAllTablesTx(ctx, tx); err != nil {
				return fmt.Errorf("BackupUseCase - ImportZIP - EraseAllTablesTx: %w", err)
			}
		}

		if err := uc.backupRepo.ImportCompetitionTx(ctx, tx, backupData.Competition); err != nil {
			return fmt.Errorf("BackupUseCase - ImportZIP - ImportCompetitionTx: %w", err)
		}
		if err := uc.backupRepo.ImportChallengesTx(ctx, tx, backupData); err != nil {
			return fmt.Errorf("BackupUseCase - ImportZIP - ImportChallengesTx: %w", err)
		}
		if err := uc.backupRepo.ImportTeamsTx(ctx, tx, backupData, opts); err != nil {
			return fmt.Errorf("BackupUseCase - ImportZIP - ImportTeamsTx: %w", err)
		}
		if err := uc.backupRepo.ImportUsersTx(ctx, tx, backupData, opts); err != nil {
			return fmt.Errorf("BackupUseCase - ImportZIP - ImportUsersTx: %w", err)
		}
		if err := uc.backupRepo.ImportAwardsTx(ctx, tx, backupData); err != nil {
			return fmt.Errorf("BackupUseCase - ImportZIP - ImportAwardsTx: %w", err)
		}
		if err := uc.backupRepo.ImportSolvesTx(ctx, tx, backupData); err != nil {
			return fmt.Errorf("BackupUseCase - ImportZIP - ImportSolvesTx: %w", err)
		}
		if err := uc.backupRepo.ImportFileMetadataTx(ctx, tx, backupData); err != nil {
			return fmt.Errorf("BackupUseCase - ImportZIP - ImportFileMetadataTx: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	if len(backupData.Files) > 0 {
		fileErrors := uc.importFilesToStorage(ctx, zr, backupData.Files, opts)
		if len(fileErrors) > 0 {
			result.Errors = fileErrors
			result.SkippedCount = len(fileErrors)
		}
	}

	uc.logger.Info("BackupUseCase - ImportZIP - completed", map[string]any{
		"challenges": len(backupData.Challenges),
		"teams":      len(backupData.Teams),
		"users":      len(backupData.Users),
		"files":      len(backupData.Files),
		"skipped":    result.SkippedCount,
	})

	return result, nil
}

const maxConcurrentFileUploads = 5

//nolint:gocognit
func (uc *BackupUseCase) importFilesToStorage(ctx context.Context, zr *zip.Reader, files []entity.File, opts entity.ImportOptions) []string {
	var mu sync.Mutex
	var errors []string
	var uploaded int

	fileMap := make(map[string]entity.File)
	for _, f := range files {
		path := fmt.Sprintf("files/challenge-%s/%s", f.ChallengeID, f.Filename)
		fileMap[path] = f
	}

	type task struct {
		zf   *zip.File
		file entity.File
	}
	var tasks []task
	for _, zf := range zr.File {
		file, ok := fileMap[zf.Name]
		if !ok {
			continue
		}
		tasks = append(tasks, task{zf: zf, file: file})
	}

	sem := make(chan struct{}, maxConcurrentFileUploads)
	var wg sync.WaitGroup
	for _, t := range tasks {
		wg.Add(1)
		go func(zf *zip.File, file entity.File) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				mu.Lock()
				errors = append(errors, fmt.Sprintf("cancelled: %s", zf.Name))
				mu.Unlock()
				return
			}

			rc, err := zf.Open()
			if err != nil {
				errMsg := fmt.Sprintf("open %s: %v", zf.Name, err)
				uc.logger.Warn("BackupUseCase - importFilesToStorage", map[string]any{"error": errMsg})
				mu.Lock()
				errors = append(errors, errMsg)
				mu.Unlock()
				return
			}

			if opts.ValidateFiles {
				hash := sha256.New()
				tee := io.TeeReader(rc, hash)
				size := zipSizeToInt64(zf.UncompressedSize64)
				if err := uc.storage.Upload(ctx, file.Location, tee, size, "application/octet-stream"); err != nil {
					_ = rc.Close()
					errMsg := fmt.Sprintf("upload %s: %v", zf.Name, err)
					uc.logger.Warn("BackupUseCase - importFilesToStorage", map[string]any{"error": errMsg})
					mu.Lock()
					errors = append(errors, errMsg)
					mu.Unlock()
					return
				}
				_ = rc.Close()

				hashStr := hex.EncodeToString(hash.Sum(nil))
				if hashStr != file.SHA256 {
					_ = uc.storage.Delete(ctx, file.Location) //nolint:errcheck
					errMsg := fmt.Sprintf("sha256 mismatch for %s: expected %s, got %s", zf.Name, file.SHA256, hashStr)
					uc.logger.Warn("BackupUseCase - importFilesToStorage", map[string]any{"error": errMsg})
					mu.Lock()
					errors = append(errors, errMsg)
					mu.Unlock()
					return
				}
			} else {
				size := zipSizeToInt64(zf.UncompressedSize64)
				if err := uc.storage.Upload(ctx, file.Location, rc, size, "application/octet-stream"); err != nil {
					_ = rc.Close()
					errMsg := fmt.Sprintf("upload %s: %v", zf.Name, err)
					uc.logger.Warn("BackupUseCase - importFilesToStorage", map[string]any{"error": errMsg})
					mu.Lock()
					errors = append(errors, errMsg)
					mu.Unlock()
					return
				}
				_ = rc.Close()
			}
			mu.Lock()
			uploaded++
			mu.Unlock()
		}(t.zf, t.file)
	}
	wg.Wait()

	if len(errors) > 0 {
		uc.logger.Warn("BackupUseCase - importFilesToStorage - completed with errors", map[string]any{
			"total":    len(files),
			"uploaded": uploaded,
			"errors":   len(errors),
		})
	}

	return errors
}

func zipSizeToInt64(u uint64) int64 {
	if u > math.MaxInt64 {
		return math.MaxInt64
	}
	return int64(u)
}
