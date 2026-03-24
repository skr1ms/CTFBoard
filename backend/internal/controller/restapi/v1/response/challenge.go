package response

import (
	"time"

	"github.com/samber/lo"
	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

func FromChallenge(c *domain.Challenge) openapi.ChallengeResponse {
	if c == nil {
		return openapi.ChallengeResponse{}
	}
	return openapi.ChallengeResponse{
		ID:             httputil.Ptr(c.ID.String()),
		Title:          httputil.Ptr(c.Title),
		Description:    httputil.Ptr(c.Description),
		Category:       httputil.Ptr(c.Category),
		ConnectionInfo: httputil.Ptr(c.ConnectionInfo),
		MaxAttempts:    httputil.Ptr(c.MaxAttempts),
		Position:       httputil.Ptr(c.Position),
		Points:         httputil.Ptr(c.Points),
		SolveCount:     httputil.Ptr(c.SolveCount),
		State:          ptrChallengeResponseState(c.State),
	}
}

func FromChallengeWithSolved(cws *domain.ChallengeWithSolved) openapi.ChallengeResponse {
	if cws == nil {
		return openapi.ChallengeResponse{}
	}
	res := FromChallenge(cws.Challenge)
	res.Solved = httputil.Ptr(cws.Solved)
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
		ID:    httputil.Ptr(t.ID.String()),
		Name:  httputil.Ptr(t.Name),
		Color: httputil.Ptr(t.Color),
	}
}

func FromTagList(items []*domain.Tag) []openapi.TagResponse {
	return lo.Map(items, func(item *domain.Tag, _ int) openapi.TagResponse { return FromTag(item) })
}

func FromScoreboardEntry(e *domain.ScoreboardEntry) openapi.ScoreboardEntryResponse {
	res := openapi.ScoreboardEntryResponse{
		TeamID:   httputil.Ptr(e.TeamID.String()),
		TeamName: httputil.Ptr(e.TeamName),
		Points:   httputil.Ptr(e.Points),
	}
	if !e.SolvedAt.IsZero() {
		res.LastSolved = httputil.Ptr(e.SolvedAt.Format(time.RFC3339))
	}
	return res
}

func FromScoreboardList(items []*domain.ScoreboardEntry) []openapi.ScoreboardEntryResponse {
	return lo.Map(items, func(item *domain.ScoreboardEntry, _ int) openapi.ScoreboardEntryResponse {
		return FromScoreboardEntry(item)
	})
}

func FromFirstBlood(fb *domain.FirstBloodEntry) openapi.FirstBloodResponse {
	return openapi.FirstBloodResponse{
		UserID:   httputil.Ptr(fb.UserID.String()),
		Username: httputil.Ptr(fb.Username),
		TeamID:   httputil.Ptr(fb.TeamID.String()),
		TeamName: httputil.Ptr(fb.TeamName),
		SolvedAt: httputil.Ptr(fb.SolvedAt.Format(time.RFC3339)),
	}
}

func FromChallengeDetail(d *usecase.ChallengeDetail) openapi.ChallengeDetailResponse {
	res := openapi.ChallengeDetailResponse{
		ID:             httputil.Ptr(d.Challenge.ID.String()),
		Title:          httputil.Ptr(d.Challenge.Title),
		Description:    httputil.Ptr(d.Challenge.Description),
		Category:       httputil.Ptr(d.Challenge.Category),
		ConnectionInfo: httputil.Ptr(d.Challenge.ConnectionInfo),
		MaxAttempts:    httputil.Ptr(d.Challenge.MaxAttempts),
		Position:       httputil.Ptr(d.Challenge.Position),
		State:          ptrChallengeDetailResponseState(d.Challenge.State),
		Points:         httputil.Ptr(d.Challenge.Points),
		SolveCount:     httputil.Ptr(d.SolveCount),
		SolvedByMe:     httputil.Ptr(d.SolvedByMe),
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
		res.FirstBlood = httputil.Ptr(FromFirstBlood(d.FirstBlood))
	}

	return res
}

func FromChallengeSolves(solves []*domain.SolveWithDetails) []openapi.ChallengeSolveEntry {
	res := make([]openapi.ChallengeSolveEntry, len(solves))
	for i, s := range solves {
		res[i] = openapi.ChallengeSolveEntry{
			TeamID:   httputil.Ptr(s.TeamID.String()),
			TeamName: httputil.Ptr(s.TeamName),
			SolvedAt: httputil.Ptr(s.SolvedAt),
		}
	}
	return res
}

func FromHintWithUnlockStatus(h *usecase.HintWithUnlockStatus) openapi.HintItem {
	res := openapi.HintItem{
		ID:         httputil.Ptr(h.Hint.ID.String()),
		Title:      httputil.Ptr(h.Hint.Title),
		Cost:       httputil.Ptr(h.Hint.Cost),
		OrderIndex: httputil.Ptr(h.Hint.OrderIndex),
		Unlocked:   httputil.Ptr(h.Unlocked),
	}
	if h.Unlocked {
		res.Content = httputil.Ptr(h.Hint.Content)
	}
	return res
}

func FromChallengeRequirements(items []*domain.ChallengeRequirement) []openapi.ChallengeRequirementResponse {
	res := make([]openapi.ChallengeRequirementResponse, len(items))
	for i, item := range items {
		res[i] = openapi.ChallengeRequirementResponse{
			ChallengeID:       httputil.Ptr(item.ChallengeID.String()),
			ChallengeTitle:    httputil.Ptr(item.ChallengeTitle),
			ChallengeCategory: item.Category,
		}
	}
	return res
}

func FromChallengeSolution(sol *domain.ChallengeSolution, downloadURLs map[string]string) openapi.ChallengeSolutionResponse {
	res := openapi.ChallengeSolutionResponse{
		ChallengeID: httputil.Ptr(sol.ChallengeID.String()),
		Content:     httputil.Ptr(sol.Content),
	}
	if len(sol.Files) > 0 {
		files := make([]openapi.FileItem, len(sol.Files))
		for i, f := range sol.Files {
			item := FromFile(f)
			if url, ok := downloadURLs[f.ID.String()]; ok {
				item.URL = httputil.Ptr(url)
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
		ChallengeID:       httputil.Ptr(entry.ChallengeID.String()),
		ChallengeTitle:    httputil.Ptr(entry.ChallengeTitle),
		ChallengeCategory: httputil.Ptr(entry.ChallengeCategory),
		Content:           httputil.Ptr(entry.Content),
	}
	if len(entry.Files) > 0 {
		files := make([]openapi.FileItem, len(entry.Files))
		for i, f := range entry.Files {
			item := FromFile(f)
			if url, ok := downloadURLs[f.ID.String()]; ok {
				item.URL = httputil.Ptr(url)
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
		IsRegex:           httputil.Ptr(flags.IsRegex),
		IsCaseInsensitive: httputil.Ptr(flags.IsCaseInsensitive),
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
