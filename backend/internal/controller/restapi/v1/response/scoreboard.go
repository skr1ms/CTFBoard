package response

import (
	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func FromScoreboardEntry(e *domain.ScoreboardEntry) openapi.ScoreboardEntryResponse {
	res := openapi.ScoreboardEntryResponse{
		TeamID:   new(e.TeamID.String()),
		TeamName: new(e.TeamName),
		Points:   new(e.Points),
	}
	if !e.SolvedAt.IsZero() {
		res.LastSolved = timePtr(&e.SolvedAt)
	}

	return res
}

func FromScoreboardList(items []*domain.ScoreboardEntry) []openapi.ScoreboardEntryResponse {
	return lo.Map(items, func(item *domain.ScoreboardEntry, _ int) openapi.ScoreboardEntryResponse {
		return FromScoreboardEntry(item)
	})
}

func ScoreboardTeamIDs(items []*domain.ScoreboardEntry) []uuid.UUID {
	return lo.Map(items, func(item *domain.ScoreboardEntry, _ int) uuid.UUID {
		return item.TeamID
	})
}

// FromScoreboardListWithAvatars builds the scoreboard response, attaching
// pre-resolved thumbnail URLs from the provided map (teamID -> thumbURL).
// Pass a nil map when avatars are disabled.
func FromScoreboardListWithAvatars(items []*domain.ScoreboardEntry, thumbURLs map[uuid.UUID]string) []openapi.ScoreboardEntryResponse {
	result := make([]openapi.ScoreboardEntryResponse, len(items))
	for i, item := range items {
		res := FromScoreboardEntry(item)

		if url, ok := thumbURLs[item.TeamID]; ok && url != "" {
			res.TeamAvatarThumbnailURL = &url
		}

		result[i] = res
	}

	return result
}

func FromFirstBlood(fb *domain.FirstBloodEntry) openapi.FirstBloodResponse {
	return openapi.FirstBloodResponse{
		UserID:   new(fb.UserID.String()),
		Username: new(fb.Username),
		TeamID:   new(fb.TeamID.String()),
		TeamName: new(fb.TeamName),
		SolvedAt: timePtr(&fb.SolvedAt),
	}
}
