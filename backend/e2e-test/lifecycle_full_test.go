package e2e_test

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	openapiTypes "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func TestE2E_FullCompetitionLifecycle_WideProductFlow(t *testing.T) {
	s := newE2ESuite(t)

	defer resetCompetitionToActive()

	suffix := e2eUID("full")
	admin := s.registerAdmin("full_admin")
	s.configureCompetition(admin, "Lifecycle "+suffix)

	tagID := s.createTag(admin, "tag_"+suffix, "#3b82f6")
	pageSlug := "rules-" + strings.ReplaceAll(suffix, "_", "-")
	s.createPage(admin, pageSlug, "Rules "+suffix)
	s.createNotification(admin, "Notice "+suffix)
	bracketID := s.createBracket(admin, "Bracket "+suffix)

	valueType := openapi.SetConfigRequestValueTypeString
	configResp, err := s.client.PutAdminConfigsKeyWithResponse(context.Background(), "lifecycle_"+suffix, openapi.PutAdminConfigsKeyJSONRequestBody{
		Value:       "enabled",
		ValueType:   &valueType,
		Description: new("e2e lifecycle config"),
	}, e2eBearer(admin.Token))
	require.NoError(t, err)
	requireStatus(t, "create lifecycle config", http.StatusOK, configResp.StatusCode(), configResp.Body)

	warmupID := s.createChallengeWithState(admin, "Warmup "+suffix, "flag{"+suffix+"_warmup}", 100, openapi.CreateChallengeRequestStateVisible)
	mediumID := s.createChallengeWithState(admin, "Medium "+suffix, "flag{"+suffix+"_medium}", 300, openapi.CreateChallengeRequestStateVisible)
	hardID := s.createChallengeWithState(admin, "Hard "+suffix, "flag{"+suffix+"_hard}", 600, openapi.CreateChallengeRequestStateVisible)
	s.updateChallengeTags(admin, warmupID, "Warmup "+suffix, 100, []string{tagID})
	s.setChallengeRequirements(admin, mediumID, warmupID)
	hintID := s.createHint(admin, mediumID, "Read the uploaded lifecycle file", 50)
	fileID := s.uploadChallengeFile(admin, mediumID, "lifecycle.txt", "medium challenge attachment")

	captain := s.registerUser("full_captain")
	member := s.registerUser("full_member")
	rival := s.registerUser("full_rival")
	alphaName := "Alpha " + suffix
	betaName := "Beta " + suffix

	s.createTeam(&captain, alphaName)
	s.createTeam(&rival, betaName)

	join, err := s.client.PostTeamsJoinWithResponse(context.Background(), openapi.PostTeamsJoinJSONRequestBody{
		InviteToken: s.inviteToken(captain),
	}, e2eBearer(member.Token))
	require.NoError(t, err)
	requireStatus(t, "member joins lifecycle team", http.StatusOK, join.StatusCode(), join.Body)

	renamedAlpha := "Alpha Renamed " + suffix
	rename, err := s.client.PatchTeamsMeWithResponse(context.Background(), openapi.PatchTeamsMeJSONRequestBody{Name: renamedAlpha}, e2eBearer(captain.Token))
	require.NoError(t, err)
	requireStatus(t, "rename team", http.StatusOK, rename.StatusCode(), rename.Body)

	captain.TeamName = renamedAlpha
	member.TeamName = renamedAlpha

	memberID := s.teamMemberID(captain.Token, member.Username)
	transfer, err := s.client.PostTeamsTransferCaptainWithResponse(context.Background(), openapi.PostTeamsTransferCaptainJSONRequestBody{
		NewCaptainID: memberID,
	}, e2eBearer(captain.Token))
	require.NoError(t, err)
	requireStatus(t, "transfer captain", http.StatusOK, transfer.StatusCode(), transfer.Body)

	teamAfterTransfer, err := s.client.GetTeamsMyWithResponse(context.Background(), e2eBearer(member.Token))
	require.NoError(t, err)
	requireStatus(t, "team after transfer", http.StatusOK, teamAfterTransfer.StatusCode(), teamAfterTransfer.Body)
	require.NotNil(t, teamAfterTransfer.JSON200)
	require.NotNil(t, teamAfterTransfer.JSON200.CaptainID)
	require.Equal(t, memberID, *teamAfterTransfer.JSON200.CaptainID)

	s.setTeamBracket(admin, captain.TeamID, bracketID)

	page, err := s.client.GetPagesSlugWithResponse(context.Background(), pageSlug)
	require.NoError(t, err)
	requireStatus(t, "public page", http.StatusOK, page.StatusCode(), page.Body)

	notifications, err := s.client.GetNotificationsWithResponse(context.Background(), nil)
	require.NoError(t, err)
	requireStatus(t, "public notifications", http.StatusOK, notifications.StatusCode(), notifications.Body)

	tags, err := s.client.GetTagsWithResponse(context.Background())
	require.NoError(t, err)
	requireStatus(t, "public tags", http.StatusOK, tags.StatusCode(), tags.Body)
	require.NotEmpty(t, *tags.JSON200)

	beforeReq, err := s.client.GetChallengesChallengeIDWithResponse(context.Background(), mediumID, e2eBearer(captain.Token))
	require.NoError(t, err)
	requireStatus(t, "locked challenge before requirements", http.StatusNotFound, beforeReq.StatusCode(), beforeReq.Body)
	s.submitFlag(captain, mediumID, "flag{"+suffix+"_medium}", false, http.StatusForbidden)

	s.submitFlag(captain, warmupID, "flag{wrong}", false, http.StatusOK)
	s.submitFlag(captain, warmupID, "flag{"+suffix+"_warmup}", true, http.StatusOK)

	firstBlood, err := s.client.GetChallengesChallengeIDFirstBloodWithResponse(context.Background(), warmupID, &openapi.GetChallengesChallengeIDFirstBloodParams{Live: new(true)}, e2eBearer(admin.Token))
	require.NoError(t, err)
	requireStatus(t, "first blood", http.StatusOK, firstBlood.StatusCode(), firstBlood.Body)
	require.NotNil(t, firstBlood.JSON200)
	require.Equal(t, renamedAlpha, *firstBlood.JSON200.TeamName)

	afterReq, err := s.client.GetChallengesChallengeIDWithResponse(context.Background(), mediumID, e2eBearer(member.Token))
	require.NoError(t, err)
	requireStatus(t, "locked challenge after requirements", http.StatusOK, afterReq.StatusCode(), afterReq.Body)

	requirements, err := s.client.GetChallengesChallengeIDRequirementsWithResponse(context.Background(), mediumID, e2eBearer(member.Token))
	require.NoError(t, err)
	requireStatus(t, "challenge requirements", http.StatusOK, requirements.StatusCode(), requirements.Body)
	require.NotNil(t, requirements.JSON200)

	unlock, err := s.client.PostChallengesChallengeIDHintsHintIDUnlockWithResponse(context.Background(), mediumID, hintID, e2eBearer(member.Token))
	require.NoError(t, err)
	requireStatus(t, "unlock lifecycle hint", http.StatusOK, unlock.StatusCode(), unlock.Body)

	files, err := s.client.GetChallengesChallengeIDFilesWithResponse(context.Background(), mediumID, nil, e2eBearer(member.Token))
	require.NoError(t, err)
	requireStatus(t, "list lifecycle files", http.StatusOK, files.StatusCode(), files.Body)
	require.NotNil(t, files.JSON200)
	require.NotEmpty(t, *files.JSON200)
	require.Equal(t, "medium challenge attachment", s.downloadFile(member, fileID))

	s.submitFlag(member, mediumID, "flag{"+suffix+"_medium}", true, http.StatusOK)
	s.rateChallenge(member, mediumID, 5)
	ratings, err := s.client.GetChallengesChallengeIDRatingsWithResponse(context.Background(), mediumID, e2eBearer(member.Token))
	require.NoError(t, err)
	requireStatus(t, "list challenge ratings", http.StatusOK, ratings.StatusCode(), ratings.Body)
	require.NotNil(t, ratings.JSON200)
	require.NotEmpty(t, *ratings.JSON200)

	s.upsertSolution(admin, mediumID)
	solution, err := s.client.GetChallengesChallengeIDSolutionWithResponse(context.Background(), mediumID, e2eBearer(member.Token))
	require.NoError(t, err)
	requireStatus(t, "get challenge solution", http.StatusOK, solution.StatusCode(), solution.Body)
	require.NotNil(t, solution.JSON200)

	solutions, err := s.client.GetChallengesSolutionsWithResponse(context.Background(), e2eBearer(member.Token))
	require.NoError(t, err)
	requireStatus(t, "list challenge solutions", http.StatusOK, solutions.StatusCode(), solutions.Body)
	require.NotNil(t, solutions.JSON200)
	require.NotEmpty(t, *solutions.JSON200)

	s.submitFlag(rival, warmupID, "flag{"+suffix+"_warmup}", true, http.StatusOK)
	s.submitFlag(rival, hardID, "flag{"+suffix+"_hard}", true, http.StatusOK)

	alphaPos, alphaPoints := s.requireScoreboardTeam(admin.Token, renamedAlpha, 350, &openapi.GetScoreboardParams{Live: new(true)})
	betaPos, betaPoints := s.requireScoreboardTeam(admin.Token, betaName, 700, &openapi.GetScoreboardParams{Live: new(true)})
	require.Greater(t, betaPoints, alphaPoints)
	require.Less(t, betaPos, alphaPos)

	bracketUUID := openapiTypes.UUID(uuid.MustParse(bracketID))
	s.requireScoreboardTeam(admin.Token, renamedAlpha, 350, &openapi.GetScoreboardParams{
		Bracket: &bracketUUID,
		Live:    new(true),
	})

	submissions, err := s.client.GetAdminSubmissionsChallengeChallengeIDWithResponse(context.Background(), warmupID, &openapi.GetAdminSubmissionsChallengeChallengeIDParams{
		Page:    new(1),
		PerPage: new(20),
		Live:    new(true),
	}, e2eBearer(admin.Token))
	require.NoError(t, err)
	requireStatus(t, "admin challenge submissions", http.StatusOK, submissions.StatusCode(), submissions.Body)
	require.NotNil(t, submissions.JSON200)
	require.NotNil(t, submissions.JSON200.Data)
	require.NotEmpty(t, *submissions.JSON200.Data)

	stats, err := s.client.GetAdminSubmissionsChallengeChallengeIDStatsWithResponse(context.Background(), warmupID, &openapi.GetAdminSubmissionsChallengeChallengeIDStatsParams{Live: new(true)}, e2eBearer(admin.Token))
	require.NoError(t, err)
	requireStatus(t, "admin submission stats", http.StatusOK, stats.StatusCode(), stats.Body)
	require.NotNil(t, stats.JSON200)

	generalStats, err := s.client.GetStatisticsGeneralWithResponse(context.Background(), &openapi.GetStatisticsGeneralParams{Live: new(true)}, e2eBearer(admin.Token))
	require.NoError(t, err)
	requireStatus(t, "general stats", http.StatusOK, generalStats.StatusCode(), generalStats.Body)
	require.NotNil(t, generalStats.JSON200)
	require.NotNil(t, generalStats.JSON200.UserCount)
	require.GreaterOrEqual(t, *generalStats.JSON200.UserCount, 3)

	challengeStats, err := s.client.GetStatisticsChallengesWithResponse(context.Background(), &openapi.GetStatisticsChallengesParams{Live: new(true)}, e2eBearer(admin.Token))
	require.NoError(t, err)
	requireStatus(t, "challenge stats", http.StatusOK, challengeStats.StatusCode(), challengeStats.Body)
	require.NotNil(t, challengeStats.JSON200)

	scoreboardStats, err := s.client.GetStatisticsScoreboardWithResponse(context.Background(), &openapi.GetStatisticsScoreboardParams{
		Limit: new(5),
		Live:  new(true),
	}, e2eBearer(admin.Token))
	require.NoError(t, err)
	requireStatus(t, "scoreboard stats", http.StatusOK, scoreboardStats.StatusCode(), scoreboardStats.Body)
	require.NotNil(t, scoreboardStats.JSON200)

	graph, err := s.client.GetScoreboardGraphWithResponse(context.Background(), &openapi.GetScoreboardGraphParams{
		Top:  new(5),
		Live: new(true),
	}, e2eBearer(admin.Token))
	require.NoError(t, err)
	requireStatus(t, "scoreboard graph", http.StatusOK, graph.StatusCode(), graph.Body)
	require.NotNil(t, graph.JSON200)

	s.createAward(admin, captain.TeamID, 150)
	s.requireScoreboardTeam(admin.Token, renamedAlpha, 500, &openapi.GetScoreboardParams{Live: new(true)})
	awards, err := s.client.GetTeamsMeAwardsWithResponse(context.Background(), e2eBearer(member.Token))
	require.NoError(t, err)
	requireStatus(t, "team awards", http.StatusOK, awards.StatusCode(), awards.Body)
	require.NotNil(t, awards.JSON200)
	require.NotEmpty(t, *awards.JSON200)

	s.setTeamHidden(admin, captain.TeamID, true)
	require.Eventually(t, func() bool {
		_, _, found := s.scoreboardPositionAndPoints(admin.Token, renamedAlpha, &openapi.GetScoreboardParams{Live: new(true)})

		return !found
	}, 5*time.Second, 200*time.Millisecond)

	s.setTeamHidden(admin, captain.TeamID, false)
	s.requireScoreboardTeam(admin.Token, renamedAlpha, 500, &openapi.GetScoreboardParams{Live: new(true)})

	endedAt := time.Now().UTC().Add(-time.Second)
	setCompetitionTimes(endedAt.Add(-2*time.Hour), endedAt, nil)
	invalidateScoreboardCache(context.Background())

	commentID := s.createComment(member, mediumID, "lifecycle comment")
	comments, err := s.client.GetChallengesChallengeIDCommentsWithResponse(context.Background(), mediumID, e2eBearer(member.Token))
	require.NoError(t, err)
	requireStatus(t, "list challenge comments", http.StatusOK, comments.StatusCode(), comments.Body)
	require.NotNil(t, comments.JSON200)
	require.NotEmpty(t, *comments.JSON200)

	deleteComment, err := s.client.DeleteCommentsIDWithResponse(context.Background(), commentID, e2eBearer(member.Token))
	require.NoError(t, err)
	requireStatus(t, "delete own comment", http.StatusNoContent, deleteComment.StatusCode(), deleteComment.Body)
}
