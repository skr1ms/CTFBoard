package helper

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/skr1ms/CTFBoard/internal/openapi"
	"github.com/stretchr/testify/require"
)

func getStr(m map[string]any, key, def string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return def
}

func getInt(m map[string]any, key string) int {
	switch v := m[key].(type) {
	case int:
		return v
	case float64:
		return int(v)
	}
	return 0
}

func getIntPtr(m map[string]any, key string) *int {
	switch v := m[key].(type) {
	case int:
		return &v
	case float64:
		i := int(v)
		return &i
	}
	return nil
}

func getStrSlice(m map[string]any, key string) []string {
	v, ok := m[key]
	if !ok {
		return nil
	}
	switch arr := v.(type) {
	case []string:
		return arr
	case []any:
		out := make([]string, 0, len(arr))
		for _, x := range arr {
			if s, ok := x.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

func GetAPIClient(t *testing.T, baseURL string) *openapi.ClientWithResponses {
	t.Helper()
	client, err := openapi.NewClientWithResponses(baseURL + "/api/v1")
	require.NoError(t, err)
	return client
}

func WithBearerToken(token string) openapi.RequestEditorFn {
	return func(_ context.Context, req *http.Request) error {
		if token != "" && !strings.HasPrefix(token, "Bearer ") {
			token = "Bearer " + token
		}
		req.Header.Set("Authorization", token)
		return nil
	}
}

func RequireStatus(t *testing.T, expect, actual int, body []byte, label string) {
	t.Helper()
	require.Equal(t, expect, actual, "%s: %s", label, body)
}

func RequireRegisterCreated(t *testing.T, resp *openapi.PostAuthRegisterResponse) {
	t.Helper()
	RequireStatus(t, http.StatusCreated, resp.StatusCode(), resp.Body, "register")
}

func RequireLoginOK(t *testing.T, resp *openapi.PostAuthLoginResponse) string {
	t.Helper()
	RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "login")
	require.NotNil(t, resp.JSON200)
	return *resp.JSON200.AccessToken
}

func RequireMeOK(t *testing.T, resp *openapi.GetAuthMeResponse) *openapi.ResponseMeResponse {
	t.Helper()
	RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "me")
	require.NotNil(t, resp.JSON200)
	return resp.JSON200
}

func RequireConflict(t *testing.T, resp *openapi.PostAuthRegisterResponse, label string) {
	t.Helper()
	RequireStatus(t, http.StatusConflict, resp.StatusCode(), resp.Body, label)
	require.NotNil(t, resp.JSON409)
	require.NotEmpty(t, *resp.JSON409.Error)
}

func RequireUnauthorized(t *testing.T, resp *openapi.PostAuthLoginResponse, label string) {
	t.Helper()
	RequireStatus(t, http.StatusUnauthorized, resp.StatusCode(), resp.Body, label)
	require.NotNil(t, resp.JSON401)
	require.NotEmpty(t, *resp.JSON401.Error)
}

func RequireMeUnauthorized(t *testing.T, resp *openapi.GetAuthMeResponse) {
	t.Helper()
	RequireStatus(t, http.StatusUnauthorized, resp.StatusCode(), resp.Body, "me")
	require.NotNil(t, resp.JSON401)
	require.NotEmpty(t, *resp.JSON401.Error)
}

func RequireMyTeamOK(t *testing.T, resp *openapi.GetTeamsMyResponse) string {
	t.Helper()
	RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "get my team")
	require.NotNil(t, resp.JSON200)
	require.NotNil(t, resp.JSON200.ID)
	return *resp.JSON200.ID
}

func RequireAwardsCount(t *testing.T, resp *openapi.GetAdminAwardsTeamTeamIDResponse, count int) {
	t.Helper()
	RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "get awards by team")
	require.NotNil(t, resp.JSON200)
	require.Len(t, *resp.JSON200, count)
}

func RequireChallengeFields(t *testing.T, c *openapi.ResponseChallengeResponse, title string, solved *bool, solveCount, points *int) {
	t.Helper()
	require.NotNil(t, c, "challenge is nil")
	if title != "" {
		require.NotNil(t, c.Title)
		require.Equal(t, title, *c.Title)
	}
	if solved != nil {
		require.NotNil(t, c.Solved)
		require.Equal(t, *solved, *c.Solved)
	}
	if solveCount != nil {
		require.NotNil(t, c.SolveCount)
		require.Equal(t, *solveCount, *c.SolveCount)
	}
	if points != nil {
		require.NotNil(t, c.Points)
		require.Equal(t, *points, *c.Points)
	}
}
