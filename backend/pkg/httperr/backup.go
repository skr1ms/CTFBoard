package httperr

import (
	"errors"
	"net/http"
)

var (
	ErrBackupJSONNotFound       = New(errors.New("backup.json not found in zip"), http.StatusBadRequest, "BACKUP_JSON_NOT_FOUND")
	ErrBackupVersionUnsupported = New(errors.New("unsupported backup version"), http.StatusBadRequest, "BACKUP_VERSION_UNSUPPORTED")
	ErrBackupTableUnsupported   = New(errors.New("unsupported backup table"), http.StatusBadRequest, "BACKUP_TABLE_UNSUPPORTED")
	ErrBackupCSVEmpty           = New(errors.New("csv file is empty"), http.StatusBadRequest, "BACKUP_CSV_EMPTY")
)
