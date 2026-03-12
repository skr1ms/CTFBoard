package load_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

const (
	NumUsers        = 1000
	NumChallenges   = 50
	NumHintChalles  = 10
	seedConcurrency = 50
)

type UserToken struct {
	UserID string
	Token  string
	TeamID string
}

type HintEntry struct {
	ChallengeID string
	HintID      string
}

type TestFixture struct {
	AdminToken      string
	ChallengeIDs    []string
	Flags           []string
	HintEntries     []HintEntry
	Users           []UserToken
	BaseURL         string
	RaceUserToken   string
	RaceChallengeID string
	RaceHintID      string
	RaceHintChalID  string
}

func seedLoadTestData(ctx context.Context, baseURL string, pool *pgxpool.Pool) (*TestFixture, error) {
	client, err := openapi.NewClientWithResponses(baseURL + "/api/v1")
	if err != nil {
		return nil, fmt.Errorf("create openapi client: %w", err)
	}

	adminToken, err := seedAdmin(ctx, client, pool)
	if err != nil {
		return nil, fmt.Errorf("seed admin: %w", err)
	}
	if err := seedStartCompetition(ctx, client, adminToken); err != nil {
		return nil, fmt.Errorf("start competition: %w", err)
	}

	challengeIDs, flags, err := seedChallenges(ctx, client, adminToken, NumChallenges)
	if err != nil {
		return nil, fmt.Errorf("seed challenges: %w", err)
	}

	hintEntries, err := seedHints(ctx, client, adminToken, challengeIDs[:NumHintChalles])
	if err != nil {
		return nil, fmt.Errorf("seed hints: %w", err)
	}

	users, err := seedUsers(ctx, client, pool, NumUsers)
	if err != nil {
		return nil, fmt.Errorf("seed users: %w", err)
	}

	raceToken, raceChallengeID, raceHintID, raceHintChalID, err := seedRaceUser(ctx, client, pool, adminToken, challengeIDs, hintEntries)
	if err != nil {
		return nil, fmt.Errorf("seed race user: %w", err)
	}

	if err := rehashUsersWithMinCost(ctx, pool, "ValidPass1"); err != nil {
		return nil, fmt.Errorf("rehash user passwords: %w", err)
	}

	return &TestFixture{
		AdminToken:      adminToken,
		ChallengeIDs:    challengeIDs,
		Flags:           flags,
		HintEntries:     hintEntries,
		Users:           users,
		BaseURL:         baseURL,
		RaceUserToken:   raceToken,
		RaceChallengeID: raceChallengeID,
		RaceHintID:      raceHintID,
		RaceHintChalID:  raceHintChalID,
	}, nil
}

func seedAdmin(ctx context.Context, client *openapi.ClientWithResponses, pool *pgxpool.Pool) (string, error) {
	username := "lt_admin"
	email := "lt_admin@loadtest.local"
	password := "ValidPass1"

	regResp, err := client.PostAuthRegisterWithResponse(ctx, openapi.PostAuthRegisterJSONRequestBody{
		Username: &username,
		Email:    &email,
		Password: &password,
	})
	if err != nil {
		return "", fmt.Errorf("register: %w", err)
	}
	if regResp.StatusCode() != 201 {
		return "", fmt.Errorf("register returned %d: %s", regResp.StatusCode(), string(regResp.Body))
	}

	_, err = pool.Exec(ctx, "UPDATE users SET role = 'admin' WHERE email = $1", email)
	if err != nil {
		return "", fmt.Errorf("promote admin: %w", err)
	}

	loginResp, err := client.PostAuthLoginWithResponse(ctx, openapi.PostAuthLoginJSONRequestBody{
		Email:    &email,
		Password: password,
	})
	if err != nil {
		return "", fmt.Errorf("login: %w", err)
	}
	if loginResp.StatusCode() != 200 || loginResp.JSON200 == nil || loginResp.JSON200.AccessToken == nil {
		return "", fmt.Errorf("login returned %d: %s", loginResp.StatusCode(), string(loginResp.Body))
	}
	bearer := "Bearer " + *loginResp.JSON200.AccessToken
	authFn := bearerEditor(bearer)

	desc := "load-test admin"
	tokenResp, err := client.PostUserTokensWithResponse(ctx, openapi.PostUserTokensJSONRequestBody{
		Description: &desc,
	}, authFn)
	if err != nil {
		return "", fmt.Errorf("create api token: %w", err)
	}
	if tokenResp.StatusCode() != 201 || tokenResp.JSON201 == nil || tokenResp.JSON201.Token == "" {
		return "", fmt.Errorf("create api token returned %d: %s", tokenResp.StatusCode(), string(tokenResp.Body))
	}
	return "Token " + tokenResp.JSON201.Token, nil
}

