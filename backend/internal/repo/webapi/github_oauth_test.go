package webapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newGitHubAPIWithURL(client *http.Client, userURL, emailsURL string) *GitHubAPI {
	return &GitHubAPI{client: client, userURL: userURL, emailsURL: emailsURL}
}

func TestGitHubAPI_FetchUserProfile_WithPublicEmail(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer token123", r.Header.Get("Authorization"))
		json.NewEncoder(w).Encode(githubUser{ID: 42, Login: "octocat", Email: "octo@example.com"})
	}))
	defer srv.Close()

	api := newGitHubAPIWithURL(srv.Client(), srv.URL+"/user", srv.URL+"/user/emails")
	profile, err := api.FetchUserProfile(t.Context(), "token123")
	require.NoError(t, err)
	assert.Equal(t, "42", profile.ID)
	assert.Equal(t, "octo@example.com", profile.Email)
	assert.Equal(t, "octocat", profile.Username)
}

func TestGitHubAPI_FetchUserProfile_FetchesPrimaryEmailWhenMissing(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/user", func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(githubUser{ID: 7, Login: "ghost", Email: ""})
	})
	mux.HandleFunc("/user/emails", func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode([]githubEmail{
			{Email: "secondary@example.com", Primary: false, Verified: true},
			{Email: "primary@example.com", Primary: true, Verified: true},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	api := newGitHubAPIWithURL(srv.Client(), srv.URL+"/user", srv.URL+"/user/emails")
	profile, err := api.FetchUserProfile(t.Context(), "tok")
	require.NoError(t, err)
	assert.Equal(t, "primary@example.com", profile.Email)
}

func TestGitHubAPI_FetchUserProfile_NoVerifiedEmail_ReturnsError(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/user", func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(githubUser{ID: 1, Login: "ghost", Email: ""})
	})
	mux.HandleFunc("/user/emails", func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode([]githubEmail{
			{Email: "unverified@example.com", Primary: true, Verified: false},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	api := newGitHubAPIWithURL(srv.Client(), srv.URL+"/user", srv.URL+"/user/emails")
	_, err := api.FetchUserProfile(t.Context(), "tok")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no verified email")
}

func TestGitHubAPI_FetchUserProfile_NonOKStatus_ReturnsError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Unauthorized"))
	}))
	defer srv.Close()

	api := newGitHubAPIWithURL(srv.Client(), srv.URL+"/user", srv.URL+"/user/emails")
	_, err := api.FetchUserProfile(t.Context(), "bad-token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "401")
}

func TestGitHubAPI_FetchUserProfile_MalformedJSON_ReturnsError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	api := newGitHubAPIWithURL(srv.Client(), srv.URL+"/user", srv.URL+"/user/emails")
	_, err := api.FetchUserProfile(t.Context(), "tok")
	require.Error(t, err)
}

func TestGitHubAPI_FetchUserProfile_EmailsEndpointFails_ReturnsError(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/user", func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(githubUser{ID: 1, Login: "ghost", Email: ""})
	})
	mux.HandleFunc("/user/emails", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	api := newGitHubAPIWithURL(srv.Client(), srv.URL+"/user", srv.URL+"/user/emails")
	_, err := api.FetchUserProfile(t.Context(), "tok")
	require.Error(t, err)
}

func TestGitHubAPI_FetchUserProfile_FallsBackToNonPrimaryVerified(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/user", func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(githubUser{ID: 1, Login: "ghost", Email: ""})
	})
	mux.HandleFunc("/user/emails", func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode([]githubEmail{
			{Email: "nonprimary@example.com", Primary: false, Verified: true},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	api := newGitHubAPIWithURL(srv.Client(), srv.URL+"/user", srv.URL+"/user/emails")
	profile, err := api.FetchUserProfile(t.Context(), "tok")
	require.NoError(t, err)
	assert.Equal(t, "nonprimary@example.com", profile.Email)
}
