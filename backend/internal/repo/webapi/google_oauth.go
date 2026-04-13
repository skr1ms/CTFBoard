package webapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

const googleUserInfoURL = "https://www.googleapis.com/oauth2/v2/userinfo"

type GoogleAPI struct {
	client      *http.Client
	userInfoURL string
}

var _ OAuthProviderAPI = (*GoogleAPI)(nil)

func NewGoogleAPI(client *http.Client) *GoogleAPI {
	if client == nil {
		client = defaultOAuthClient
	}

	return &GoogleAPI{client: client, userInfoURL: googleUserInfoURL}
}

type googleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
}

// FetchUserProfile fetches Google userinfo from the OAuth2 userinfo endpoint
// with retry. Rejects accounts whose email is not marked verified by Google.
// When the display name is empty the email address is used as the username.
func (g *GoogleAPI) FetchUserProfile(ctx context.Context, accessToken string) (*domain.OAuthUserProfile, error) {
	mkReq := func() (*http.Request, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.userInfoURL, http.NoBody)
		if err != nil {
			return nil, err
		}

		req.Header.Set("Authorization", "Bearer "+accessToken)

		return req, nil
	}

	resp, err := doWithRetry(ctx, g.client, mkReq)
	if err != nil {
		return nil, fmt.Errorf("GoogleAPI - FetchUserProfile - Do: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("GoogleAPI - FetchUserProfile: API returned %d (read body: %w)", resp.StatusCode, readErr)
		}

		return nil, fmt.Errorf("GoogleAPI - FetchUserProfile: API returned %d: %s", resp.StatusCode, body)
	}

	var info googleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("GoogleAPI - FetchUserProfile - Decode: %w", err)
	}

	if !info.VerifiedEmail {
		return nil, fmt.Errorf("GoogleAPI - FetchUserProfile: email %q is not verified", info.Email)
	}

	username := info.Name
	if username == "" {
		username = info.Email
	}

	return &domain.OAuthUserProfile{
		ID:       info.ID,
		Email:    info.Email,
		Username: username,
	}, nil
}
