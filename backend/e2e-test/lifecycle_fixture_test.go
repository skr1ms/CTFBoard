package e2e_test

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	openapiTypes "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func (s *e2eSuite) configureCompetition(admin e2eActor, name string) {
	s.t.Helper()

	s.configureCompetitionMode(admin, name, nil)
}

func (s *e2eSuite) configureCompetitionMode(admin e2eActor, name string, mode *openapi.UpdateCompetitionRequestMode) {
	s.t.Helper()

	resp, err := s.client.PutAdminCompetitionWithResponse(context.Background(), openapi.PutAdminCompetitionJSONRequestBody{
		Name:            name,
		IsPaused:        new(false),
		IsPublic:        new(true),
		AllowTeamSwitch: new(true),
		Mode:            mode,
	}, e2eBearer(admin.Token))
	require.NoError(s.t, err)
	requireStatus(s.t, "update competition", http.StatusOK, resp.StatusCode(), resp.Body)
}

func (s *e2eSuite) createChallengeWithState(admin e2eActor, title, flag string, points int, state openapi.CreateChallengeRequestState) string {
	s.t.Helper()

	decay := 1
	resp, err := s.client.PostAdminChallengesWithResponse(context.Background(), openapi.PostAdminChallengesJSONRequestBody{
		Title:        title,
		Description:  "lifecycle challenge",
		Category:     "misc",
		Flag:         flag,
		Points:       points,
		State:        &state,
		InitialValue: &points,
		MinValue:     &points,
		Decay:        &decay,
	}, e2eBearer(admin.Token))
	require.NoError(s.t, err)
	requireStatus(s.t, "create lifecycle challenge", http.StatusCreated, resp.StatusCode(), resp.Body)
	require.NotNil(s.t, resp.JSON201)
	require.NotNil(s.t, resp.JSON201.ID)

	return *resp.JSON201.ID
}

func (s *e2eSuite) updateChallengeTags(admin e2eActor, challengeID, title string, points int, tagIDs []string) {
	s.t.Helper()

	state := openapi.UpdateChallengeRequestStateVisible
	decay := 1
	resp, err := s.client.PutAdminChallengesIDWithResponse(context.Background(), challengeID, openapi.PutAdminChallengesIDJSONRequestBody{
		Title:        title,
		Description:  "lifecycle challenge with tags",
		Category:     "misc",
		Points:       points,
		State:        &state,
		InitialValue: &points,
		MinValue:     &points,
		Decay:        &decay,
		TagIds:       &tagIDs,
	}, e2eBearer(admin.Token))
	require.NoError(s.t, err)
	requireStatus(s.t, "update challenge tags", http.StatusOK, resp.StatusCode(), resp.Body)
}

func (s *e2eSuite) setChallengeRequirements(admin e2eActor, challengeID string, requirementIDs ...string) {
	s.t.Helper()

	resp, err := s.client.PutAdminChallengesChallengeIDRequirementsWithResponse(context.Background(), challengeID, openapi.PutAdminChallengesChallengeIDRequirementsJSONRequestBody{
		RequirementIds: &requirementIDs,
	}, e2eBearer(admin.Token))
	require.NoError(s.t, err)
	requireStatus(s.t, "set challenge requirements", http.StatusNoContent, resp.StatusCode(), resp.Body)
}

func (s *e2eSuite) createTag(admin e2eActor, name, color string) string {
	s.t.Helper()

	resp, err := s.client.PostAdminTagsWithResponse(context.Background(), openapi.PostAdminTagsJSONRequestBody{
		Name:  name,
		Color: &color,
	}, e2eBearer(admin.Token))
	require.NoError(s.t, err)
	requireStatus(s.t, "create tag", http.StatusCreated, resp.StatusCode(), resp.Body)
	require.NotNil(s.t, resp.JSON201)
	require.NotNil(s.t, resp.JSON201.ID)

	return *resp.JSON201.ID
}

func (s *e2eSuite) createPage(admin e2eActor, slug, title string) {
	s.t.Helper()

	content := "Lifecycle rules page"
	isDraft := false
	order := 1
	resp, err := s.client.PostAdminPagesWithResponse(context.Background(), openapi.PostAdminPagesJSONRequestBody{
		Slug:       slug,
		Title:      title,
		Content:    &content,
		IsDraft:    &isDraft,
		OrderIndex: &order,
	}, e2eBearer(admin.Token))
	require.NoError(s.t, err)
	requireStatus(s.t, "create page", http.StatusCreated, resp.StatusCode(), resp.Body)
}

func (s *e2eSuite) createNotification(admin e2eActor, title string) {
	s.t.Helper()

	typ := openapi.CreateNotificationRequestTypeInfo
	pinned := true
	resp, err := s.client.PostAdminNotificationsWithResponse(context.Background(), openapi.PostAdminNotificationsJSONRequestBody{
		Title:    title,
		Content:  "Lifecycle notification",
		Type:     &typ,
		IsPinned: &pinned,
	}, e2eBearer(admin.Token))
	require.NoError(s.t, err)
	requireStatus(s.t, "create notification", http.StatusCreated, resp.StatusCode(), resp.Body)
}

