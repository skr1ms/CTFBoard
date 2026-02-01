package response

import (
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/openapi"
)

func FromGeneralStats(s *entity.GeneralStats) openapi.EntityGeneralStats {
	return openapi.EntityGeneralStats{
		UserCount:      ptr(s.UserCount),
		TeamCount:      ptr(s.TeamCount),
		ChallengeCount: ptr(s.ChallengeCount),
		SolveCount:     ptr(s.SolveCount),
	}
}

func FromChallengeStatsList(stats []*entity.ChallengeStats) []openapi.EntityChallengeStats {
	res := make([]openapi.EntityChallengeStats, len(stats))
	for i, s := range stats {
		res[i] = openapi.EntityChallengeStats{
			ID:         ptr(s.ID.String()),
			Title:      ptr(s.Title),
			Points:     ptr(s.Points),
			SolveCount: ptr(s.SolveCount),
			Category:   ptr(s.Category),
		}
	}
	return res
}

func FromChallengeDetailStats(s *entity.ChallengeDetailStats) openapi.EntityChallengeDetailStats {
	res := openapi.EntityChallengeDetailStats{
		ID:               ptr(s.ID.String()),
		Title:            ptr(s.Title),
		Category:         ptr(s.Category),
		Points:           ptr(s.Points),
		SolveCount:       ptr(s.SolveCount),
		TotalTeams:       ptr(s.TotalTeams),
		PercentageSolved: ptr(float32(s.PercentageSolved)),
	}
	if s.FirstBlood != nil {
		res.FirstBlood = &openapi.EntityChallengeSolveEntry{
			TeamID:   ptr(s.FirstBlood.TeamID.String()),
			TeamName: ptr(s.FirstBlood.TeamName),
			SolvedAt: ptr(s.FirstBlood.SolvedAt),
		}
	}
	if len(s.Solves) > 0 {
		solves := make([]openapi.EntityChallengeSolveEntry, len(s.Solves))
		for i, e := range s.Solves {
			solves[i] = openapi.EntityChallengeSolveEntry{
				TeamID:   ptr(e.TeamID.String()),
				TeamName: ptr(e.TeamName),
				SolvedAt: ptr(e.SolvedAt),
			}
		}
		res.Solves = &solves
	}
	return res
}

func FromScoreboardHistoryList(stats []*entity.ScoreboardHistoryEntry) []openapi.EntityScoreboardHistoryEntry {
	res := make([]openapi.EntityScoreboardHistoryEntry, len(stats))
	for i, s := range stats {
		res[i] = openapi.EntityScoreboardHistoryEntry{
			TeamID:    ptr(s.TeamID.String()),
			TeamName:  ptr(s.TeamName),
			Points:    ptr(s.Points),
			Timestamp: ptr(s.Timestamp.String()),
		}
	}
	return res
}

func FromScoreboardGraph(g *entity.ScoreboardGraph) openapi.EntityScoreboardGraph {
	teams := make([]openapi.EntityTeamTimeline, len(g.Teams))
	for i, t := range g.Teams {
		timeline := make([]openapi.EntityScorePoint, len(t.Timeline))
		for j, p := range t.Timeline {
			timeline[j] = openapi.EntityScorePoint{
				Timestamp: ptr(p.Timestamp.Format("2006-01-02T15:04:05Z07:00")),
				Score:     ptr(p.Score),
			}
		}
		teams[i] = openapi.EntityTeamTimeline{
			TeamID:   ptr(t.TeamID.String()),
			TeamName: ptr(t.TeamName),
			Timeline: &timeline,
		}
	}
	return openapi.EntityScoreboardGraph{
		Range: &openapi.EntityTimeRange{
			Start: ptr(g.Range.Start.Format("2006-01-02T15:04:05Z07:00")),
			End:   ptr(g.Range.End.Format("2006-01-02T15:04:05Z07:00")),
		},
		Teams: &teams,
	}
}
