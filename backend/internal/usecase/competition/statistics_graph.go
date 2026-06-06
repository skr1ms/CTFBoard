package competition

import (
	"slices"
	"strings"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

// buildScoreboardGraph transforms a flat list of scoreboard history entries
// into a ScoreboardGraph. It groups entries by team ID and builds a per-team
// timeline of ScorePoints (timestamp + cumulative score). The shared time
// range (earliest and latest timestamp across all entries) is computed in the
// same pass. Teams are sorted alphabetically by name in the result so the
// output is deterministic.
func buildScoreboardGraph(history []*domain.ScoreboardHistoryEntry) *domain.ScoreboardGraph {
	if len(history) == 0 {
		return &domain.ScoreboardGraph{
			Range: domain.TimeRange{},
			Teams: []domain.TeamTimeline{},
		}
	}

	teamMap := make(map[string]*domain.TeamTimeline)

	var minTime, maxTime time.Time

	for i, h := range history {
		if i == 0 {
			minTime = h.Timestamp
			maxTime = h.Timestamp
		} else {
			if h.Timestamp.Before(minTime) {
				minTime = h.Timestamp
			}

			if h.Timestamp.After(maxTime) {
				maxTime = h.Timestamp
			}
		}

		teamIDStr := h.TeamID.String()

		tl, exists := teamMap[teamIDStr]
		if !exists {
			tl = &domain.TeamTimeline{
				TeamID:   h.TeamID,
				TeamName: h.TeamName,
				Timeline: []domain.ScorePoint{},
			}
			teamMap[teamIDStr] = tl
		}

		tl.Timeline = append(tl.Timeline, domain.ScorePoint{
			Timestamp: h.Timestamp,
			Score:     h.Points,
		})
	}

	teams := make([]domain.TeamTimeline, 0, len(teamMap))
	for _, tl := range teamMap {
		teams = append(teams, *tl)
	}

	slices.SortFunc(teams, func(a, b domain.TeamTimeline) int {
		return strings.Compare(a.TeamName, b.TeamName)
	})

	return &domain.ScoreboardGraph{
		Range: domain.TimeRange{
			Start: minTime,
			End:   maxTime,
		},
		Teams: teams,
	}
}
