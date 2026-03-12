package response

import (
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

func FromChallenge(c *entity.Challenge) openapi.ChallengeResponse {
	if c == nil {
		return openapi.ChallengeResponse{}
	}
	return openapi.ChallengeResponse{
		ID:          ptr(c.ID.String()),
		Title:       ptr(c.Title),
		Description: ptr(c.Description),
		Category:    ptr(c.Category),
		Points:      ptr(c.Points),
		SolveCount:  ptr(c.SolveCount),
		IsHidden:    ptr(c.IsHidden),
	}
}

func FromChallengeWithSolved(cws *entity.ChallengeWithSolved) openapi.ChallengeResponse {
	if cws == nil {
		return openapi.ChallengeResponse{}
	}
	res := FromChallenge(cws.Challenge)
	res.Solved = ptr(cws.Solved)
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
	res := make([]openapi.ChallengeResponse, len(items))
	for i, item := range items {
		res[i] = FromChallengeWithTags(item)
	}
	return res
}

func FromTag(t *entity.Tag) openapi.TagResponse {
	return openapi.TagResponse{
		ID:    ptr(t.ID.String()),
		Name:  ptr(t.Name),
		Color: ptr(t.Color),
	}
}

func FromTagList(items []*entity.Tag) []openapi.TagResponse {
	res := make([]openapi.TagResponse, len(items))
	for i, item := range items {
		res[i] = FromTag(item)
	}
	return res
}

func FromScoreboardEntry(e *entity.ScoreboardEntry) openapi.ScoreboardEntryResponse {
	res := openapi.ScoreboardEntryResponse{
		TeamID:   ptr(e.TeamID.String()),
		TeamName: ptr(e.TeamName),
		Points:   ptr(e.Points),
	}
	if !e.SolvedAt.IsZero() {
		res.LastSolved = ptr(e.SolvedAt.Format(time.RFC3339))
	}
	return res
}

func FromScoreboardList(items []*entity.ScoreboardEntry) []openapi.ScoreboardEntryResponse {
	res := make([]openapi.ScoreboardEntryResponse, len(items))
	for i, item := range items {
		res[i] = FromScoreboardEntry(item)
	}
	return res
}

func FromFirstBlood(fb *entity.FirstBloodEntry) openapi.FirstBloodResponse {
	return openapi.FirstBloodResponse{
		UserID:   ptr(fb.UserID.String()),
		Username: ptr(fb.Username),
		TeamID:   ptr(fb.TeamID.String()),
		TeamName: ptr(fb.TeamName),
		SolvedAt: ptr(fb.SolvedAt.Format(time.RFC3339)),
	}
}

func FromChallengeDetail(d *usecase.ChallengeDetail) openapi.ChallengeDetailResponse {
	res := openapi.ChallengeDetailResponse{
		ID:          ptr(d.Challenge.ID.String()),
		Title:       ptr(d.Challenge.Title),
		Description: ptr(d.Challenge.Description),
		Category:    ptr(d.Challenge.Category),
		Points:      ptr(d.Challenge.Points),
		SolveCount:  ptr(d.SolveCount),
		SolvedByMe:  ptr(d.SolvedByMe),
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
		res.FirstBlood = ptr(FromFirstBlood(d.FirstBlood))
	}

	return res
}

func FromChallengeSolves(solves []*entity.SolveWithDetails) []openapi.ChallengeSolveEntry {
	res := make([]openapi.ChallengeSolveEntry, len(solves))
	for i, s := range solves {
		res[i] = openapi.ChallengeSolveEntry{
			TeamID:   ptr(s.TeamID.String()),
			TeamName: ptr(s.TeamName),
			SolvedAt: ptr(s.SolvedAt),
		}
	}
	return res
}

func FromHintWithUnlockStatus(h *usecase.HintWithUnlockStatus) openapi.HintItem {
	res := openapi.HintItem{
		ID:         ptr(h.Hint.ID.String()),
		Cost:       ptr(h.Hint.Cost),
		OrderIndex: ptr(h.Hint.OrderIndex),
		Unlocked:   ptr(h.Unlocked),
	}
	if h.Unlocked {
		res.Content = ptr(h.Hint.Content)
	}
	return res
}

func FromChallengeRequirements(items []*entity.ChallengeRequirement) []openapi.ChallengeRequirementResponse {
	res := make([]openapi.ChallengeRequirementResponse, len(items))
	for i, item := range items {
		res[i] = openapi.ChallengeRequirementResponse{
			ChallengeID:       ptr(item.ChallengeID.String()),
			ChallengeTitle:    ptr(item.ChallengeTitle),
			ChallengeCategory: item.Category,
		}
	}
	return res
}

func FromChallengeSolution(sol *entity.ChallengeSolution, downloadURLs map[string]string) openapi.ChallengeSolutionResponse {
	res := openapi.ChallengeSolutionResponse{
		ChallengeID: ptr(sol.ChallengeID.String()),
		Content:     ptr(sol.Content),
	}
	if len(sol.Files) > 0 {
		files := make([]openapi.FileItem, len(sol.Files))
		for i, f := range sol.Files {
			item := FromFile(f)
			if url, ok := downloadURLs[f.ID.String()]; ok {
				item.URL = ptr(url)
			}
			files[i] = item
		}
		res.Files = &files
	}
	return res
}

func FromChallengeSolutionEntryList(entries []*entity.ChallengeSolutionEntry, downloadURLs map[string]string) []openapi.ChallengeSolutionEntry {
	res := make([]openapi.ChallengeSolutionEntry, len(entries))
	for i, entry := range entries {
		res[i] = FromChallengeSolutionEntry(entry, downloadURLs)
	}
	return res
}

func FromChallengeSolutionEntry(entry *entity.ChallengeSolutionEntry, downloadURLs map[string]string) openapi.ChallengeSolutionEntry {
	res := openapi.ChallengeSolutionEntry{
		ChallengeID:       ptr(entry.ChallengeID.String()),
		ChallengeTitle:    ptr(entry.ChallengeTitle),
		ChallengeCategory: ptr(entry.ChallengeCategory),
		Content:           ptr(entry.Content),
	}
	if len(entry.Files) > 0 {
		files := make([]openapi.FileItem, len(entry.Files))
		for i, f := range entry.Files {
			item := FromFile(f)
			if url, ok := downloadURLs[f.ID.String()]; ok {
				item.URL = ptr(url)
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

func FromChallengeFlags(flags *entity.ChallengeFlags) openapi.ChallengeFlagsResponse {
	res := openapi.ChallengeFlagsResponse{
		Flags:             &[]string{flags.FlagHash},
		IsRegex:           ptr(flags.IsRegex),
		IsCaseInsensitive: ptr(flags.IsCaseInsensitive),
	}
	if flags.FlagRegex != nil && *flags.FlagRegex != "" {
		res.FlagRegex = flags.FlagRegex
	}
	if flags.FlagFormatRegex != nil {
		res.FlagFormatRegex = flags.FlagFormatRegex
	}
	return res
}

func FromChallengeEntityList(challenges []*entity.Challenge) []openapi.ChallengeResponse {
	res := make([]openapi.ChallengeResponse, len(challenges))
	for i, c := range challenges {
		res[i] = FromChallenge(c)
	}
	return res
}

func FromChallengeTypes(types []string) []string {
	return types
}

func EmptyChallengeSolutionEntryList() []openapi.ChallengeSolutionEntry {
	return []openapi.ChallengeSolutionEntry{}
}
