package helper

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func (h *E2EHelper) CreateChallenge(token string, data map[string]any) string {
	h.t.Helper()
	id := h.CreateChallengeExpectStatus(token, data, http.StatusCreated)
	require.NotEmpty(h.t, id, "create challenge returned empty id")

	return id
}

func (h *E2EHelper) CreateChallengeExpectStatus(token string, data map[string]any, expectStatus int) string {
	h.t.Helper()

	req := openapi.PostAdminChallengesJSONRequestBody{
		Category:    getStr(data, "category", "misc"),
		Description: getStr(data, "description", ""),
		Flag:        getStr(data, "flag", ""),
		Points:      getInt(data, "points"),
		Title:       getStr(data, "title", ""),
	}
	if v, ok := data["state"].(string); ok {
		s := openapi.CreateChallengeRequestState(v)
		req.State = &s
	}

	if v := getStr(data, "connection_info", ""); v != "" {
		req.ConnectionInfo = &v
	}

	req.MaxAttempts = getIntPtr(data, "max_attempts")

	req.Position = getIntPtr(data, "position")
	if v, ok := data["is_regex"].(bool); ok {
		req.IsRegex = &v
	}

	if v, ok := data["is_case_insensitive"].(bool); ok {
		req.IsCaseInsensitive = &v
	}

	req.InitialValue = getIntPtr(data, "initial_value")
	req.MinValue = getIntPtr(data, "min_value")
	req.Decay = getIntPtr(data, "decay")
	resp, err := h.client.PostAdminChallengesWithResponse(context.Background(), req, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "create challenge")

	if resp.JSON201 != nil && resp.JSON201.ID != nil {
		return *resp.JSON201.ID
	}

	return ""
}

func (h *E2EHelper) CreateBasicChallenge(token, title, flag string, points int) string {
	h.t.Helper()

	return h.CreateChallenge(token, map[string]any{
		"title":         title,
		"description":   "Standard basic challenge",
		"flag":          flag,
		"points":        points,
		"category":      "misc",
		"state":         "visible",
		"initial_value": points,
		"min_value":     points,
		"decay":         1,
	})
}

func (h *E2EHelper) UpdateChallenge(token, challengeID string, data map[string]any) {
	h.t.Helper()
	h.UpdateChallengeExpectStatus(token, challengeID, data, http.StatusOK)
}

func (h *E2EHelper) UpdateChallengeExpectStatus(token, challengeID string, data map[string]any, expectStatus int) {
	h.t.Helper()

	req := openapi.PutAdminChallengesIDJSONRequestBody{
		Category:    getStr(data, "category", "misc"),
		Description: getStr(data, "description", ""),
		Points:      getInt(data, "points"),
		Title:       getStr(data, "title", ""),
	}
	if v, ok := data["flag"].(string); ok {
		req.Flag = &v
	}

	if v, ok := data["state"].(string); ok {
		s := openapi.UpdateChallengeRequestState(v)
		req.State = &s
	}

	if v := getStr(data, "connection_info", ""); v != "" {
		req.ConnectionInfo = &v
	}

	req.MaxAttempts = getIntPtr(data, "max_attempts")

	req.Position = getIntPtr(data, "position")
	if tagIDs := getStrSlice(data, "tag_ids"); len(tagIDs) > 0 {
		req.TagIds = &tagIDs
	}

	resp, err := h.client.PutAdminChallengesIDWithResponse(context.Background(), challengeID, req, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "update challenge")
}

func (h *E2EHelper) DeleteChallenge(token, challengeID string) {
	h.t.Helper()
	h.DeleteChallengeExpectStatus(token, challengeID, http.StatusNoContent)
}

func (h *E2EHelper) DeleteChallengeExpectStatus(token, challengeID string, expectStatus int) {
	h.t.Helper()
	resp, err := h.client.DeleteAdminChallengesIDWithResponse(context.Background(), challengeID, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "delete challenge")
}

func (h *E2EHelper) SubmitFlag(token, challengeID, flag string, expectStatus int) *openapi.PostChallengesChallengeIDSubmitResponse {
	h.t.Helper()
	resp, err := h.client.PostChallengesChallengeIDSubmitWithResponse(context.Background(), challengeID, openapi.PostChallengesChallengeIDSubmitJSONRequestBody{Flag: flag}, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "submit flag")

	return resp
}

func (h *E2EHelper) SubmitFlagExpectStatus(token, challengeID, flag string, allowedStatuses ...int) *openapi.PostChallengesChallengeIDSubmitResponse {
	h.t.Helper()
	resp, err := h.client.PostChallengesChallengeIDSubmitWithResponse(context.Background(), challengeID, openapi.PostChallengesChallengeIDSubmitJSONRequestBody{Flag: flag}, WithBearerToken(token))
	require.NoError(h.t, err)
	require.Contains(h.t, allowedStatuses, resp.StatusCode(), "submit flag: status %d not in %v body=%s", resp.StatusCode(), allowedStatuses, string(resp.Body))

	return resp
}

