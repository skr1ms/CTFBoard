package e2e_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

const e2ePassword = "ValidPass1"

type e2eActor struct {
	Username string
	Email    string
	Password string
	Token    string
	UserID   string
	TeamID   string
	TeamName string
}

type e2eSuite struct {
	t      *testing.T
	client *openapi.ClientWithResponses
}

func newE2ESuite(t *testing.T) *e2eSuite {
	t.Helper()

	jar, err := cookiejar.New(nil)
	require.NoError(t, err)

	client, err := openapi.NewClientWithResponses(
		GetTestBaseURL()+"/api/v1",
		openapi.WithHTTPClient(&http.Client{Jar: jar}),
	)
	require.NoError(t, err)

	resetAppSettingsFull()
	resetCompetitionToActive()
	invalidateScoreboardCache(context.Background())

	return &e2eSuite{t: t, client: client}
}

func e2eUID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}

func e2eBearer(token string) openapi.RequestEditorFn {
	return func(_ context.Context, req *http.Request) error {
		if token != "" && !strings.HasPrefix(token, "Bearer ") {
			token = "Bearer " + token
		}

		req.Header.Set("Authorization", token)

		return nil
	}
}

func e2eLowercaseBearer(token string) openapi.RequestEditorFn {
	return func(_ context.Context, req *http.Request) error {
		if token != "" && !strings.HasPrefix(token, "bearer ") {
			token = "bearer " + token
		}

		req.Header.Set("Authorization", token)

		return nil
	}
}

func e2eAPIToken(token string) openapi.RequestEditorFn {
	return func(_ context.Context, req *http.Request) error {
		if token != "" && !strings.HasPrefix(token, "Token ") {
			token = "Token " + token
		}

		req.Header.Set("Authorization", token)

		return nil
	}
}

func requireStatus(t *testing.T, label string, want, got int, body []byte) {
	t.Helper()
	require.Equal(t, want, got, "%s: %s", label, body)
}

func (s *e2eSuite) registerUser(prefix string) e2eActor {
	s.t.Helper()

	username := e2eUID(prefix)
	email := username + "@example.com"

	resp, err := s.client.PostAuthRegisterWithResponse(context.Background(), openapi.PostAuthRegisterJSONRequestBody{
		Username: username,
		Email:    email,
		Password: e2ePassword,
	})
	require.NoError(s.t, err)
	requireStatus(s.t, "register", http.StatusCreated, resp.StatusCode(), resp.Body)

	actor := s.login(email, e2ePassword)
	actor.Username = username

	return actor
}

func (s *e2eSuite) login(email, password string) e2eActor {
	s.t.Helper()

	resp, err := s.client.PostAuthLoginWithResponse(context.Background(), openapi.PostAuthLoginJSONRequestBody{
		Email:    email,
		Password: password,
	})
	require.NoError(s.t, err)
	requireStatus(s.t, "login", http.StatusOK, resp.StatusCode(), resp.Body)
	require.NotNil(s.t, resp.JSON200)
	require.NotNil(s.t, resp.JSON200.AccessToken)

	me, err := s.client.GetAuthMeWithResponse(context.Background(), e2eBearer(*resp.JSON200.AccessToken))
	require.NoError(s.t, err)
	requireStatus(s.t, "me", http.StatusOK, me.StatusCode(), me.Body)
	require.NotNil(s.t, me.JSON200)
	require.NotNil(s.t, me.JSON200.ID)
	require.NotNil(s.t, me.JSON200.Email)
	require.NotNil(s.t, me.JSON200.Username)

	return e2eActor{
		Username: *me.JSON200.Username,
		Email:    *me.JSON200.Email,
		Password: password,
		Token:    *resp.JSON200.AccessToken,
		UserID:   *me.JSON200.ID,
	}
}

func (s *e2eSuite) registerAdmin(prefix string) e2eActor {
	s.t.Helper()

	admin := s.registerUser(prefix)

	_, err := TestPool.Exec(context.Background(), "UPDATE users SET role = 'admin' WHERE id = $1", admin.UserID)
	require.NoError(s.t, err)

	if TestRedis != nil {
		_ = TestRedis.Del(context.Background(), "user:"+admin.UserID).Err()
	}

	fresh := s.login(admin.Email, admin.Password)
	fresh.Username = admin.Username

	return fresh
}

