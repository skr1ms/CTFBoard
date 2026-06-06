package wire

import (
	"net"
	"net/http"
	"slices"
	"strings"

	"github.com/wahrwelt-kit/go-httpkit/httputil"
	kitMiddleware "github.com/wahrwelt-kit/go-httpkit/httputil/middleware"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/errmap"
)

func metricsAllowlistMiddleware(allowedIPs []string, next http.Handler) http.Handler {
	nets := make([]*net.IPNet, 0, len(allowedIPs))

	ips := make([]net.IP, 0, len(allowedIPs))
	for _, s := range allowedIPs {
		if strings.Contains(s, "/") {
			_, n, err := net.ParseCIDR(s)
			if err != nil {
				continue
			}

			nets = append(nets, n)
		} else {
			ip := net.ParseIP(s)
			if ip != nil {
				ips = append(ips, ip)
			}
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIP := kitMiddleware.GetClientIPFromContext(r.Context())

		ip := net.ParseIP(clientIP)
		if ip == nil {
			httputil.HandleError(w, r, errmap.MapAppError(apperr.ErrAccessDenied))

			return
		}

		for _, n := range nets {
			if n.Contains(ip) {
				next.ServeHTTP(w, r)

				return
			}
		}

		if slices.ContainsFunc(ips, ip.Equal) {
			next.ServeHTTP(w, r)

			return
		}

		httputil.HandleError(w, r, errmap.MapAppError(apperr.ErrAccessDenied))
	})
}

func isSwaggerOrOpenAPIDocPath(path string) bool {
	switch path {
	case "/openapi.json", "/api/v1/openapi.json", "/swagger", "/api/v1/swagger":
		return true
	default:
		return strings.HasPrefix(path, "/swagger/") || strings.HasPrefix(path, "/api/v1/swagger/")
	}
}
