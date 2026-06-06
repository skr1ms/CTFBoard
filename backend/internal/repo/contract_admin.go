package repo

import (
	"context"

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
		ImportCompetition(ctx context.Context, comp *domain.Competition) error
		ImportTags(ctx context.Context, data *domain.BackupData) error
		ImportChallenges(ctx context.Context, data *domain.BackupData) error
		ImportChallengeTags(ctx context.Context, data *domain.BackupData) error
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
	}
)
