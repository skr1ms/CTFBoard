package webapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

const (
	githubUserURL   = "https://api.github.com/user"
	githubEmailsURL = "https://api.github.com/user/emails"
)

type GitHubAPI struct {
	client    *http.Client
	userURL   string
	emailsURL string
}

var _ OAuthProviderAPI = (*GitHubAPI)(nil)

func NewGitHubAPI(client *http.Client) *GitHubAPI {
	if client == nil {
		client = defaultOAuthClient
	}

	return &GitHubAPI{client: client, userURL: githubUserURL, emailsURL: githubEmailsURL}
}

type githubUser struct {
	ID    int64  `json:"id"`
	Login string `json:"login"`
	Email string `json:"email"`
}

type githubEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

// FetchUserProfile retrieves the GitHub user's profile. It calls fetchUser to
// get the login and email; if the email field is empty (users may hide it),
// it falls back to fetchPrimaryEmail to find a primary verified address.
func (g *GitHubAPI) FetchUserProfile(ctx context.Context, accessToken string) (*domain.OAuthUserProfile, error) {
	user, err := g.fetchUser(ctx, accessToken)
	if err != nil {
		return nil, err
	}

	email := user.Email
	if email == "" {
		email, err = g.fetchPrimaryEmail(ctx, accessToken)
		if err != nil {
			return nil, fmt.Errorf("GitHubAPI - FetchUserProfile - fetchPrimaryEmail: %w", err)
		}
	}

	return &domain.OAuthUserProfile{
		ID:       strconv.FormatInt(user.ID, 10),
		Email:    email,
		Username: user.Login,
	}, nil
}

// fetchUser calls the GitHub /user endpoint with retry and decodes the response
// into a githubUser struct.
func (g *GitHubAPI) fetchUser(ctx context.Context, accessToken string) (*githubUser, error) {
	mkReq := func() (*http.Request, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.userURL, http.NoBody)
		if err != nil {
			return nil, err
		}

		req.Header.Set("Authorization", "Bearer "+accessToken)
		req.Header.Set("Accept", "application/vnd.github+json")

		return req, nil
	}

	resp, err := doWithRetry(ctx, g.client, mkReq)
	if err != nil {
		return nil, fmt.Errorf("GitHubAPI - fetchUser - Do: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("GitHubAPI - fetchUser: API returned %d (read body: %w)", resp.StatusCode, readErr)
		}

		return nil, fmt.Errorf("GitHubAPI - fetchUser: API returned %d: %s", resp.StatusCode, body)
	}

	var user githubUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("GitHubAPI - fetchUser - Decode: %w", err)
	}

	return &user, nil
}

// fetchPrimaryEmail calls the GitHub /user/emails endpoint and returns the
// first primary+verified address. Falls back to any verified address if no
// primary+verified one exists. Returns an error when no verified email is found.
func (g *GitHubAPI) fetchPrimaryEmail(ctx context.Context, accessToken string) (string, error) {
	mkReq := func() (*http.Request, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.emailsURL, http.NoBody)
		if err != nil {
			return nil, err
		}

		req.Header.Set("Authorization", "Bearer "+accessToken)
		req.Header.Set("Accept", "application/vnd.github+json")

		return req, nil
	}

	resp, err := doWithRetry(ctx, g.client, mkReq)
	if err != nil {
		return "", fmt.Errorf("GitHubAPI - fetchPrimaryEmail - Do: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return "", fmt.Errorf("GitHubAPI - fetchPrimaryEmail: API returned %d (read body: %w)", resp.StatusCode, readErr)
		}

		return "", fmt.Errorf("GitHubAPI - fetchPrimaryEmail: API returned %d: %s", resp.StatusCode, body)
	}

	var emails []githubEmail
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", fmt.Errorf("GitHubAPI - fetchPrimaryEmail - Decode: %w", err)
	}

	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email, nil
		}
	}

	for _, e := range emails {
		if e.Verified {
			return e.Email, nil
		}
	}

	return "", errors.New("GitHubAPI - fetchPrimaryEmail: no verified email found")
}
