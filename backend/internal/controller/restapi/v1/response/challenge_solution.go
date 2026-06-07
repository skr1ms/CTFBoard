package response

import (
	"github.com/samber/lo"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func FromChallengeSolution(sol *domain.ChallengeSolution, downloadURLs map[string]string) openapi.ChallengeSolutionResponse {
	res := openapi.ChallengeSolutionResponse{
		ChallengeID: new(sol.ChallengeID.String()),
		Content:     new(sol.Content),
		State:       (*openapi.ChallengeSolutionResponseState)(&sol.State),
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

func ChallengeSolutionEntryFiles(entries []*domain.ChallengeSolutionEntry) []*domain.File {
	var files []*domain.File

	for _, entry := range entries {
		files = append(files, entry.Files...)
	}

	return files
}

func FromChallengeSolutionEntry(entry *domain.ChallengeSolutionEntry, downloadURLs map[string]string) openapi.ChallengeSolutionEntry {
	res := openapi.ChallengeSolutionEntry{
		ChallengeID:       new(entry.ChallengeID.String()),
		ChallengeTitle:    new(entry.ChallengeTitle),
		ChallengeCategory: new(entry.ChallengeCategory),
		Content:           new(entry.Content),
		State:             (*openapi.ChallengeSolutionEntryState)(&entry.State),
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

func EmptyChallengeSolutionEntryList() []openapi.ChallengeSolutionEntry {
	return []openapi.ChallengeSolutionEntry{}
}
