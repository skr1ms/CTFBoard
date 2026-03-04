package httperr

import (
	"errors"
	"net/http"
)

var (
	ErrBackupJSONNotFound = &HTTPError{
		Err:        errors.New("backup.json not found in zip"),
		StatusCode: http.StatusBadRequest,
		Code:       "BACKUP_JSON_NOT_FOUND",
	}
	ErrBackupVersionUnsupported = &HTTPError{
		Err:        errors.New("unsupported backup version"),
		StatusCode: http.StatusBadRequest,
		Code:       "BACKUP_VERSION_UNSUPPORTED",
	}
	ErrBackupTableUnsupported = &HTTPError{
		Err:        errors.New("unsupported backup table"),
		StatusCode: http.StatusBadRequest,
		Code:       "BACKUP_TABLE_UNSUPPORTED",
	}
	ErrBackupCSVEmpty = &HTTPError{
		Err:        errors.New("csv file is empty"),
		StatusCode: http.StatusBadRequest,
		Code:       "BACKUP_CSV_EMPTY",
	}
)
