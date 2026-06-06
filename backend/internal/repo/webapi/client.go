package webapi

import (
	"net/http"
	"time"
)

const defaultOAuthTimeout = 10 * time.Second

func NewOAuthHTTPClient() *http.Client {
	return &http.Client{Timeout: defaultOAuthTimeout}
}