func (s *e2eSuite) createBracket(admin e2eActor, name string) string {
	s.t.Helper()

	description := "Lifecycle bracket"
	isDefault := false
	resp, err := s.client.PostAdminBracketsWithResponse(context.Background(), openapi.PostAdminBracketsJSONRequestBody{
		Name:        name,
		Description: &description,
		IsDefault:   &isDefault,
	}, e2eBearer(admin.Token))
	require.NoError(s.t, err)
	requireStatus(s.t, "create bracket", http.StatusCreated, resp.StatusCode(), resp.Body)
	require.NotNil(s.t, resp.JSON201)
	require.NotNil(s.t, resp.JSON201.ID)

	return *resp.JSON201.ID
}

func (s *e2eSuite) setTeamBracket(admin e2eActor, teamID, bracketID string) {
	s.t.Helper()

	parsed := openapiTypes.UUID(uuid.MustParse(bracketID))
	resp, err := s.client.PatchAdminTeamsIDBracketWithResponse(context.Background(), teamID, openapi.PatchAdminTeamsIDBracketJSONRequestBody{
		BracketID: &parsed,
	}, e2eBearer(admin.Token))
	require.NoError(s.t, err)
	requireStatus(s.t, "set team bracket", http.StatusOK, resp.StatusCode(), resp.Body)
}

func (s *e2eSuite) setTeamHidden(admin e2eActor, teamID string, hidden bool) {
	s.t.Helper()

	resp, err := s.client.PatchAdminTeamsIDHiddenWithResponse(context.Background(), teamID, openapi.PatchAdminTeamsIDHiddenJSONRequestBody{
		Hidden: hidden,
	}, e2eBearer(admin.Token))
	require.NoError(s.t, err)
	requireStatus(s.t, "set team hidden", http.StatusOK, resp.StatusCode(), resp.Body)
}

func (s *e2eSuite) scoreboardPositionAndPoints(token, teamName string, params *openapi.GetScoreboardParams) (int, int, bool) {
	s.t.Helper()

	resp, err := s.client.GetScoreboardWithResponse(context.Background(), params, e2eBearer(token))
	require.NoError(s.t, err)
	requireStatus(s.t, "scoreboard lookup", http.StatusOK, resp.StatusCode(), resp.Body)
	require.NotNil(s.t, resp.JSON200)

	for i, entry := range *resp.JSON200 {
		if entry.TeamName != nil && *entry.TeamName == teamName {
			require.NotNil(s.t, entry.Points)

			return i, *entry.Points, true
		}
	}

	return -1, 0, false
}

func (s *e2eSuite) requireScoreboardTeam(token, teamName string, minPoints int, params *openapi.GetScoreboardParams) (int, int) {
	s.t.Helper()

	position, points, found := s.scoreboardPositionAndPoints(token, teamName, params)
	require.True(s.t, found, "team %q not found in scoreboard", teamName)
	require.GreaterOrEqual(s.t, points, minPoints)

	return position, points
}

func (s *e2eSuite) teamMemberID(token, username string) string {
	s.t.Helper()

	resp, err := s.client.GetTeamsMyWithResponse(context.Background(), e2eBearer(token))
	require.NoError(s.t, err)
	requireStatus(s.t, "get team members", http.StatusOK, resp.StatusCode(), resp.Body)
	require.NotNil(s.t, resp.JSON200)
	require.NotNil(s.t, resp.JSON200.Members)

	for _, member := range *resp.JSON200.Members {
		if member.Username != nil && *member.Username == username {
			require.NotNil(s.t, member.ID)

			return *member.ID
		}
	}

	s.t.Fatalf("member %q not found", username)

	return ""
}

func (s *e2eSuite) createComment(actor e2eActor, challengeID, content string) string {
	s.t.Helper()

	resp, err := s.client.PostChallengesChallengeIDCommentsWithResponse(context.Background(), challengeID, openapi.PostChallengesChallengeIDCommentsJSONRequestBody{
		Content: content,
	}, e2eBearer(actor.Token))
	require.NoError(s.t, err)
	requireStatus(s.t, "create challenge comment", http.StatusCreated, resp.StatusCode(), resp.Body)
	require.NotNil(s.t, resp.JSON201)
	require.NotNil(s.t, resp.JSON201.ID)

	return *resp.JSON201.ID
}

func (s *e2eSuite) rateChallenge(actor e2eActor, challengeID string, value int) {
	s.t.Helper()

	review := "lifecycle review"
	resp, err := s.client.PutChallengesChallengeIDRatingWithResponse(context.Background(), challengeID, openapi.PutChallengesChallengeIDRatingJSONRequestBody{
		Value:  value,
		Review: &review,
	}, e2eBearer(actor.Token))
	require.NoError(s.t, err)
	requireStatus(s.t, "rate challenge", http.StatusOK, resp.StatusCode(), resp.Body)
}

func (s *e2eSuite) upsertSolution(admin e2eActor, challengeID string) {
	s.t.Helper()

	resp, err := s.client.PostAdminChallengesChallengeIDSolutionWithResponse(context.Background(), challengeID, openapi.PostAdminChallengesChallengeIDSolutionJSONRequestBody{
		Content: "## Lifecycle writeup\n\nSolve the challenge.",
	}, e2eBearer(admin.Token))
	require.NoError(s.t, err)
	requireStatus(s.t, "upsert solution", http.StatusOK, resp.StatusCode(), resp.Body)
}
