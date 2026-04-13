package webapi

import (
	"net/http"
	"time"
)

const defaultOAuthTimeout = 10 * time.Second

var defaultOAuthClient = &http.Client{Timeout: defaultOAuthTimeout}
