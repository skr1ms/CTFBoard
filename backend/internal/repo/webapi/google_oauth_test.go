package webapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newGoogleAPIWithURL(client *http.Client, userInfoURL string) *GoogleAPI {
	return &GoogleAPI{client: client, userInfoURL: userInfoURL}
}

func TestGoogleAPI_FetchUserProfile_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer mytoken", r.Header.Get("Authorization"))

		err := json.NewEncoder(w).Encode(googleUserInfo{
			ID:            "g123",
			Email:         "alice@gmail.com",
			VerifiedEmail: true,
			Name:          "Alice",
		})
		if err != nil {
			return
		}
	}))
	defer srv.Close()

	api := newGoogleAPIWithURL(srv.Client(), srv.URL)
	profile, err := api.FetchUserProfile(t.Context(), "mytoken")
	require.NoError(t, err)
	assert.Equal(t, "g123", profile.ID)
	assert.Equal(t, "alice@gmail.com", profile.Email)
	assert.Equal(t, "Alice", profile.Username)
}

func TestGoogleAPI_FetchUserProfile_EmptyName_FallsBackToEmail(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		err := json.NewEncoder(w).Encode(googleUserInfo{
			ID:            "g456",
			Email:         "noname@gmail.com",
			VerifiedEmail: true,
			Name:          "",
		})
		if err != nil {
			return
		}
	}))
	defer srv.Close()

	api := newGoogleAPIWithURL(srv.Client(), srv.URL)
	profile, err := api.FetchUserProfile(t.Context(), "tok")
	require.NoError(t, err)
	assert.Equal(t, "noname@gmail.com", profile.Username)
}

func TestGoogleAPI_FetchUserProfile_UnverifiedEmail_ReturnsError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		err := json.NewEncoder(w).Encode(googleUserInfo{
			ID:            "g789",
			Email:         "unverified@gmail.com",
			VerifiedEmail: false,
			Name:          "Bob",
		})
		if err != nil {
			return
		}
	}))
	defer srv.Close()

	api := newGoogleAPIWithURL(srv.Client(), srv.URL)
	_, err := api.FetchUserProfile(t.Context(), "tok")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not verified")
}

func TestGoogleAPI_FetchUserProfile_NonOKStatus_ReturnsError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)

		if _, err := w.Write([]byte("Unauthorized")); err != nil {
			return
		}
	}))
	defer srv.Close()

	api := newGoogleAPIWithURL(srv.Client(), srv.URL)
	_, err := api.FetchUserProfile(t.Context(), "bad")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "401")
}

func TestGoogleAPI_FetchUserProfile_MalformedJSON_ReturnsError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if _, err := w.Write([]byte("not json at all")); err != nil {
			return
		}
	}))
	defer srv.Close()

	api := newGoogleAPIWithURL(srv.Client(), srv.URL)
	_, err := api.FetchUserProfile(t.Context(), "tok")
	require.Error(t, err)
}
