package repo

import (
	"context"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

// =============================================================================
// Audit log
// =============================================================================

type (
	// AuditLogRepository records admin audit events for compliance and traceability.
	AuditLogRepository interface {
		Create(ctx context.Context, log *domain.AuditLog) error
	}
)

// =============================================================================
// Backup
// =============================================================================

type (
	// BackupRepository provides bulk import/export and erase operations used by the backup subsystem.
	BackupRepository interface {
		EraseAllTables(ctx context.Context) error
		EraseTables(ctx context.Context, tables []string) error
		ErasePages(ctx context.Context) error
		ImportCompetition(ctx context.Context, comp *domain.Competition) error
		ImportTags(ctx context.Context, data *domain.BackupData) error
		ImportTopics(ctx context.Context, data *domain.BackupData) error
		ImportPages(ctx context.Context, data *domain.BackupData) error
		ImportChallenges(ctx context.Context, data *domain.BackupData) error
		ImportChallengeTags(ctx context.Context, data *domain.BackupData) error
		ImportChallengeTopics(ctx context.Context, data *domain.BackupData) error
		ImportUsers(ctx context.Context, data *domain.BackupData, opts domain.ImportOptions) error
		ImportTeams(ctx context.Context, data *domain.BackupData, opts domain.ImportOptions) error
		UpdateUserTeamIDs(ctx context.Context, data *domain.BackupData) error
		ImportAwards(ctx context.Context, data *domain.BackupData) error
		ImportSolves(ctx context.Context, data *domain.BackupData) error
		ImportHintUnlocks(ctx context.Context, data *domain.BackupData) error
		ImportFileMetadata(ctx context.Context, data *domain.BackupData) error
		ImportBrackets(ctx context.Context, data *domain.BackupData) error
		ImportChallengeRequirements(ctx context.Context, data *domain.BackupData) error
		ImportSolutions(ctx context.Context, data *domain.BackupData) error
		ImportRatings(ctx context.Context, data *domain.BackupData) error
		ImportComments(ctx context.Context, data *domain.BackupData) error
		ImportFields(ctx context.Context, data *domain.BackupData) error
		ImportFieldValues(ctx context.Context, data *domain.BackupData) error
		ImportCSV(ctx context.Context, tableName string, header []string, rows [][]string) (int, []string, error)
		CreateImportJob(ctx context.Context, job *domain.ImportJob) (*domain.ImportJob, error)
		GetImportJob(ctx context.Context, id uuid.UUID) (*domain.ImportJob, error)
		MarkImportJobRunning(ctx context.Context, id uuid.UUID, phase domain.ImportJobPhase) (*domain.ImportJob, error)
		UpdateImportJobPhase(ctx context.Context, id uuid.UUID, phase domain.ImportJobPhase) error
		CompleteImportJob(ctx context.Context, id uuid.UUID, result *domain.ImportResult) error
		FailImportJob(ctx context.Context, id uuid.UUID, message string) error
		FailInterruptedImportJobs(ctx context.Context) error
	}
)
