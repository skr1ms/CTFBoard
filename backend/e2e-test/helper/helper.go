package helper

import (
	"testing"

	"github.com/gavv/httpexpect/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skr1ms/CTFBoard/internal/openapi"
	"github.com/stretchr/testify/require"
)

type E2EHelper struct {
	t       *testing.T
	e       *httpexpect.Expect
	client  *openapi.ClientWithResponses
	pool    *pgxpool.Pool
	baseURL string
}

func (h *E2EHelper) Pool() *pgxpool.Pool {
	return h.pool
}

func NewE2EHelper(t *testing.T, e *httpexpect.Expect, pool *pgxpool.Pool, baseURL string) *E2EHelper {
	t.Helper()
	client, err := openapi.NewClientWithResponses(baseURL + "/api/v1")
	require.NoError(t, err)
	return &E2EHelper{
		t:       t,
		e:       e,
		client:  client,
		pool:    pool,
		baseURL: baseURL,
	}
}

func (h *E2EHelper) Client() *openapi.ClientWithResponses {
	return h.client
}