func seedStartCompetition(ctx context.Context, client *openapi.ClientWithResponses, adminToken string) error {
	now := time.Now().UTC()
	start := now.Add(-1 * time.Hour)
	end := now.Add(48 * time.Hour)
	isPaused := false
	allowSwitch := true
	mode := "flexible"
	authFn := bearerEditor(adminToken)

	resp, err := client.PutAdminCompetitionWithResponse(ctx, openapi.PutAdminCompetitionJSONRequestBody{
		Name:            "Load Test CTF",
		StartTime:       &start,
		EndTime:         &end,
		IsPaused:        &isPaused,
		AllowTeamSwitch: &allowSwitch,
		Mode:            &mode,
	}, authFn)
	if err != nil {
		return fmt.Errorf("put competition: %w", err)
	}
	if resp.StatusCode() != 200 {
		return fmt.Errorf("put competition returned %d: %s", resp.StatusCode(), string(resp.Body))
	}
	return nil
}

func seedChallenges(ctx context.Context, client *openapi.ClientWithResponses, adminToken string, n int) ([]string, []string, error) {
	ids := make([]string, 0, n)
	flags := make([]string, 0, n)
	authFn := bearerEditor(adminToken)

	for i := range n {
		title := fmt.Sprintf("LoadTest Challenge %d", i+1)
		flag := fmt.Sprintf("FLAG{load_test_%d}", i+1)
		desc := "Load test challenge"
		category := "misc"
		points := 100
		initialValue := 100
		minValue := 100
		decay := 1
		isHidden := false

		resp, err := client.PostAdminChallengesWithResponse(ctx, openapi.PostAdminChallengesJSONRequestBody{
			Title:        title,
			Flag:         flag,
			Description:  desc,
			Category:     category,
			Points:       points,
			IsHidden:     &isHidden,
			InitialValue: &initialValue,
			MinValue:     &minValue,
			Decay:        &decay,
		}, authFn)
		if err != nil {
			return nil, nil, fmt.Errorf("create challenge %d: %w", i, err)
		}
		if resp.StatusCode() != 201 || resp.JSON201 == nil || resp.JSON201.ID == nil {
			return nil, nil, fmt.Errorf("create challenge %d returned %d: %s", i, resp.StatusCode(), string(resp.Body))
		}
		ids = append(ids, *resp.JSON201.ID)
		flags = append(flags, flag)
	}
	return ids, flags, nil
}

func seedHints(ctx context.Context, client *openapi.ClientWithResponses, adminToken string, challengeIDs []string) ([]HintEntry, error) {
	entries := make([]HintEntry, 0, len(challengeIDs))
	authFn := bearerEditor(adminToken)

	for i, chalID := range challengeIDs {
		content := fmt.Sprintf("Hint for challenge %d", i+1)
		cost := 10
		order := 1

		resp, err := client.PostAdminChallengesChallengeIDHintsWithResponse(ctx, chalID,
			openapi.PostAdminChallengesChallengeIDHintsJSONRequestBody{
				Content:    content,
				Cost:       &cost,
				OrderIndex: &order,
			}, authFn)
		if err != nil {
			return nil, fmt.Errorf("create hint for challenge %s: %w", chalID, err)
		}
		if resp.StatusCode() != 201 || resp.JSON201 == nil || resp.JSON201.ID == nil {
			return nil, fmt.Errorf("create hint returned %d: %s", resp.StatusCode(), string(resp.Body))
		}
		entries = append(entries, HintEntry{ChallengeID: chalID, HintID: *resp.JSON201.ID})
	}
	return entries, nil
}

type seedResult struct {
	ut  UserToken
	err error
}

func seedUsers(ctx context.Context, client *openapi.ClientWithResponses, _ *pgxpool.Pool, n int) ([]UserToken, error) {
	sem := make(chan struct{}, seedConcurrency)
	results := make([]seedResult, n)
	var wg sync.WaitGroup

	for i := range n {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int) {
			defer wg.Done()
			defer func() { <-sem }()
			results[idx] = seedSingleUser(ctx, client, idx)
		}(i)
	}
	wg.Wait()

	tokens := make([]UserToken, 0, n)
	for _, r := range results {
		if r.err != nil {
			return nil, r.err
		}
		tokens = append(tokens, r.ut)
	}
	return tokens, nil
}