func (h *E2EHelper) GetChallengesExpectStatus(token string, expectStatus int) *openapi.GetChallengesResponse {
	h.t.Helper()
	resp, err := h.client.GetChallengesWithResponse(context.Background(), nil, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "get challenges")

	return resp
}

func (h *E2EHelper) FindChallengeInList(token, challengeID string) *openapi.ChallengeResponse {
	h.t.Helper()
	resp := h.GetChallengesExpectStatus(token, http.StatusOK)
	require.NotNil(h.t, resp.JSON200)

	idx := slices.IndexFunc(*resp.JSON200, func(c openapi.ChallengeResponse) bool {
		return c.ID != nil && *c.ID == challengeID
	})
	if idx < 0 {
		h.t.Fatalf("Challenge %s not found in list", challengeID)

		return nil
	}

	return &(*resp.JSON200)[idx]
}

func (h *E2EHelper) AssertChallengeMissing(token, challengeID string) {
	h.t.Helper()
	resp := h.GetChallengesExpectStatus(token, http.StatusOK)
	require.NotNil(h.t, resp.JSON200)

	idx := slices.IndexFunc(*resp.JSON200, func(c openapi.ChallengeResponse) bool {
		return c.ID != nil && *c.ID == challengeID
	})
	if idx >= 0 {
		h.t.Fatalf("Challenge %s should NOT be in list", challengeID)
	}
}

func (h *E2EHelper) GetChallengeDetailExpectStatus(token, challengeID string, expectStatus int) *openapi.GetChallengesChallengeIDResponse {
	h.t.Helper()
	resp, err := h.client.GetChallengesChallengeIDWithResponse(context.Background(), challengeID, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "get challenge detail")

	return resp
}

func (h *E2EHelper) SetChallengeRequirements(token, challengeID string, requirementIDs []string) {
	h.t.Helper()
	h.SetChallengeRequirementsExpectStatus(token, challengeID, requirementIDs, http.StatusNoContent)
}

func (h *E2EHelper) SetChallengeRequirementsExpectStatus(token, challengeID string, requirementIDs []string, expectStatus int) {
	h.t.Helper()

	req := openapi.PutAdminChallengesChallengeIDRequirementsJSONRequestBody{}

	if len(requirementIDs) > 0 {
		req.RequirementIds = &requirementIDs
	}

	resp, err := h.client.PutAdminChallengesChallengeIDRequirementsWithResponse(context.Background(), challengeID, req, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "set challenge requirements")
}

func (h *E2EHelper) FirstBloodAvailable(token, challengeID string) bool {
	resp, err := h.client.GetChallengesChallengeIDFirstBloodWithResponse(context.Background(), challengeID, nil, WithBearerToken(token))

	return err == nil && resp != nil && resp.StatusCode() == http.StatusOK
}

func (h *E2EHelper) GetFirstBlood(token, challengeID string, expectStatus int) *openapi.GetChallengesChallengeIDFirstBloodResponse {
	h.t.Helper()
	resp, err := h.client.GetChallengesChallengeIDFirstBloodWithResponse(context.Background(), challengeID, nil, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "first-blood")

	return resp
}

func (h *E2EHelper) GetFirstBloodWithRetry(token, challengeID string, maxTries int, sleep time.Duration) *openapi.GetChallengesChallengeIDFirstBloodResponse {
	h.t.Helper()

	var last *openapi.GetChallengesChallengeIDFirstBloodResponse

	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = sleep
	bo.MaxInterval = sleep * 4
	bo.MaxElapsedTime = 0
	op := func() error {
		resp, err := h.client.GetChallengesChallengeIDFirstBloodWithResponse(context.Background(), challengeID, nil, WithBearerToken(token))
		require.NoError(h.t, err)

		last = resp
		if resp.StatusCode() == http.StatusOK {
			return nil
		}

		if resp.StatusCode() != http.StatusNotFound {
			return backoff.Permanent(errors.New("unexpected status"))
		}

		return errors.New("not found")
	}

	maxRetries := max(maxTries-1, 0)

	err := backoff.Retry(op, backoff.WithMaxRetries(bo, uint64(maxRetries)))
	if err != nil {
		h.t.Logf("GetFirstBloodWithRetry: %v", err)
	}

	if last != nil && last.StatusCode() != http.StatusOK {
		RequireStatus(h.t, http.StatusOK, last.StatusCode(), last.Body, "first-blood")
	}

	return last
}

func (h *E2EHelper) AssertFirstBlood(token, challengeID, expectedUsername, expectedTeamName string) {
	h.t.Helper()
	resp := h.GetFirstBloodWithRetry(token, challengeID, 4, 400*time.Millisecond)
	require.NotNil(h.t, resp.JSON200)
	require.NotNil(h.t, resp.JSON200.Username, "username")
	require.Equal(h.t, expectedUsername, *resp.JSON200.Username)
	require.NotNil(h.t, resp.JSON200.TeamName, "team_name")
	require.Equal(h.t, expectedTeamName, *resp.JSON200.TeamName)
	require.NotNil(h.t, resp.JSON200.UserID)
	require.NotNil(h.t, resp.JSON200.TeamID)
	require.NotNil(h.t, resp.JSON200.SolvedAt)
}
