package helper

import (
	"io"
	"net/http"
)

const trustedHTMLContentSecurityPolicy = "default-src 'none'; img-src https: data:; style-src 'unsafe-inline'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'"

// RenderTrustedHTML renders server-owned HTML documents. Do not use it for user
// supplied HTML; templates must keep html/template escaping enabled.
func RenderTrustedHTML(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", trustedHTMLContentSecurityPolicy)
	w.WriteHeader(status)
	_, _ = io.WriteString(w, body)
}
