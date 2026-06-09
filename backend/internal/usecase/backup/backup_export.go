package backup

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

const (
	exportTimeout         = 5 * time.Minute
	exportZIPCloseTimeout = 5 * time.Second
)

// Export assembles a BackupData snapshot by fetching all selected data categories
// in parallel using an errgroup bounded by a 5-minute context timeout. Each
// category (competition settings, challenges with hints and tags, brackets,
// requirements, solutions, comments, ratings, custom fields, and optionally
// teams, users, awards, solves, hint unlocks, and files) is fetched in its own
// goroutine; results are written to the BackupData struct under a shared mutex
// The function returns an error if any goroutine fails or the timeout expires.
func (uc *BackupUseCase) Export(ctx context.Context, opts domain.ExportOptions) (*domain.BackupData, error) {
	ctx, cancel := context.WithTimeout(ctx, exportTimeout)
	defer cancel()

	g, gCtx := errgroup.WithContext(ctx)
	backup := &domain.BackupData{
		Version:    domain.BackupVersion,
		ExportedAt: time.Now().UTC(),
	}

	var mu sync.Mutex

	g.Go(func() error {
		comp, err := uc.deps.CompetitionRepo.Get(gCtx)
		if err != nil {
			return fmt.Errorf("BackupUseCase - Export - CompetitionRepo.Get: %w", err)
		}

		mu.Lock()
		backup.Competition = comp
		mu.Unlock()

		return nil
	})

	g.Go(func() error {
		challenges, err := uc.fetchChallengesWithHints(gCtx)
		if err != nil {
			return fmt.Errorf("BackupUseCase - Export - fetchChallengesWithHints: %w", err)
		}

		if uc.deps.TagRepo != nil {
			challengeIDs := make([]uuid.UUID, len(challenges))
			for i, c := range challenges {
				challengeIDs[i] = c.ID
			}

			tagsByChallenge, err := uc.deps.TagRepo.GetByChallengeIDs(gCtx, challengeIDs)
			if err != nil {
				return fmt.Errorf("BackupUseCase - Export - TagRepo.GetByChallengeIDs: %w", err)
			}

			seenTags := make(map[uuid.UUID]domain.Tag)

			for i := range challenges {
				tags := tagsByChallenge[challenges[i].ID]

				tagIDs := make([]uuid.UUID, 0, len(tags))
				for _, t := range tags {
					seenTags[t.ID] = *t
					tagIDs = append(tagIDs, t.ID)
				}

				challenges[i].TagIDs = tagIDs
			}

			uniqueTags := make([]domain.Tag, 0, len(seenTags))
			for _, t := range seenTags {
				uniqueTags = append(uniqueTags, t)
			}

			mu.Lock()
			backup.Tags = uniqueTags
			mu.Unlock()
		}

		if uc.deps.TopicRepo != nil {
			challengeIDs := make([]uuid.UUID, len(challenges))
			for i, c := range challenges {
				challengeIDs[i] = c.ID
			}

			topicsByChallenge, err := uc.deps.TopicRepo.GetByChallengeIDs(gCtx, challengeIDs)
			if err != nil {
				return fmt.Errorf("BackupUseCase - Export - TopicRepo.GetByChallengeIDs: %w", err)
			}

			seenTopics := make(map[uuid.UUID]domain.Topic)

			for i := range challenges {
				topics := topicsByChallenge[challenges[i].ID]

				topicIDs := make([]uuid.UUID, 0, len(topics))
				for _, t := range topics {
					seenTopics[t.ID] = *t
					topicIDs = append(topicIDs, t.ID)
				}

				challenges[i].TopicIDs = topicIDs
			}

			uniqueTopics := make([]domain.Topic, 0, len(seenTopics))
			for _, t := range seenTopics {
				uniqueTopics = append(uniqueTopics, t)
			}

			mu.Lock()
			backup.Topics = uniqueTopics
			mu.Unlock()
		}

		mu.Lock()
		backup.Challenges = challenges
		mu.Unlock()

		return nil
	})

	uc.exportBrackets(gCtx, backup, &mu, g)
	uc.exportPages(gCtx, backup, &mu, g)
	uc.exportChallengeRequirements(gCtx, backup, &mu, g)
	uc.exportSolutions(gCtx, backup, &mu, g)
	uc.exportComments(gCtx, backup, &mu, g)
	uc.exportRatings(gCtx, backup, &mu, g)
	uc.exportFields(gCtx, backup, &mu, g)
	uc.exportFieldValues(gCtx, backup, &mu, g)
	uc.exportOptional(gCtx, backup, opts, &mu, g)

	err := g.Wait()
	if err != nil {
		return nil, fmt.Errorf("BackupUseCase - Export - errgroup.Wait: %w", err)
	}

	return backup, nil
}
