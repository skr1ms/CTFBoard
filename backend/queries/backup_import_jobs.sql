-- name: CreateBackupImportJob :one
INSERT INTO backup_import_jobs (
    id, requested_by, client_ip, archive_filename, archive_size, staging_location, options, status, phase
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, 'queued', 'queued'
) RETURNING id, requested_by, client_ip, archive_filename, archive_size, staging_location,
          status, phase, options, result, error, created_at, started_at, finished_at, updated_at;

-- name: GetBackupImportJob :one
SELECT id, requested_by, client_ip, archive_filename, archive_size, staging_location,
       status, phase, options, result, error, created_at, started_at, finished_at, updated_at
FROM backup_import_jobs
WHERE id = $1;

-- name: MarkBackupImportJobRunning :one
UPDATE backup_import_jobs
SET status = 'running',
    phase = $2,
    started_at = COALESCE(started_at, CURRENT_TIMESTAMP),
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING id, requested_by, client_ip, archive_filename, archive_size, staging_location,
          status, phase, options, result, error, created_at, started_at, finished_at, updated_at;

-- name: UpdateBackupImportJobPhase :exec
UPDATE backup_import_jobs
SET phase = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
  AND status = 'running';

-- name: CompleteBackupImportJob :exec
UPDATE backup_import_jobs
SET status = 'completed',
    phase = 'finished',
    result = $2,
    error = NULL,
    finished_at = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: FailBackupImportJob :exec
UPDATE backup_import_jobs
SET status = 'failed',
    phase = 'finished',
    error = $2,
    finished_at = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: FailInterruptedBackupImportJobs :exec
UPDATE backup_import_jobs
SET status = 'failed',
    phase = 'finished',
    error = 'import interrupted by backend restart',
    finished_at = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
WHERE status IN ('queued', 'running');
