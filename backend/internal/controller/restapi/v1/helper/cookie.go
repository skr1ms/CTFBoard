package helper

import (
	"net/http"
	"net/url"
	"strings"
)

type CookieOptions struct {
	Name     string
	Value    string
	Path     string
	MaxAge   int
	Secure   bool
	SameSite http.SameSite
}

func SetHTTPOnlyCookie(w http.ResponseWriter, opts CookieOptions) {
	http.SetCookie(w, &http.Cookie{
		Name:     opts.Name,
		Value:    opts.Value,
		Path:     opts.Path,
		MaxAge:   opts.MaxAge,
		HttpOnly: true,
		Secure:   opts.Secure,
		SameSite: opts.SameSite,
	})
}

func ClearHTTPOnlyCookie(w http.ResponseWriter, name, path string, secure bool, sameSite http.SameSite) {
	SetHTTPOnlyCookie(w, CookieOptions{
		Name:     name,
		Path:     path,
		MaxAge:   -1,
		Secure:   secure,
		SameSite: sameSite,
	})
}

func RedirectFound(w http.ResponseWriter, r *http.Request, target string) {
	http.Redirect(w, r, target, http.StatusFound)
}

func FrontendCallbackURL(frontendURL string, q url.Values) string {
	return strings.TrimRight(frontendURL, "/") + "/auth/callback?" + q.Encode()
}
