package helper

import (
	"context"
	"net/http"

	"github.com/skr1ms/CTFBoard/internal/openapi"
	"github.com/stretchr/testify/require"
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
	if v, ok := data["is_hidden"].(bool); ok {
		req.IsHidden = &v
	}
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
	hidden := false
	return h.CreateChallenge(token, map[string]any{
		"title":         title,
		"description":   "Standard basic challenge",
		"flag":          flag,
		"points":        points,
		"category":      "misc",
		"is_hidden":     hidden,
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
	if v, ok := data["is_hidden"].(bool); ok {
		req.IsHidden = &v
	}
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

func (h *E2EHelper) SubmitFlag(token, challengeID, flag string, expectStatus int) *openapi.PostChallengesIDSubmitResponse {
	h.t.Helper()
	resp, err := h.client.PostChallengesIDSubmitWithResponse(context.Background(), challengeID, openapi.PostChallengesIDSubmitJSONRequestBody{Flag: flag}, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "submit flag")
	return resp
}

func (h *E2EHelper) GetChallengesExpectStatus(token string, expectStatus int) *openapi.GetChallengesResponse {
	h.t.Helper()
	resp, err := h.client.GetChallengesWithResponse(context.Background(), nil, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "get challenges")
	return resp
}

func (h *E2EHelper) FindChallengeInList(token, challengeID string) *openapi.ResponseChallengeResponse {
	h.t.Helper()
	resp := h.GetChallengesExpectStatus(token, http.StatusOK)
	require.NotNil(h.t, resp.JSON200)
	for i := range *resp.JSON200 {
		c := &(*resp.JSON200)[i]
		if c.ID != nil && *c.ID == challengeID {
			return c
		}
	}
	h.t.Fatalf("Challenge %s not found in list", challengeID)
	return nil
}

func (h *E2EHelper) AssertChallengeMissing(token, challengeID string) {
	h.t.Helper()
	resp := h.GetChallengesExpectStatus(token, http.StatusOK)
	require.NotNil(h.t, resp.JSON200)
	for i := range *resp.JSON200 {
		c := &(*resp.JSON200)[i]
		if c.ID != nil && *c.ID == challengeID {
			h.t.Fatalf("Challenge %s should NOT be in list", challengeID)
		}
	}
}

func (h *E2EHelper) GetFirstBlood(challengeID string, expectStatus int) *openapi.GetChallengesIDFirstBloodResponse {
	h.t.Helper()
	resp, err := h.client.GetChallengesIDFirstBloodWithResponse(context.Background(), challengeID)
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "first-blood")
	return resp
}

func (h *E2EHelper) AssertFirstBlood(challengeID, expectedUsername, expectedTeamName string) {
	h.t.Helper()
	resp := h.GetFirstBlood(challengeID, http.StatusOK)
	require.NotNil(h.t, resp.JSON200)
	require.NotNil(h.t, resp.JSON200.Username, "username")
	require.Equal(h.t, expectedUsername, *resp.JSON200.Username)
	require.NotNil(h.t, resp.JSON200.TeamName, "team_name")
	require.Equal(h.t, expectedTeamName, *resp.JSON200.TeamName)
	require.NotNil(h.t, resp.JSON200.UserID)
	require.NotNil(h.t, resp.JSON200.TeamID)
	require.NotNil(h.t, resp.JSON200.SolvedAt)
}
