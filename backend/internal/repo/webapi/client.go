package webapi

import (
	"net/http"
	"time"
)

var defaultOAuthClient = &http.Client{Timeout: 10 * time.Second}
