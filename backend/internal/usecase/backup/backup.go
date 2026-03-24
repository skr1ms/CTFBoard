package backup

import (
	"context"
	"fmt"

	"github.com/wahrwelt-kit/go-logkit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/storage"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

const defaultCSVExportMaxRows = 100_000

type BackupUseCase struct {
	deps BackupDeps
}

type BackupDeps struct {
	CompetitionRepo repo.CompetitionRepository
	ChallengeRepo   repo.ChallengeRepository
	TagRepo         repo.TagRepository
	HintRepo        repo.HintRepository
	TeamRepo        repo.TeamRepository
	UserRepo        repo.UserRepository
	AwardRepo       repo.AwardRepository
	SolveRepo       repo.SolveRepository
	SubmissionRepo  repo.SubmissionRepository
	FileRepo        repo.FileRepository
	BackupRepo      repo.BackupRepository
	SettingsRepo    repo.SettingsRepository
	AuditLogRepo    repo.AuditLogRepository
	BracketRepo     repo.BracketRepository
	CommentRepo     repo.CommentRepository
	FieldRepo       repo.FieldRepository
	FieldValueRepo  repo.FieldValueRepository
	RatingRepo      repo.RatingRepository
	Storage         storage.Provider
	TM              repo.TransactionManager
	Logger          logkit.Logger
}

var _ usecase.BackupUseCase = (*BackupUseCase)(nil)

func NewBackupUseCase(deps BackupDeps) *BackupUseCase {
	if deps.Logger == nil {
		deps.Logger = logkit.Noop()
	}
	return &BackupUseCase{deps: deps}
}

func (uc *BackupUseCase) Reset(ctx context.Context, opts domain.AdminResetOptions) error {
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
	cws, err := uc.deps.ChallengeRepo.GetAllForBackup(ctx)
	if err != nil {
		return nil, fmt.Errorf("BackupUseCase - ExportCSV - ChallengeRepo.GetAllForBackup: %w", err)
	}
	challenges := make([]*domain.Challenge, len(cws))
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
	solves, err := uc.deps.SolveRepo.GetAllForBackup(ctx)
	if err != nil {
		return nil, fmt.Errorf("BackupUseCase - ExportCSV - SolveRepo.GetAllForBackup: %w", err)
	}
	return csvExportSolves(solves)
}

func (uc *BackupUseCase) exportCSVAwards(ctx context.Context) ([]byte, error) {
	awards, err := uc.deps.AwardRepo.GetAllForBackup(ctx)
	if err != nil {
		return nil, fmt.Errorf("BackupUseCase - ExportCSV - AwardRepo.GetAllForBackup: %w", err)
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

	if tableName == "users" {
		rows = csvNormalizeUserRoles(header, rows)
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
