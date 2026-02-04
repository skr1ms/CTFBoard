package entityError

import (
	"errors"
	"net/http"
)

var ErrAppSettingsNotFound = &HTTPError{
	Err:        errors.New("app settings not found"),
	StatusCode: http.StatusNotFound,
	Code:       "APP_SETTINGS_NOT_FOUND",
}

var ErrConfigNotFound = &HTTPError{
	Err:        errors.New("config not found"),
	StatusCode: http.StatusNotFound,
	Code:       "CONFIG_NOT_FOUND",
}
