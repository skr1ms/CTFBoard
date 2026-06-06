package backup

import (
	"context"
	"fmt"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func (uc *BackupUseCase) exportBrackets(ctx context.Context, backup *domain.BackupData, mu *sync.Mutex, g *errgroup.Group) {
	if uc.deps.BracketRepo == nil {
		return
	}

	g.Go(func() error {
		list, err := uc.deps.BracketRepo.GetAll(ctx)
		if err != nil {
			return fmt.Errorf("BackupUseCase - Export - BracketRepo.GetAll: %w", err)
		}

		out := make([]domain.Bracket, len(list))
		for i, b := range list {
			out[i] = *b
		}

		mu.Lock()
		backup.Brackets = out
		mu.Unlock()

		return nil
	})
}

func (uc *BackupUseCase) exportChallengeRequirements(ctx context.Context, backup *domain.BackupData, mu *sync.Mutex, g *errgroup.Group) {
	g.Go(func() error {
		list, err := uc.deps.ChallengeRepo.GetAllRequirementPairs(ctx)
		if err != nil {
			return fmt.Errorf("BackupUseCase - Export - ChallengeRepo.GetAllRequirementPairs: %w", err)
		}

		out := make([]domain.ChallengeRequirementPair, len(list))
		for i, p := range list {
			out[i] = *p
		}

		mu.Lock()
		backup.ChallengeRequirements = out
		mu.Unlock()

		return nil
	})
}

func (uc *BackupUseCase) exportSolutions(ctx context.Context, backup *domain.BackupData, mu *sync.Mutex, g *errgroup.Group) {
	g.Go(func() error {
		list, err := uc.deps.ChallengeRepo.GetAllSolutions(ctx)
		if err != nil {
			return fmt.Errorf("BackupUseCase - Export - ChallengeRepo.GetAllSolutions: %w", err)
		}

		out := make([]domain.SolutionBackup, len(list))
		for i, s := range list {
			out[i] = *s
		}

		mu.Lock()
		backup.Solutions = out
		mu.Unlock()

		return nil
	})
}

func (uc *BackupUseCase) exportComments(ctx context.Context, backup *domain.BackupData, mu *sync.Mutex, g *errgroup.Group) {
	if uc.deps.CommentRepo == nil {
		return
	}

	g.Go(func() error {
		list, err := uc.deps.CommentRepo.GetAll(ctx)
		if err != nil {
			return fmt.Errorf("BackupUseCase - Export - CommentRepo.GetAll: %w", err)
		}

		out := make([]domain.Comment, len(list))
		for i, c := range list {
			out[i] = *c
		}

		mu.Lock()
		backup.Comments = out
		mu.Unlock()

		return nil
	})
}

func (uc *BackupUseCase) exportRatings(ctx context.Context, backup *domain.BackupData, mu *sync.Mutex, g *errgroup.Group) {
	if uc.deps.RatingRepo == nil {
		return
	}

	g.Go(func() error {
		list, err := uc.deps.RatingRepo.GetAll(ctx)
		if err != nil {
			return fmt.Errorf("BackupUseCase - Export - RatingRepo.GetAll: %w", err)
		}

		out := make([]domain.Rating, len(list))
		for i, r := range list {
			out[i] = *r
		}

		mu.Lock()
		backup.Ratings = out
		mu.Unlock()

		return nil
	})
}

func (uc *BackupUseCase) exportFields(ctx context.Context, backup *domain.BackupData, mu *sync.Mutex, g *errgroup.Group) {
	if uc.deps.FieldRepo == nil {
		return
	}

	g.Go(func() error {
		list, err := uc.deps.FieldRepo.GetAll(ctx)
		if err != nil {
			return fmt.Errorf("BackupUseCase - Export - FieldRepo.GetAll: %w", err)
		}

		out := make([]domain.Field, len(list))
		for i, f := range list {
			out[i] = *f
		}

		mu.Lock()
		backup.Fields = out
		mu.Unlock()

		return nil
	})
}

func (uc *BackupUseCase) exportFieldValues(ctx context.Context, backup *domain.BackupData, mu *sync.Mutex, g *errgroup.Group) {
	if uc.deps.FieldValueRepo == nil {
		return
	}

	g.Go(func() error {
		list, err := uc.deps.FieldValueRepo.GetAll(ctx)
		if err != nil {
			return fmt.Errorf("BackupUseCase - Export - FieldValueRepo.GetAll: %w", err)
		}

		out := make([]domain.FieldValue, len(list))
		for i, v := range list {
			out[i] = *v
		}

		mu.Lock()
		backup.FieldValues = out
		mu.Unlock()

		return nil
	})
}

func (uc *BackupUseCase) exportOptional(
	ctx context.Context,
	backup *domain.BackupData,
	opts domain.ExportOptions,
	mu *sync.Mutex,
	g *errgroup.Group,
) {
	uc.exportOptionalTeams(ctx, backup, opts, mu, g)
	uc.exportOptionalUsers(ctx, backup, opts, mu, g)
	uc.exportOptionalAwards(ctx, backup, opts, mu, g)
	uc.exportOptionalSolves(ctx, backup, opts, mu, g)
	uc.exportOptionalHintUnlocks(ctx, backup, opts, mu, g)
	uc.exportOptionalFiles(ctx, backup, opts, mu, g)
}

func (uc *BackupUseCase) exportOptionalTeams(ctx context.Context, backup *domain.BackupData, opts domain.ExportOptions, mu *sync.Mutex, g *errgroup.Group) {
	if !opts.IncludeTeams {
		return
	}

	g.Go(func() error {
		teams, err := uc.fetchTeamsWithMembers(ctx)
		if err != nil {
			return fmt.Errorf("BackupUseCase - Export - fetchTeamsWithMembers: %w", err)
		}

		mu.Lock()
		backup.Teams = teams
		mu.Unlock()

		return nil
	})
}

func (uc *BackupUseCase) exportOptionalUsers(ctx context.Context, backup *domain.BackupData, opts domain.ExportOptions, mu *sync.Mutex, g *errgroup.Group) {
	if !opts.IncludeUsers {
		return
	}

	g.Go(func() error {
		users, err := uc.fetchUsers(ctx)
		if err != nil {
			return fmt.Errorf("BackupUseCase - Export - fetchUsers: %w", err)
		}

		mu.Lock()
		backup.Users = users
		mu.Unlock()

		return nil
	})
}

func (uc *BackupUseCase) exportOptionalAwards(ctx context.Context, backup *domain.BackupData, opts domain.ExportOptions, mu *sync.Mutex, g *errgroup.Group) {
	if !opts.IncludeAwards {
		return
	}

	g.Go(func() error {
		awards, err := uc.fetchAwards(ctx)
		if err != nil {
			return fmt.Errorf("BackupUseCase - Export - fetchAwards: %w", err)
		}

		mu.Lock()
		backup.Awards = awards
		mu.Unlock()

		return nil
	})
}

func (uc *BackupUseCase) exportOptionalSolves(ctx context.Context, backup *domain.BackupData, opts domain.ExportOptions, mu *sync.Mutex, g *errgroup.Group) {
	if !opts.IncludeSolves {
		return
	}

	g.Go(func() error {
		solves, err := uc.fetchSolves(ctx)
		if err != nil {
			return fmt.Errorf("BackupUseCase - Export - fetchSolves: %w", err)
		}

		mu.Lock()
		backup.Solves = solves
		mu.Unlock()

		return nil
	})
}

func (uc *BackupUseCase) exportOptionalHintUnlocks(ctx context.Context, backup *domain.BackupData, opts domain.ExportOptions, mu *sync.Mutex, g *errgroup.Group) {
	if !opts.IncludeHintUnlocks {
		return
	}

	g.Go(func() error {
		unlocks, err := uc.deps.HintRepo.GetAllUnlocksForBackup(ctx)
		if err != nil {
			return fmt.Errorf("BackupUseCase - Export - HintRepo.GetAllUnlocksForBackup: %w", err)
		}

		result := make([]domain.HintUnlock, len(unlocks))
		for i, u := range unlocks {
			result[i] = *u
		}

		mu.Lock()
		backup.HintUnlocks = result
		mu.Unlock()

		return nil
	})
}

func (uc *BackupUseCase) exportOptionalFiles(ctx context.Context, backup *domain.BackupData, opts domain.ExportOptions, mu *sync.Mutex, g *errgroup.Group) {
	if !opts.IncludeFiles {
		return
	}

	g.Go(func() error {
		files, err := uc.fetchFiles(ctx)
		if err != nil {
			return fmt.Errorf("BackupUseCase - Export - fetchFiles: %w", err)
		}

		mu.Lock()
		backup.Files = files
		mu.Unlock()

		return nil
	})
}
