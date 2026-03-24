package webapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
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

func (g *GitHubAPI) FetchUserProfile(ctx context.Context, accessToken string) (*OAuthUserProfile, error) {
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

	return &OAuthUserProfile{
		ID:       strconv.FormatInt(user.ID, 10),
		Email:    email,
		Username: user.Login,
	}, nil
}

func (g *GitHubAPI) fetchUser(ctx context.Context, accessToken string) (*githubUser, error) {
	mkReq := func() (*http.Request, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.userURL, nil)
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

func (g *GitHubAPI) fetchPrimaryEmail(ctx context.Context, accessToken string) (string, error) {
	mkReq := func() (*http.Request, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.emailsURL, nil)
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

	return "", fmt.Errorf("GitHubAPI - fetchPrimaryEmail: no verified email found")
}
