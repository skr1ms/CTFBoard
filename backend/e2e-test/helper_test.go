package e2e_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/gavv/httpexpect/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skr1ms/CTFBoard/internal/openapi"
	"github.com/stretchr/testify/require"
)

type E2EHelper struct {
	t      *testing.T
	e      *httpexpect.Expect
	client *openapi.ClientWithResponses
	pool   *pgxpool.Pool
}

func NewE2EHelper(t *testing.T, e *httpexpect.Expect, pool *pgxpool.Pool) *E2EHelper {
	t.Helper()
	client, err := openapi.NewClientWithResponses(GetTestBaseURL() + "/api/v1")
	require.NoError(t, err)
	return &E2EHelper{
		t:      t,
		e:      e,
		client: client,
		pool:   pool,
	}
}

func (h *E2EHelper) RegisterUserAndLogin(username string) (email, password, token string) {
	h.t.Helper()
	email = username + "@example.com"
	password = "password123"
	h.Register(username, email, password)
	resp := h.Login(email, password, http.StatusOK)
	require.NotNil(h.t, resp.JSON200)
	token = "Bearer " + *resp.JSON200.AccessToken
	return email, password, token
}

func (h *E2EHelper) RegisterUser(username string) (email, password string) {
	h.t.Helper()
	email = username + "@example.com"
	password = "password123"
	h.Register(username, email, password)
	return email, password
}

func (h *E2EHelper) RegisterAdmin(username string) (email, password, token string) {
	h.t.Helper()
	email, password, token = h.RegisterUserAndLogin(username)
	meResp := h.MeWithClient(context.Background(), h.client, token)
	require.Equal(h.t, http.StatusOK, meResp.StatusCode())
	require.NotNil(h.t, meResp.JSON200)
	require.NotNil(h.t, meResp.JSON200.ID)
	userID := *meResp.JSON200.ID
	_, err := h.pool.Exec(context.Background(), "UPDATE users SET role = 'admin' WHERE ID = $1", userID)
	require.NoError(h.t, err)
	resp := h.Login(email, password, http.StatusOK)
	require.NotNil(h.t, resp.JSON200)
	token = "Bearer " + *resp.JSON200.AccessToken
	return email, password, token
}

func (h *E2EHelper) SetupCompetition(adminNamePrefix string) (string, string) {
	h.t.Helper()
	suffix := time.Now().Format("150405")
	username := adminNamePrefix + "_" + suffix
	_, _, token := h.RegisterAdmin(username)
	h.StartCompetition(token)
	return username, token
}
