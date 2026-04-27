package helper

import (
	"net/http"
	"net/http/cookiejar"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

type E2EHelper struct {
	t       *testing.T
	client  *openapi.ClientWithResponses
	pool    *pgxpool.Pool
	redis   *redis.Client
	baseURL string
}

func (h *E2EHelper) Pool() *pgxpool.Pool {
	return h.pool
}

func (h *E2EHelper) Redis() *redis.Client {
	return h.redis
}

func NewE2EHelper(t *testing.T, _ any, pool *pgxpool.Pool, redisClient *redis.Client, baseURL string) *E2EHelper {
	t.Helper()

	// Cookie jar is required so the httpOnly ctf_refresh cookie set by login/refresh
	// is automatically sent to refresh and logout endpoints.
	jar, err := cookiejar.New(nil)
	require.NoError(t, err)

	httpClient := &http.Client{Jar: jar}

	client, err := openapi.NewClientWithResponses(baseURL+"/api/v1", openapi.WithHTTPClient(httpClient))
	require.NoError(t, err)

	return &E2EHelper{
		t:       t,
		client:  client,
		pool:    pool,
		redis:   redisClient,
		baseURL: baseURL,
	}
}

func (h *E2EHelper) Client() *openapi.ClientWithResponses {
	return h.client
}
