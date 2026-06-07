package backup

import (
	"context"
	"fmt"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

// importZIPRunTx runs the database import inside a single transaction. When
// EraseExisting is set it first writes an audit log entry and calls
// EraseAllTables to wipe all existing data before importing. The actual
// multi-table import is delegated to importZIPRunTxImports.
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

				err := uc.deps.AuditLogRepo.Create(ctx, auditEntry)
				if err != nil {
					return fmt.Errorf("BackupUseCase - ImportZIP - AuditLogRepo.Create: %w", err)
				}
			}

			err := uc.deps.BackupRepo.EraseAllTables(ctx)
			if err != nil {
				return fmt.Errorf("BackupUseCase - ImportZIP - BackupRepo.EraseAllTables: %w", err)
			}
		}

		return uc.importZIPRunTxImports(ctx, backupData, opts)
	})
}

// importZIPRunTxImports inserts all backup entities in dependency order inside
// the caller's transaction: competition settings -> tags -> topics -> challenges -> brackets ->
// users -> teams -> memberships -> awards -> solves -> hint unlocks -> files -> requirements
// -> challenge topics -> solutions -> ratings -> comments -> fields -> field values. Each step calls a repo
// method that uses ON CONFLICT DO NOTHING so re-imports are idempotent.
func (uc *BackupUseCase) importZIPRunTxImports(ctx context.Context, backupData *domain.BackupData, opts domain.ImportOptions) error {
	err := uc.deps.BackupRepo.ImportCompetition(ctx, backupData.Competition)
	if err != nil {
		return fmt.Errorf("BackupUseCase - ImportZIP - BackupRepo.ImportCompetition: %w", err)
	}

	err = uc.deps.BackupRepo.ImportTags(ctx, backupData)
	if err != nil {
		return fmt.Errorf("BackupUseCase - ImportZIP - BackupRepo.ImportTags: %w", err)
	}

	err = uc.deps.BackupRepo.ImportTopics(ctx, backupData)
	if err != nil {
		return fmt.Errorf("BackupUseCase - ImportZIP - BackupRepo.ImportTopics: %w", err)
	}

	err = uc.deps.BackupRepo.ImportChallenges(ctx, backupData)
	if err != nil {
		return fmt.Errorf("BackupUseCase - ImportZIP - BackupRepo.ImportChallenges: %w", err)
	}

	err = uc.deps.BackupRepo.ImportChallengeTags(ctx, backupData)
	if err != nil {
		return fmt.Errorf("BackupUseCase - ImportZIP - BackupRepo.ImportChallengeTags: %w", err)
	}

	err = uc.deps.BackupRepo.ImportChallengeTopics(ctx, backupData)
	if err != nil {
		return fmt.Errorf("BackupUseCase - ImportZIP - BackupRepo.ImportChallengeTopics: %w", err)
	}

	err = uc.deps.BackupRepo.ImportBrackets(ctx, backupData)
	if err != nil {
		return fmt.Errorf("BackupUseCase - ImportZIP - BackupRepo.ImportBrackets: %w", err)
	}

	uc.importNormalizeUserRoles(backupData, opts)

	err = uc.deps.BackupRepo.ImportUsers(ctx, backupData, opts)
	if err != nil {
		return fmt.Errorf("BackupUseCase - ImportZIP - BackupRepo.ImportUsers: %w", err)
	}

	err = uc.deps.BackupRepo.ImportTeams(ctx, backupData, opts)
	if err != nil {
		return fmt.Errorf("BackupUseCase - ImportZIP - BackupRepo.ImportTeams: %w", err)
	}

	err = uc.deps.BackupRepo.UpdateUserTeamIDs(ctx, backupData)
	if err != nil {
		return fmt.Errorf("BackupUseCase - ImportZIP - BackupRepo.UpdateUserTeamIDs: %w", err)
	}

	err = uc.deps.BackupRepo.ImportAwards(ctx, backupData)
	if err != nil {
		return fmt.Errorf("BackupUseCase - ImportZIP - BackupRepo.ImportAwards: %w", err)
	}

	err = uc.deps.BackupRepo.ImportSolves(ctx, backupData)
	if err != nil {
		return fmt.Errorf("BackupUseCase - ImportZIP - BackupRepo.ImportSolves: %w", err)
	}

	err = uc.deps.BackupRepo.ImportHintUnlocks(ctx, backupData)
	if err != nil {
		return fmt.Errorf("BackupUseCase - ImportZIP - BackupRepo.ImportHintUnlocks: %w", err)
	}

	err = uc.deps.BackupRepo.ImportFileMetadata(ctx, backupData)
	if err != nil {
		return fmt.Errorf("BackupUseCase - ImportZIP - BackupRepo.ImportFileMetadata: %w", err)
	}

	err = uc.deps.BackupRepo.ImportChallengeRequirements(ctx, backupData)
	if err != nil {
		return fmt.Errorf("BackupUseCase - ImportZIP - BackupRepo.ImportChallengeRequirements: %w", err)
	}

	err = uc.deps.BackupRepo.ImportSolutions(ctx, backupData)
	if err != nil {
		return fmt.Errorf("BackupUseCase - ImportZIP - BackupRepo.ImportSolutions: %w", err)
	}

	err = uc.deps.BackupRepo.ImportRatings(ctx, backupData)
	if err != nil {
		return fmt.Errorf("BackupUseCase - ImportZIP - BackupRepo.ImportRatings: %w", err)
	}

	err = uc.deps.BackupRepo.ImportComments(ctx, backupData)
	if err != nil {
		return fmt.Errorf("BackupUseCase - ImportZIP - BackupRepo.ImportComments: %w", err)
	}

	err = uc.deps.BackupRepo.ImportFields(ctx, backupData)
	if err != nil {
		return fmt.Errorf("BackupUseCase - ImportZIP - BackupRepo.ImportFields: %w", err)
	}

	err = uc.deps.BackupRepo.ImportFieldValues(ctx, backupData)
	if err != nil {
		return fmt.Errorf("BackupUseCase - ImportZIP - BackupRepo.ImportFieldValues: %w", err)
	}

	return nil
}

// importNormalizeUserRoles ensures every imported user has a valid role
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