func (s *e2eSuite) createTeam(actor *e2eActor, name string) {
	s.t.Helper()

	resp, err := s.client.PostTeamsWithResponse(context.Background(), openapi.PostTeamsJSONRequestBody{Name: name}, e2eBearer(actor.Token))
	require.NoError(s.t, err)
	requireStatus(s.t, "create team", http.StatusCreated, resp.StatusCode(), resp.Body)

	myTeam, err := s.client.GetTeamsMyWithResponse(context.Background(), e2eBearer(actor.Token))
	require.NoError(s.t, err)
	requireStatus(s.t, "get my team", http.StatusOK, myTeam.StatusCode(), myTeam.Body)
	require.NotNil(s.t, myTeam.JSON200)
	require.NotNil(s.t, myTeam.JSON200.ID)
	require.NotNil(s.t, myTeam.JSON200.Name)

	actor.TeamID = *myTeam.JSON200.ID
	actor.TeamName = *myTeam.JSON200.Name
}

func (s *e2eSuite) createSoloTeam(actor *e2eActor) {
	s.t.Helper()

	_, err := TestPool.Exec(context.Background(), "UPDATE competition SET mode = 'solo_only', updated_at = now() WHERE id = 1")
	require.NoError(s.t, err)

	if TestRedis != nil {
		_ = TestRedis.Del(context.Background(), "competition").Err()
	}

	if testCompetitionUC != nil {
		testCompetitionUC.InvalidateLocalCache()
	}

	resp, err := s.client.PostTeamsSoloWithResponse(context.Background(), openapi.PostTeamsSoloJSONRequestBody{}, e2eBearer(actor.Token))
	require.NoError(s.t, err)
	requireStatus(s.t, "create solo team", http.StatusCreated, resp.StatusCode(), resp.Body)

	myTeam, err := s.client.GetTeamsMyWithResponse(context.Background(), e2eBearer(actor.Token))
	require.NoError(s.t, err)
	requireStatus(s.t, "get my solo team", http.StatusOK, myTeam.StatusCode(), myTeam.Body)
	require.NotNil(s.t, myTeam.JSON200)
	require.NotNil(s.t, myTeam.JSON200.ID)
	require.NotNil(s.t, myTeam.JSON200.Name)

	actor.TeamID = *myTeam.JSON200.ID
	actor.TeamName = *myTeam.JSON200.Name
}

func (s *e2eSuite) inviteToken(actor e2eActor) string {
	s.t.Helper()

	resp, err := s.client.GetTeamsMeInviteWithResponse(context.Background(), e2eBearer(actor.Token))
	require.NoError(s.t, err)
	requireStatus(s.t, "get invite", http.StatusOK, resp.StatusCode(), resp.Body)
	require.NotNil(s.t, resp.JSON200)
	require.NotNil(s.t, resp.JSON200.InviteToken)

	return *resp.JSON200.InviteToken
}

func (s *e2eSuite) createChallenge(admin e2eActor, title, flag string, points int) string {
	s.t.Helper()

	state := openapi.CreateChallengeRequestStateVisible
	decay := 1
	req := openapi.PostAdminChallengesJSONRequestBody{
		Title:        title,
		Description:  "journey challenge",
		Category:     "misc",
		Flag:         flag,
		Points:       points,
		State:        &state,
		InitialValue: &points,
		MinValue:     &points,
		Decay:        &decay,
	}

	resp, err := s.client.PostAdminChallengesWithResponse(context.Background(), req, e2eBearer(admin.Token))
	require.NoError(s.t, err)
	requireStatus(s.t, "create challenge", http.StatusCreated, resp.StatusCode(), resp.Body)
	require.NotNil(s.t, resp.JSON201)
	require.NotNil(s.t, resp.JSON201.ID)

	return *resp.JSON201.ID
}

func (s *e2eSuite) createHint(admin e2eActor, challengeID, content string, cost int) string {
	s.t.Helper()

	order := 1
	resp, err := s.client.PostAdminChallengesChallengeIDHintsWithResponse(context.Background(), challengeID, openapi.PostAdminChallengesChallengeIDHintsJSONRequestBody{
		Content:    content,
		Cost:       &cost,
		OrderIndex: &order,
	}, e2eBearer(admin.Token))
	require.NoError(s.t, err)
	requireStatus(s.t, "create hint", http.StatusCreated, resp.StatusCode(), resp.Body)
	require.NotNil(s.t, resp.JSON201)
	require.NotNil(s.t, resp.JSON201.ID)

	return *resp.JSON201.ID
}

func (s *e2eSuite) createAward(admin e2eActor, teamID string, value int) {
	s.t.Helper()

	resp, err := s.client.PostAdminAwardsWithResponse(context.Background(), openapi.PostAdminAwardsJSONRequestBody{
		TeamID:      teamID,
		Value:       value,
		Description: "e2e balance seed",
	}, e2eBearer(admin.Token))
	require.NoError(s.t, err)
	requireStatus(s.t, "create award", http.StatusCreated, resp.StatusCode(), resp.Body)
}

