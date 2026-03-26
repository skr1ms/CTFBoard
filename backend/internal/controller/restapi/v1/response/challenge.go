package response

import (
	"context"
	"time"

	"github.com/samber/lo"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

func FromChallenge(c *domain.Challenge) openapi.ChallengeResponse {
	if c == nil {
		return openapi.ChallengeResponse{}
	}

	return openapi.ChallengeResponse{
		ID:             new(c.ID.String()),
		Title:          new(c.Title),
		Description:    new(c.Description),
		Category:       new(c.Category),
		ConnectionInfo: new(c.ConnectionInfo),
		MaxAttempts:    new(c.MaxAttempts),
		Position:       new(c.Position),
		Points:         new(c.Points),
		SolveCount:     new(c.SolveCount),
		State:          ptrChallengeResponseState(c.State),
	}
}

func FromChallengeWithSolved(cws *domain.ChallengeWithSolved) openapi.ChallengeResponse {
	if cws == nil {
		return openapi.ChallengeResponse{}
	}

	res := FromChallenge(cws.Challenge)
	res.Solved = new(cws.Solved)

	return res
}

func FromChallengeWithTags(cwt *usecase.ChallengeWithTags) openapi.ChallengeResponse {
	res := FromChallengeWithSolved(cwt.ChallengeWithSolved)
	if len(cwt.Tags) > 0 {
		tags := make([]openapi.TagResponse, len(cwt.Tags))
		for i, t := range cwt.Tags {
			tags[i] = FromTag(t)
		}

		res.Tags = &tags
	}

	return res
}

func FromChallengeList(items []*usecase.ChallengeWithTags) []openapi.ChallengeResponse {
	return lo.Map(items, func(item *usecase.ChallengeWithTags, _ int) openapi.ChallengeResponse {
		return FromChallengeWithTags(item)
	})
}

func FromTag(t *domain.Tag) openapi.TagResponse {
	return openapi.TagResponse{
		ID:    new(t.ID.String()),
		Name:  new(t.Name),
		Color: new(t.Color),
	}
}

func FromTagList(items []*domain.Tag) []openapi.TagResponse {
	return lo.Map(items, func(item *domain.Tag, _ int) openapi.TagResponse { return FromTag(item) })
}

func FromScoreboardEntry(e *domain.ScoreboardEntry) openapi.ScoreboardEntryResponse {
	res := openapi.ScoreboardEntryResponse{
		TeamID:   new(e.TeamID.String()),
		TeamName: new(e.TeamName),
		Points:   new(e.Points),
	}
	if !e.SolvedAt.IsZero() {
		res.LastSolved = new(e.SolvedAt.Format(time.RFC3339))
	}

	return res
}

func FromScoreboardList(items []*domain.ScoreboardEntry) []openapi.ScoreboardEntryResponse {
	return lo.Map(items, func(item *domain.ScoreboardEntry, _ int) openapi.ScoreboardEntryResponse {
		return FromScoreboardEntry(item)
	})
}

func FromScoreboardListWithAvatars(ctx context.Context, items []*domain.ScoreboardEntry, avatarUC usecase.AvatarUseCase) ([]openapi.ScoreboardEntryResponse, error) {
	result := make([]openapi.ScoreboardEntryResponse, len(items))
	for i, item := range items {
		res := FromScoreboardEntry(item)

		if avatarUC != nil {
			_, thumbURL, err := avatarUC.GetTeamAvatarURL(ctx, item.TeamID)
			if err != nil {
				return nil, err
			}

			if thumbURL != nil && *thumbURL != "" {
				res.TeamAvatarThumbnailURL = thumbURL
			}
		}

		result[i] = res
	}

	return result, nil
}

func FromFirstBlood(fb *domain.FirstBloodEntry) openapi.FirstBloodResponse {
	return openapi.FirstBloodResponse{
		UserID:   new(fb.UserID.String()),
		Username: new(fb.Username),
		TeamID:   new(fb.TeamID.String()),
		TeamName: new(fb.TeamName),
		SolvedAt: new(fb.SolvedAt.Format(time.RFC3339)),
	}
}

func FromChallengeDetail(d *usecase.ChallengeDetail) openapi.ChallengeDetailResponse {
	res := openapi.ChallengeDetailResponse{
		ID:             new(d.Challenge.ID.String()),
		Title:          new(d.Challenge.Title),
		Description:    new(d.Challenge.Description),
		Category:       new(d.Challenge.Category),
		ConnectionInfo: new(d.Challenge.ConnectionInfo),
		MaxAttempts:    new(d.Challenge.MaxAttempts),
		Position:       new(d.Challenge.Position),
		State:          ptrChallengeDetailResponseState(d.Challenge.State),
		Points:         new(d.Challenge.Points),
		SolveCount:     new(d.SolveCount),
		SolvedByMe:     new(d.SolvedByMe),
	}

	if len(d.Tags) > 0 {
		tags := make([]openapi.TagResponse, len(d.Tags))
		for i, t := range d.Tags {
			tags[i] = FromTag(t)
		}

		res.Tags = &tags
	}

	if len(d.Files) > 0 {
		files := make([]openapi.FileItem, len(d.Files))
		for i, f := range d.Files {
			files[i] = FromFile(f)
		}

		res.Files = &files
	}

	if len(d.Hints) > 0 {
		hints := make([]openapi.HintItem, len(d.Hints))
		for i, h := range d.Hints {
			hints[i] = FromHintWithUnlockStatus(h)
		}

		res.Hints = &hints
	}

	if d.FirstBlood != nil {
		res.FirstBlood = new(FromFirstBlood(d.FirstBlood))
	}

	return res
}

func FromChallengeSolves(solves []*domain.SolveWithDetails) []openapi.ChallengeSolveEntry {
	res := make([]openapi.ChallengeSolveEntry, len(solves))
	for i, s := range solves {
		res[i] = openapi.ChallengeSolveEntry{
			TeamID:   new(s.TeamID.String()),
			TeamName: new(s.TeamName),
			SolvedAt: new(s.SolvedAt),
		}
	}

	return res
}

func FromHintWithUnlockStatus(h *usecase.HintWithUnlockStatus) openapi.HintItem {
	res := openapi.HintItem{
		ID:         new(h.Hint.ID.String()),
		Title:      new(h.Hint.Title),
		Cost:       new(h.Hint.Cost),
		OrderIndex: new(h.Hint.OrderIndex),
		Unlocked:   new(h.Unlocked),
	}
	if h.Unlocked {
		res.Content = new(h.Hint.Content)
	}

	return res
}

func FromChallengeRequirements(items []*domain.ChallengeRequirement) []openapi.ChallengeRequirementResponse {
	res := make([]openapi.ChallengeRequirementResponse, len(items))
	for i, item := range items {
		res[i] = openapi.ChallengeRequirementResponse{
			ChallengeID:       new(item.ChallengeID.String()),
			ChallengeTitle:    new(item.ChallengeTitle),
			ChallengeCategory: item.Category,
		}
	}

	return res
}

func FromChallengeSolution(sol *domain.ChallengeSolution, downloadURLs map[string]string) openapi.ChallengeSolutionResponse {
	res := openapi.ChallengeSolutionResponse{
		ChallengeID: new(sol.ChallengeID.String()),
		Content:     new(sol.Content),
	}
	if len(sol.Files) > 0 {
		files := make([]openapi.FileItem, len(sol.Files))
		for i, f := range sol.Files {
			item := FromFile(f)
			if url, ok := downloadURLs[f.ID.String()]; ok {
				item.URL = new(url)
			}

			files[i] = item
		}

		res.Files = &files
	}

	return res
}

func FromChallengeSolutionEntryList(entries []*domain.ChallengeSolutionEntry, downloadURLs map[string]string) []openapi.ChallengeSolutionEntry {
	return lo.Map(entries, func(e *domain.ChallengeSolutionEntry, _ int) openapi.ChallengeSolutionEntry {
		return FromChallengeSolutionEntry(e, downloadURLs)
	})
}

func FromChallengeSolutionEntry(entry *domain.ChallengeSolutionEntry, downloadURLs map[string]string) openapi.ChallengeSolutionEntry {
	res := openapi.ChallengeSolutionEntry{
		ChallengeID:       new(entry.ChallengeID.String()),
		ChallengeTitle:    new(entry.ChallengeTitle),
		ChallengeCategory: new(entry.ChallengeCategory),
		Content:           new(entry.Content),
	}
	if len(entry.Files) > 0 {
		files := make([]openapi.FileItem, len(entry.Files))
		for i, f := range entry.Files {
			item := FromFile(f)
			if url, ok := downloadURLs[f.ID.String()]; ok {
				item.URL = new(url)
			}

			files[i] = item
		}

		res.Files = &files
	}

	return res
}

func FromSubmitFlag(correct bool, message string) openapi.SubmitFlagResponse {
	return openapi.SubmitFlagResponse{Correct: correct, Message: message}
}

func FromChallengeFlags(flags *domain.ChallengeFlags) openapi.ChallengeFlagsResponse {
	res := openapi.ChallengeFlagsResponse{
		Flags:             &[]string{flags.FlagHash},
		IsRegex:           new(flags.IsRegex),
		IsCaseInsensitive: new(flags.IsCaseInsensitive),
	}
	if flags.FlagRegex != nil && *flags.FlagRegex != "" {
		res.FlagRegex = flags.FlagRegex
	}

	if flags.FlagFormatRegex != nil {
		res.FlagFormatRegex = flags.FlagFormatRegex
	}

	return res
}

func FromChallenges(challenges []*domain.Challenge) []openapi.ChallengeResponse {
	return lo.Map(challenges, func(c *domain.Challenge, _ int) openapi.ChallengeResponse { return FromChallenge(c) })
}

func FromChallengeTypes(types []string) []string {
	return types
}

func EmptyChallengeSolutionEntryList() []openapi.ChallengeSolutionEntry {
	return []openapi.ChallengeSolutionEntry{}
}

func ptrChallengeResponseState(s string) *openapi.ChallengeResponseState {
	if s == "" {
		return nil
	}

	v := openapi.ChallengeResponseState(s)

	return &v
}

func ptrChallengeDetailResponseState(s string) *openapi.ChallengeDetailResponseState {
	if s == "" {
		return nil
	}

	v := openapi.ChallengeDetailResponseState(s)

	return &v
}
