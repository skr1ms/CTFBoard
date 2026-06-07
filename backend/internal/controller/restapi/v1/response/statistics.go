package response

import (
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func FromGeneralStats(s *domain.GeneralStats) openapi.GeneralStats {
	return openapi.GeneralStats{
		UserCount:      new(s.UserCount),
		TeamCount:      new(s.TeamCount),
		ChallengeCount: new(s.ChallengeCount),
		SolveCount:     new(s.SolveCount),
	}
}

func FromChallengeStatsList(stats []*domain.ChallengeStats) []openapi.ChallengeStats {
	res := make([]openapi.ChallengeStats, len(stats))
	for i, s := range stats {
		res[i] = openapi.ChallengeStats{
			ID:         new(s.ID.String()),
			Title:      new(s.Title),
			Points:     new(s.Points),
			SolveCount: new(s.SolveCount),
			Category:   new(s.Category),
		}
	}

	return res
}

func FromChallengeDetailStats(s *domain.ChallengeDetailStats) openapi.ChallengeDetailStats {
	res := openapi.ChallengeDetailStats{
		ID:               new(s.ID.String()),
		Title:            new(s.Title),
		Category:         new(s.Category),
		Points:           new(s.Points),
		SolveCount:       new(s.SolveCount),
		TotalTeams:       new(s.TotalTeams),
		PercentageSolved: new(float32(s.PercentageSolved)),
	}
	if s.FirstBlood != nil {
		res.FirstBlood = new(openapi.ChallengeSolveEntry{
			TeamID:   new(s.FirstBlood.TeamID.String()),
			TeamName: new(s.FirstBlood.TeamName),
			SolvedAt: new(s.FirstBlood.SolvedAt),
		})
	}

	if len(s.Solves) > 0 {
		solves := make([]openapi.ChallengeSolveEntry, len(s.Solves))
		for i, e := range s.Solves {
			solves[i] = openapi.ChallengeSolveEntry{
				TeamID:   new(e.TeamID.String()),
				TeamName: new(e.TeamName),
				SolvedAt: new(e.SolvedAt),
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
			TeamID:    new(s.TeamID.String()),
			TeamName:  new(s.TeamName),
			Points:    new(s.Points),
			Timestamp: timePtr(&s.Timestamp),
		}
	}

	return res
}

func FromChallengeSolvePercentages(data []*domain.ChallengeSolvePercentage) []openapi.ChallengeSolvePercentage {
	res := make([]openapi.ChallengeSolvePercentage, len(data))
	for i, d := range data {
		res[i] = openapi.ChallengeSolvePercentage{
			ID:         new(d.ID.String()),
			Title:      new(d.Title),
			Category:   new(d.Category),
			SolveCount: new(d.SolveCount),
			TotalTeams: new(d.TotalTeams),
			Percentage: new(float32(d.Percentage)),
		}
	}

	return res
}

func FromScoreDistribution(data []*domain.ScoreDistributionBucket) []openapi.ScoreDistributionBucket {
	res := make([]openapi.ScoreDistributionBucket, len(data))
	for i, d := range data {
		res[i] = openapi.ScoreDistributionBucket{
			Bucket: new(d.Bucket),
			Count:  new(d.Count),
		}
	}

	return res
}

func FromSubmissionTimeSeries(data *domain.SubmissionTimeSeriesStats) openapi.SubmissionTimeSeriesResponse {
	items := make([]openapi.SubmissionTimeSeries, len(data.Items))
	for i, item := range data.Items {
		items[i] = openapi.SubmissionTimeSeries{
			Date:      parseDate(item.Date),
			Correct:   new(item.Correct),
			Incorrect: new(item.Incorrect),
		}
	}

	return openapi.SubmissionTimeSeriesResponse{
		Items:          &items,
		TotalCorrect:   new(data.TotalCorrect),
		TotalIncorrect: new(data.TotalIncorrect),
	}
}

func FromRegistrationTimeSeries(data []*domain.RegistrationTimePoint) []openapi.RegistrationTimePoint {
	res := make([]openapi.RegistrationTimePoint, len(data))
	for i, d := range data {
		res[i] = openapi.RegistrationTimePoint{
			Date:  parseDate(d.Date),
			Count: new(d.Count),
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
				Timestamp: timePtr(&p.Timestamp),
				Score:     new(p.Score),
			}
		}

		teams[i] = openapi.TeamTimeline{
			TeamID:   new(t.TeamID.String()),
			TeamName: new(t.TeamName),
			Timeline: &timeline,
		}
	}

	return openapi.ScoreboardGraph{
		Range: new(openapi.TimeRange{
			Start: timePtr(&g.Range.Start),
			End:   timePtr(&g.Range.End),
		}),
		Teams: &teams,
	}
}

func FromSolveMatrixList(matrix []*domain.SolveMatrixRow) []openapi.SolveMatrixRow {
	res := make([]openapi.SolveMatrixRow, len(matrix))
	for i, row := range matrix {
		res[i] = openapi.SolveMatrixRow{
			TeamID:            new(row.TeamID),
			TeamName:          new(row.TeamName),
			ChallengeID:       new(row.ChallengeID),
			ChallengeTitle:    new(row.ChallengeTitle),
			ChallengeCategory: new(row.ChallengeCategory),
			Solved:            new(row.Solved),
			SolvedAt:          row.SolvedAt,
		}
	}

	return res
}

func FromAdminStatisticsFunnel(funnel *domain.AdminStatisticsFunnel) openapi.AdminStatisticsFunnel {
	if funnel == nil {
		return openapi.AdminStatisticsFunnel{}
	}

	challenges := make([]openapi.FunnelChallengeRow, len(funnel.Challenges))
	for i, row := range funnel.Challenges {
		challenges[i] = openapi.FunnelChallengeRow{
			ChallengeID:       new(row.ChallengeID),
			ChallengeTitle:    new(row.ChallengeTitle),
			ChallengeCategory: new(row.ChallengeCategory),
			OpenedCount:       new(row.OpenedCount),
			AttemptedCount:    new(row.AttemptedCount),
			SolvedCount:       new(row.SolvedCount),
		}
	}

	teams := make([]openapi.FunnelTeamRow, len(funnel.Teams))
	for i, row := range funnel.Teams {
		teams[i] = openapi.FunnelTeamRow{
			TeamID:         new(row.TeamID),
			TeamName:       new(row.TeamName),
			OpenedCount:    new(row.OpenedCount),
			AttemptedCount: new(row.AttemptedCount),
			SolvedCount:    new(row.SolvedCount),
		}
	}

	teamCells := make([]openapi.FunnelTeamCell, len(funnel.TeamCells))
	for i, cell := range funnel.TeamCells {
		teamCells[i] = openapi.FunnelTeamCell{
			TeamID:           new(cell.TeamID),
			ChallengeID:      new(cell.ChallengeID),
			Opened:           new(cell.Opened),
			Attempted:        new(cell.Attempted),
			Solved:           new(cell.Solved),
			FirstOpenedAt:    cell.FirstOpenedAt,
			FirstAttemptedAt: cell.FirstAttemptedAt,
			SolvedAt:         cell.SolvedAt,
		}
	}

	users := make([]openapi.FunnelUserRow, len(funnel.Users))
	for i, row := range funnel.Users {
		users[i] = openapi.FunnelUserRow{
			UserID:         new(row.UserID),
			Username:       new(row.Username),
			OpenedCount:    new(row.OpenedCount),
			AttemptedCount: new(row.AttemptedCount),
			SolvedCount:    new(row.SolvedCount),
		}
	}

	userCells := make([]openapi.FunnelUserCell, len(funnel.UserCells))
	for i, cell := range funnel.UserCells {
		userCells[i] = openapi.FunnelUserCell{
			UserID:           new(cell.UserID),
			ChallengeID:      new(cell.ChallengeID),
			Opened:           new(cell.Opened),
			Attempted:        new(cell.Attempted),
			Solved:           new(cell.Solved),
			FirstOpenedAt:    cell.FirstOpenedAt,
			FirstAttemptedAt: cell.FirstAttemptedAt,
			SolvedAt:         cell.SolvedAt,
		}
	}

	return openapi.AdminStatisticsFunnel{
		Challenges: &challenges,
		Teams:      &teams,
		TeamCells:  &teamCells,
		Users:      &users,
		UserCells:  &userCells,
	}
}
