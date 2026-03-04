package response

import (
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func FromGeneralStats(s *entity.GeneralStats) openapi.GeneralStats {
	return openapi.GeneralStats{
		UserCount:      ptr(s.UserCount),
		TeamCount:      ptr(s.TeamCount),
		ChallengeCount: ptr(s.ChallengeCount),
		SolveCount:     ptr(s.SolveCount),
	}
}

func FromChallengeStatsList(stats []*entity.ChallengeStats) []openapi.ChallengeStats {
	res := make([]openapi.ChallengeStats, len(stats))
	for i, s := range stats {
		res[i] = openapi.ChallengeStats{
			ID:         ptr(s.ID.String()),
			Title:      ptr(s.Title),
			Points:     ptr(s.Points),
			SolveCount: ptr(s.SolveCount),
			Category:   ptr(s.Category),
		}
	}
	return res
}

func FromChallengeDetailStats(s *entity.ChallengeDetailStats) openapi.ChallengeDetailStats {
	res := openapi.ChallengeDetailStats{
		ID:               ptr(s.ID.String()),
		Title:            ptr(s.Title),
		Category:         ptr(s.Category),
		Points:           ptr(s.Points),
		SolveCount:       ptr(s.SolveCount),
		TotalTeams:       ptr(s.TotalTeams),
		PercentageSolved: ptr(float32(s.PercentageSolved)),
	}
	if s.FirstBlood != nil {
		res.FirstBlood = ptr(openapi.ChallengeSolveEntry{
			TeamID:   ptr(s.FirstBlood.TeamID.String()),
			TeamName: ptr(s.FirstBlood.TeamName),
			SolvedAt: ptr(s.FirstBlood.SolvedAt),
		})
	}
	if len(s.Solves) > 0 {
		solves := make([]openapi.ChallengeSolveEntry, len(s.Solves))
		for i, e := range s.Solves {
			solves[i] = openapi.ChallengeSolveEntry{
				TeamID:   ptr(e.TeamID.String()),
				TeamName: ptr(e.TeamName),
				SolvedAt: ptr(e.SolvedAt),
			}
		}
		res.Solves = &solves
	}
	return res
}

func FromScoreboardHistoryList(stats []*entity.ScoreboardHistoryEntry) []openapi.ScoreboardHistoryEntry {
	res := make([]openapi.ScoreboardHistoryEntry, len(stats))
	for i, s := range stats {
		res[i] = openapi.ScoreboardHistoryEntry{
			TeamID:    ptr(s.TeamID.String()),
			TeamName:  ptr(s.TeamName),
			Points:    ptr(s.Points),
			Timestamp: ptr(s.Timestamp.Format(time.RFC3339)),
		}
	}
	return res
}

func FromChallengeSolvePercentages(data []*entity.ChallengeSolvePercentage) []openapi.ChallengeSolvePercentage {
	res := make([]openapi.ChallengeSolvePercentage, len(data))
	for i, d := range data {
		res[i] = openapi.ChallengeSolvePercentage{
			ID:         ptr(d.ID.String()),
			Title:      ptr(d.Title),
			Category:   ptr(d.Category),
			SolveCount: ptr(d.SolveCount),
			TotalTeams: ptr(d.TotalTeams),
			Percentage: ptr(float32(d.Percentage)),
		}
	}
	return res
}

func FromScoreDistribution(data []*entity.ScoreDistributionBucket) []openapi.ScoreDistributionBucket {
	res := make([]openapi.ScoreDistributionBucket, len(data))
	for i, d := range data {
		res[i] = openapi.ScoreDistributionBucket{
			Bucket: ptr(d.Bucket),
			Count:  ptr(d.Count),
		}
	}
	return res
}

func FromSubmissionTimeSeries(data *entity.SubmissionTimeSeriesStats) openapi.SubmissionTimeSeriesResponse {
	items := make([]openapi.SubmissionTimeSeries, len(data.Items))
	for i, item := range data.Items {
		items[i] = openapi.SubmissionTimeSeries{
			Date:      parseDate(item.Date),
			Correct:   ptr(item.Correct),
			Incorrect: ptr(item.Incorrect),
		}
	}
	return openapi.SubmissionTimeSeriesResponse{
		Items:          &items,
		TotalCorrect:   ptr(data.TotalCorrect),
		TotalIncorrect: ptr(data.TotalIncorrect),
	}
}

func FromRegistrationTimeSeries(data []*entity.RegistrationTimePoint) []openapi.RegistrationTimePoint {
	res := make([]openapi.RegistrationTimePoint, len(data))
	for i, d := range data {
		res[i] = openapi.RegistrationTimePoint{
			Date:  parseDate(d.Date),
			Count: ptr(d.Count),
		}
	}
	return res
}

func FromScoreboardGraph(g *entity.ScoreboardGraph) openapi.ScoreboardGraph {
	teams := make([]openapi.TeamTimeline, len(g.Teams))
	for i, t := range g.Teams {
		timeline := make([]openapi.ScorePoint, len(t.Timeline))
		for j, p := range t.Timeline {
			timeline[j] = openapi.ScorePoint{
				Timestamp: ptr(p.Timestamp.Format(time.RFC3339)),
				Score:     ptr(p.Score),
			}
		}
		teams[i] = openapi.TeamTimeline{
			TeamID:   ptr(t.TeamID.String()),
			TeamName: ptr(t.TeamName),
			Timeline: &timeline,
		}
	}
	return openapi.ScoreboardGraph{
		Range: ptr(openapi.TimeRange{
			Start: ptr(g.Range.Start.Format(time.RFC3339)),
			End:   ptr(g.Range.End.Format(time.RFC3339)),
		}),
		Teams: &teams,
	}
}

func FromSolveMatrixList(matrix []*entity.SolveMatrixRow) []openapi.SolveMatrixRow {
	res := make([]openapi.SolveMatrixRow, len(matrix))
	for i, row := range matrix {
		res[i] = openapi.SolveMatrixRow{
			TeamID:            ptr(row.TeamID),
			TeamName:          ptr(row.TeamName),
			ChallengeID:       ptr(row.ChallengeID),
			ChallengeTitle:    ptr(row.ChallengeTitle),
			ChallengeCategory: ptr(row.ChallengeCategory),
			Solved:            ptr(row.Solved),
			SolvedAt:          row.SolvedAt,
		}
	}
	return res
}