func (s *e2eSuite) submitFlag(actor e2eActor, challengeID, flag string, correct bool, status int) {
	s.t.Helper()

	resp, err := s.client.PostChallengesChallengeIDSubmitWithResponse(context.Background(), challengeID, openapi.PostChallengesChallengeIDSubmitJSONRequestBody{
		Flag: flag,
	}, e2eBearer(actor.Token))
	require.NoError(s.t, err)
	requireStatus(s.t, "submit flag", status, resp.StatusCode(), resp.Body)

	if status == http.StatusOK {
		require.NotNil(s.t, resp.JSON200)
		require.Equal(s.t, correct, resp.JSON200.Correct)
	}
}

func (s *e2eSuite) requireTeamScore(token, teamName string, expected int) {
	s.t.Helper()

	resp, err := s.client.GetScoreboardWithResponse(context.Background(), &openapi.GetScoreboardParams{}, e2eBearer(token))
	require.NoError(s.t, err)
	requireStatus(s.t, "scoreboard", http.StatusOK, resp.StatusCode(), resp.Body)
	require.NotNil(s.t, resp.JSON200)

	for _, entry := range *resp.JSON200 {
		if entry.TeamName != nil && *entry.TeamName == teamName {
			require.NotNil(s.t, entry.Points)
			require.Equal(s.t, expected, *entry.Points)

			return
		}
	}

	s.t.Fatalf("team %q not found in scoreboard", teamName)
}

func (s *e2eSuite) uploadChallengeFile(admin e2eActor, challengeID, name, content string) string {
	s.t.Helper()

	var buf bytes.Buffer

	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile("file", name)
	require.NoError(s.t, err)
	_, err = part.Write([]byte(content))
	require.NoError(s.t, err)
	require.NoError(s.t, writer.Close())

	resp, err := s.client.PostAdminChallengesChallengeIDFilesWithBodyWithResponse(
		context.Background(),
		challengeID,
		writer.FormDataContentType(),
		&buf,
		e2eBearer(admin.Token),
	)
	require.NoError(s.t, err)
	requireStatus(s.t, "upload challenge file", http.StatusCreated, resp.StatusCode(), resp.Body)
	require.NotNil(s.t, resp.JSON201)
	require.Equal(s.t, name, resp.JSON201.Filename)
	require.Equal(s.t, int64(len(content)), resp.JSON201.Size)

	sum := sha256.Sum256([]byte(content))
	require.Equal(s.t, hex.EncodeToString(sum[:]), resp.JSON201.Sha256)

	return resp.JSON201.ID
}

func (s *e2eSuite) downloadFile(actor e2eActor, fileID string) string {
	s.t.Helper()

	resp, err := s.client.GetFilesIDDownloadWithResponse(context.Background(), fileID, e2eBearer(actor.Token))
	require.NoError(s.t, err)
	requireStatus(s.t, "get download url", http.StatusOK, resp.StatusCode(), resp.Body)
	require.NotNil(s.t, resp.JSON200)

	downloadURL := resp.JSON200.URL
	if strings.HasPrefix(downloadURL, "/") {
		downloadURL = GetTestBaseURL() + downloadURL
	} else if parsed, err := url.Parse(downloadURL); err == nil {
		base, baseErr := url.Parse(GetTestBaseURL())
		if baseErr == nil {
			parsed.Scheme = base.Scheme
			parsed.Host = base.Host
			downloadURL = parsed.String()
		}
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, downloadURL, http.NoBody)
	require.NoError(s.t, err)
	require.NoError(s.t, e2eBearer(actor.Token)(context.Background(), req))

	raw, err := http.DefaultClient.Do(req)
	require.NoError(s.t, err)

	defer raw.Body.Close()

	body, err := io.ReadAll(raw.Body)
	require.NoError(s.t, err)
	requireStatus(s.t, "download file", http.StatusOK, raw.StatusCode, body)

	return string(body)
}

func waitWSMessage(t *testing.T, conn *websocket.Conn, typ string, timeout time.Duration) map[string]any {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for {
		_, data, err := conn.Read(ctx)
		require.NoError(t, err)

		var msg map[string]any
		if json.Unmarshal(data, &msg) != nil {
			continue
		}

		if got, _ := msg["type"].(string); got == typ {
			return msg
		}
	}
}

func (s *e2eSuite) setScoreVisibility(admin e2eActor, visibility string) {
	s.t.Helper()

	valueType := openapi.SetConfigRequestValueTypeString
	description := "e2e score visibility"
	resp, err := s.client.PutAdminConfigsKeyWithResponse(context.Background(), "score_visibility", openapi.PutAdminConfigsKeyJSONRequestBody{
		Value:       visibility,
		ValueType:   &valueType,
		Description: &description,
	}, e2eBearer(admin.Token))
	require.NoError(s.t, err)
	requireStatus(s.t, "set score_visibility", http.StatusOK, resp.StatusCode(), resp.Body)
}
