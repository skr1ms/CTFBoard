package response

import (
	"time"

	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func FromGeneralStats(s *domain.GeneralStats) openapi.GeneralStats {
	return openapi.GeneralStats{
		UserCount:      httputil.Ptr(s.UserCount),
		TeamCount:      httputil.Ptr(s.TeamCount),
		ChallengeCount: httputil.Ptr(s.ChallengeCount),
		SolveCount:     httputil.Ptr(s.SolveCount),
	}
}

func FromChallengeStatsList(stats []*domain.ChallengeStats) []openapi.ChallengeStats {
	res := make([]openapi.ChallengeStats, len(stats))
	for i, s := range stats {
		res[i] = openapi.ChallengeStats{
			ID:         httputil.Ptr(s.ID.String()),
			Title:      httputil.Ptr(s.Title),
			Points:     httputil.Ptr(s.Points),
			SolveCount: httputil.Ptr(s.SolveCount),
			Category:   httputil.Ptr(s.Category),
		}
	}
	return res
}

func FromChallengeDetailStats(s *domain.ChallengeDetailStats) openapi.ChallengeDetailStats {
	res := openapi.ChallengeDetailStats{
		ID:               httputil.Ptr(s.ID.String()),
		Title:            httputil.Ptr(s.Title),
		Category:         httputil.Ptr(s.Category),
		Points:           httputil.Ptr(s.Points),
		SolveCount:       httputil.Ptr(s.SolveCount),
		TotalTeams:       httputil.Ptr(s.TotalTeams),
		PercentageSolved: httputil.Ptr(float32(s.PercentageSolved)),
	}
	if s.FirstBlood != nil {
		res.FirstBlood = httputil.Ptr(openapi.ChallengeSolveEntry{
			TeamID:   httputil.Ptr(s.FirstBlood.TeamID.String()),
			TeamName: httputil.Ptr(s.FirstBlood.TeamName),
			SolvedAt: httputil.Ptr(s.FirstBlood.SolvedAt),
		})
	}
	if len(s.Solves) > 0 {
		solves := make([]openapi.ChallengeSolveEntry, len(s.Solves))
		for i, e := range s.Solves {
			solves[i] = openapi.ChallengeSolveEntry{
				TeamID:   httputil.Ptr(e.TeamID.String()),
				TeamName: httputil.Ptr(e.TeamName),
				SolvedAt: httputil.Ptr(e.SolvedAt),
			}
		}
		res.Solves = &solves
	}
	return res
}

func FromScoreboardHistoryList(stats []*domain.ScoreboardHistoryEntry) []openapi.ScoreboardHistoryEntry {
	res := make([]openapi.ScoreboardHistoryEntry, len(stats))
	for i, s := range stats {
		res[i] = openapi.ScoreboardHistoryEntry{
			TeamID:    httputil.Ptr(s.TeamID.String()),
			TeamName:  httputil.Ptr(s.TeamName),
			Points:    httputil.Ptr(s.Points),
			Timestamp: httputil.Ptr(s.Timestamp.Format(time.RFC3339)),
		}
	}
	return res
}

func FromChallengeSolvePercentages(data []*domain.ChallengeSolvePercentage) []openapi.ChallengeSolvePercentage {
	res := make([]openapi.ChallengeSolvePercentage, len(data))
	for i, d := range data {
		res[i] = openapi.ChallengeSolvePercentage{
			ID:         httputil.Ptr(d.ID.String()),
			Title:      httputil.Ptr(d.Title),
			Category:   httputil.Ptr(d.Category),
			SolveCount: httputil.Ptr(d.SolveCount),
			TotalTeams: httputil.Ptr(d.TotalTeams),
			Percentage: httputil.Ptr(float32(d.Percentage)),
		}
	}
	return res
}

func FromScoreDistribution(data []*domain.ScoreDistributionBucket) []openapi.ScoreDistributionBucket {
	res := make([]openapi.ScoreDistributionBucket, len(data))
	for i, d := range data {
		res[i] = openapi.ScoreDistributionBucket{
			Bucket: httputil.Ptr(d.Bucket),
			Count:  httputil.Ptr(d.Count),
		}
	}
	return res
}

func FromSubmissionTimeSeries(data *domain.SubmissionTimeSeriesStats) openapi.SubmissionTimeSeriesResponse {
	items := make([]openapi.SubmissionTimeSeries, len(data.Items))
	for i, item := range data.Items {
		items[i] = openapi.SubmissionTimeSeries{
			Date:      parseDate(item.Date),
			Correct:   httputil.Ptr(item.Correct),
			Incorrect: httputil.Ptr(item.Incorrect),
		}
	}
	return openapi.SubmissionTimeSeriesResponse{
		Items:          &items,
		TotalCorrect:   httputil.Ptr(data.TotalCorrect),
		TotalIncorrect: httputil.Ptr(data.TotalIncorrect),
	}
}

func FromRegistrationTimeSeries(data []*domain.RegistrationTimePoint) []openapi.RegistrationTimePoint {
	res := make([]openapi.RegistrationTimePoint, len(data))
	for i, d := range data {
		res[i] = openapi.RegistrationTimePoint{
			Date:  parseDate(d.Date),
			Count: httputil.Ptr(d.Count),
		}
	}
	return res
}

func FromScoreboardGraph(g *domain.ScoreboardGraph) openapi.ScoreboardGraph {
	teams := make([]openapi.TeamTimeline, len(g.Teams))
	for i, t := range g.Teams {
		timeline := make([]openapi.ScorePoint, len(t.Timeline))
		for j, p := range t.Timeline {
			timeline[j] = openapi.ScorePoint{
				Timestamp: httputil.Ptr(p.Timestamp.Format(time.RFC3339)),
				Score:     httputil.Ptr(p.Score),
			}
		}
		teams[i] = openapi.TeamTimeline{
			TeamID:   httputil.Ptr(t.TeamID.String()),
			TeamName: httputil.Ptr(t.TeamName),
			Timeline: &timeline,
		}
	}
	return openapi.ScoreboardGraph{
		Range: httputil.Ptr(openapi.TimeRange{
			Start: httputil.Ptr(g.Range.Start.Format(time.RFC3339)),
			End:   httputil.Ptr(g.Range.End.Format(time.RFC3339)),
		}),
		Teams: &teams,
	}
}

func FromSolveMatrixList(matrix []*domain.SolveMatrixRow) []openapi.SolveMatrixRow {
	res := make([]openapi.SolveMatrixRow, len(matrix))
	for i, row := range matrix {
		res[i] = openapi.SolveMatrixRow{
			TeamID:            httputil.Ptr(row.TeamID),
			TeamName:          httputil.Ptr(row.TeamName),
			ChallengeID:       httputil.Ptr(row.ChallengeID),
			ChallengeTitle:    httputil.Ptr(row.ChallengeTitle),
			ChallengeCategory: httputil.Ptr(row.ChallengeCategory),
			Solved:            httputil.Ptr(row.Solved),
			SolvedAt:          row.SolvedAt,
		}
	}
	return res
}