func seedSingleUser(ctx context.Context, client *openapi.ClientWithResponses, idx int) seedResult {
	username := fmt.Sprintf("lt_user_%04d", idx)
	email := fmt.Sprintf("lt_user_%04d@loadtest.local", idx)
	password := "ValidPass1"

	regResp, err := client.PostAuthRegisterWithResponse(ctx, openapi.PostAuthRegisterJSONRequestBody{
		Username: &username,
		Email:    &email,
		Password: &password,
	})
	if err != nil {
		return seedResult{err: fmt.Errorf("register user %d: %w", idx, err)}
	}
	if regResp.StatusCode() != 201 {
		return seedResult{err: fmt.Errorf("register user %d returned %d: %s", idx, regResp.StatusCode(), string(regResp.Body))}
	}

	loginResp, err := client.PostAuthLoginWithResponse(ctx, openapi.PostAuthLoginJSONRequestBody{
		Email:    &email,
		Password: password,
	})
	if err != nil {
		return seedResult{err: fmt.Errorf("login user %d: %w", idx, err)}
	}
	if loginResp.StatusCode() != 200 || loginResp.JSON200 == nil || loginResp.JSON200.AccessToken == nil {
		return seedResult{err: fmt.Errorf("login user %d returned %d: %s", idx, loginResp.StatusCode(), string(loginResp.Body))}
	}
	token := "Bearer " + *loginResp.JSON200.AccessToken
	authFn := bearerEditor(token)

	teamResp, err := client.PostTeamsSoloWithResponse(ctx, openapi.PostTeamsSoloJSONRequestBody{}, authFn)
	if err != nil {
		return seedResult{err: fmt.Errorf("create team user %d: %w", idx, err)}
	}
	if teamResp.StatusCode() != 201 {
		return seedResult{err: fmt.Errorf("create team user %d returned %d: %s", idx, teamResp.StatusCode(), string(teamResp.Body))}
	}

	myTeamResp, err := client.GetTeamsMyWithResponse(ctx, authFn)
	if err != nil {
		return seedResult{err: fmt.Errorf("get team user %d: %w", idx, err)}
	}
	teamID := ""
	if myTeamResp.StatusCode() == 200 && myTeamResp.JSON200 != nil && myTeamResp.JSON200.ID != nil {
		teamID = *myTeamResp.JSON200.ID
	}

	meResp, err := client.GetAuthMeWithResponse(ctx, authFn)
	if err != nil {
		return seedResult{err: fmt.Errorf("get me user %d: %w", idx, err)}
	}
	userID := ""
	if meResp.StatusCode() == 200 && meResp.JSON200 != nil && meResp.JSON200.ID != nil {
		userID = *meResp.JSON200.ID
	}

	return seedResult{ut: UserToken{UserID: userID, Token: token, TeamID: teamID}}
}

func seedRaceUser(
	ctx context.Context,
	client *openapi.ClientWithResponses,
	pool *pgxpool.Pool,
	_ string,
	challengeIDs []string,
	hintEntries []HintEntry,
) (string, string, string, string, error) {
	username := "lt_race_user"
	email := "lt_race@loadtest.local"
	password := "ValidPass1"

	regResp, err := client.PostAuthRegisterWithResponse(ctx, openapi.PostAuthRegisterJSONRequestBody{
		Username: &username,
		Email:    &email,
		Password: &password,
	})
	if err != nil {
		return "", "", "", "", fmt.Errorf("register race user: %w", err)
	}
	if regResp.StatusCode() != 201 {
		return "", "", "", "", fmt.Errorf("register race user returned %d: %s", regResp.StatusCode(), string(regResp.Body))
	}
	_ = pool

	loginResp, err := client.PostAuthLoginWithResponse(ctx, openapi.PostAuthLoginJSONRequestBody{
		Email:    &email,
		Password: password,
	})
	if err != nil {
		return "", "", "", "", fmt.Errorf("login race user: %w", err)
	}
	if loginResp.StatusCode() != 200 || loginResp.JSON200 == nil || loginResp.JSON200.AccessToken == nil {
		return "", "", "", "", fmt.Errorf("login race user returned %d: %s", loginResp.StatusCode(), string(loginResp.Body))
	}
	token := "Bearer " + *loginResp.JSON200.AccessToken
	authFn := bearerEditor(token)

	teamResp, err := client.PostTeamsSoloWithResponse(ctx, openapi.PostTeamsSoloJSONRequestBody{}, authFn)
	if err != nil {
		return "", "", "", "", fmt.Errorf("create race team: %w", err)
	}
	if teamResp.StatusCode() != 201 {
		return "", "", "", "", fmt.Errorf("create race team returned %d: %s", teamResp.StatusCode(), string(teamResp.Body))
	}

	raceChallengeID := challengeIDs[len(challengeIDs)-1]

	raceHintID := ""
	raceHintChalID := ""
	if len(hintEntries) > 0 {
		raceHintID = hintEntries[0].HintID
		raceHintChalID = hintEntries[0].ChallengeID
	}

	return token, raceChallengeID, raceHintID, raceHintChalID, nil
}

func bearerEditor(token string) openapi.RequestEditorFn {
	return func(_ context.Context, req *http.Request) error {
		t := token
		if t != "" && !strings.Contains(t, " ") {
			t = "Bearer " + t
		}
		req.Header.Set("Authorization", t)
		return nil
	}
}

func rehashUsersWithMinCost(ctx context.Context, pool *pgxpool.Pool, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		return fmt.Errorf("generate min-cost hash: %w", err)
	}
	_, err = pool.Exec(ctx, "UPDATE users SET password_hash = $1", string(hash))
	return err
}
