package v1

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wahrwelt-kit/go-logkit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

type oauthProvidersSettingsUC struct {
	settings *domain.Settings
	err      error
}

func (uc oauthProvidersSettingsUC) Get(context.Context) (*domain.Settings, error) {
	return uc.settings, uc.err
}

func (uc oauthProvidersSettingsUC) Update(context.Context, *domain.Settings, uuid.UUID, string) error {
	return nil
}

func TestGetAuthOauthProvidersUsesRuntimeSettings(t *testing.T) {
	t.Parallel()

	server := NewServer(&helper.ServerDeps{
		User: helper.UserDeps{
			OAuthGitHubEnabled: true,
			OAuthGoogleEnabled: true,
		},
		Admin: helper.AdminDeps{
			SettingsUC: oauthProvidersSettingsUC{
				settings: &domain.Settings{
					OAuthGithubEnabled: true,
					OAuthGoogleEnabled: false,
				},
			},
		},
		Infra: helper.InfraDeps{Logger: logkit.Noop()},
	})

	rr := httptest.NewRecorder()
	server.GetAuthOauthProviders(rr, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/auth/oauth/providers", http.NoBody))

	require.Equal(t, http.StatusOK, rr.Code)

	var got map[string]bool
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	assert.Equal(t, map[string]bool{"github": true, "google": false}, got)
}

func TestGetAuthOauthProvidersFallsBackToConfig(t *testing.T) {
	t.Parallel()

	server := NewServer(&helper.ServerDeps{
		User: helper.UserDeps{
			OAuthGitHubEnabled: false,
			OAuthGoogleEnabled: true,
		},
		Admin: helper.AdminDeps{
			SettingsUC: oauthProvidersSettingsUC{err: errors.New("settings unavailable")},
		},
		Infra: helper.InfraDeps{Logger: logkit.Noop()},
	})

	rr := httptest.NewRecorder()
	server.GetAuthOauthProviders(rr, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/auth/oauth/providers", http.NoBody))

	require.Equal(t, http.StatusOK, rr.Code)

	var got map[string]bool
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	assert.Equal(t, map[string]bool{"github": false, "google": true}, got)
}
