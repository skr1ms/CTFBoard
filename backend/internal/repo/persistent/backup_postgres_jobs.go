package persistent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
)

func toDomainImportJob(row sqlc.BackupImportJob) (*domain.ImportJob, error) {
	var opts domain.ImportOptions

	if len(row.Options) > 0 {
		if err := json.Unmarshal(row.Options, &opts); err != nil {
			return nil, fmt.Errorf("BackupRepo - toDomainImportJob - options: %w", err)
		}
	}

	opts.AdminUserID = row.RequestedBy

	if row.ClientIp != nil {
		opts.AdminIP = *row.ClientIp
	}

	var result *domain.ImportResult

	if len(row.Result) > 0 {
		var decoded domain.ImportResult

		if err := json.Unmarshal(row.Result, &decoded); err != nil {
			return nil, fmt.Errorf("BackupRepo - toDomainImportJob - result: %w", err)
		}

		result = &decoded
	}

	return &domain.ImportJob{
		ID:              row.ID,
		RequestedBy:     row.RequestedBy,
		ClientIP:        stringPtrValue(row.ClientIp),
		ArchiveFilename: row.ArchiveFilename,
		ArchiveSize:     row.ArchiveSize,
		StagingLocation: row.StagingLocation,
		Options:         opts,
		Status:          domain.ImportJobStatus(row.Status),
		Phase:           domain.ImportJobPhase(row.Phase),
		Result:          result,
		Error:           row.Error,
		CreatedAt:       timestamptzTime(row.CreatedAt),
		StartedAt:       timestamptzTimePtr(row.StartedAt),
		FinishedAt:      timestamptzTimePtr(row.FinishedAt),
		UpdatedAt:       timestamptzTime(row.UpdatedAt),
	}, nil
}

func timestamptzTime(ts pgtype.Timestamptz) time.Time {
	if !ts.Valid {
		return time.Time{}
	}

	return ts.Time
}

func timestamptzTimePtr(ts pgtype.Timestamptz) *time.Time {
	if !ts.Valid {
		return nil
	}

	return &ts.Time
}

func stringPtrValue(v *string) string {
	if v == nil {
		return ""
	}

	return *v
}

func stringPtrNonEmpty(v string) *string {
	if v == "" {
		return nil
	}

	return &v
}

func (r *BackupRepo) CreateImportJob(ctx context.Context, job *domain.ImportJob) (*domain.ImportJob, error) {
	options, err := json.Marshal(job.Options)
	if err != nil {
		return nil, fmt.Errorf("BackupRepo - CreateImportJob - options: %w", err)
	}

	row, err := r.Q(ctx).CreateBackupImportJob(ctx, sqlc.CreateBackupImportJobParams{
		ID:              job.ID,
		RequestedBy:     job.RequestedBy,
		ClientIp:        stringPtrNonEmpty(job.ClientIP),
		ArchiveFilename: job.ArchiveFilename,
		ArchiveSize:     job.ArchiveSize,
		StagingLocation: job.StagingLocation,
		Options:         options,
	})
	if err != nil {
		return nil, fmt.Errorf("BackupRepo - CreateImportJob: %w", err)
	}

	return toDomainImportJob(row)
}

func (r *BackupRepo) GetImportJob(ctx context.Context, id uuid.UUID) (*domain.ImportJob, error) {
	row, err := GetOrNotFound(
		func() (sqlc.BackupImportJob, error) { return r.Q(ctx).GetBackupImportJob(ctx, id) },
		apperr.ErrBackupImportJobNotFound,
		"BackupRepo - GetImportJob",
	)
	if err != nil {
		return nil, err
	}

	return toDomainImportJob(row)
}

func (r *BackupRepo) MarkImportJobRunning(ctx context.Context, id uuid.UUID, phase domain.ImportJobPhase) (*domain.ImportJob, error) {
	row, err := r.Q(ctx).MarkBackupImportJobRunning(ctx, sqlc.MarkBackupImportJobRunningParams{
		ID:    id,
		Phase: string(phase),
	})
	if err != nil {
		return nil, fmt.Errorf("BackupRepo - MarkImportJobRunning: %w", err)
	}

	return toDomainImportJob(row)
}

func (r *BackupRepo) UpdateImportJobPhase(ctx context.Context, id uuid.UUID, phase domain.ImportJobPhase) error {
	if err := r.Q(ctx).UpdateBackupImportJobPhase(ctx, sqlc.UpdateBackupImportJobPhaseParams{
		ID:    id,
		Phase: string(phase),
	}); err != nil {
		return fmt.Errorf("BackupRepo - UpdateImportJobPhase: %w", err)
	}

	return nil
}

func (r *BackupRepo) CompleteImportJob(ctx context.Context, id uuid.UUID, result *domain.ImportResult) error {
	encoded, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("BackupRepo - CompleteImportJob - result: %w", err)
	}

	if err := r.Q(ctx).CompleteBackupImportJob(ctx, sqlc.CompleteBackupImportJobParams{
		ID:     id,
		Result: encoded,
	}); err != nil {
		return fmt.Errorf("BackupRepo - CompleteImportJob: %w", err)
	}

	return nil
}

func (r *BackupRepo) FailImportJob(ctx context.Context, id uuid.UUID, message string) error {
	if err := r.Q(ctx).FailBackupImportJob(ctx, sqlc.FailBackupImportJobParams{
		ID:    id,
		Error: &message,
	}); err != nil {
		return fmt.Errorf("BackupRepo - FailImportJob: %w", err)
	}

	return nil
}

func (r *BackupRepo) FailInterruptedImportJobs(ctx context.Context) error {
	if err := r.Q(ctx).FailInterruptedBackupImportJobs(ctx); err != nil {
		return fmt.Errorf("BackupRepo - FailInterruptedImportJobs: %w", err)
	}

	return nil
}
