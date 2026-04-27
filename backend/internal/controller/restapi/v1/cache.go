package v1

import "net/http"

const (
	cacheStatic = "public, max-age=600, stale-while-revalidate=60"
	cacheShort  = "public, max-age=30, stale-while-revalidate=10"
	cacheMedium = "public, max-age=120, stale-while-revalidate=30"
	cacheMicro  = "public, max-age=2"
)

// setPublicCache writes Cache-Control and Vary headers for publicly cacheable
// endpoints. Pass varyAuth=true for endpoints behind optional authentication
// (e.g. scoreboard) so the edge cache keys separately on Authorization.
func setPublicCache(w http.ResponseWriter, maxAge string, varyAuth bool) {
	w.Header().Set("Cache-Control", maxAge)

	if varyAuth {
		w.Header().Set("Vary", "Accept-Encoding, Authorization")
	} else {
		w.Header().Set("Vary", "Accept-Encoding")
	}
}
