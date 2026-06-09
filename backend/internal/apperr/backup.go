package apperr

import "errors"

var (
	ErrBackupJSONNotFound         = errors.New("backup.json not found in zip")
	ErrBackupVersionUnsupported   = errors.New("unsupported backup version")
	ErrBackupTableUnsupported     = errors.New("unsupported backup table")
	ErrBackupCSVEmpty             = errors.New("csv file is empty")
	ErrBackupImportJobNotFound    = errors.New("backup import job not found")
	ErrBackupImportAlreadyRunning = errors.New("backup import already running")
)
