package challenge

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/cacheutil"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/computil"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/guard"
)

func (uc *ChallengeUseCase) shouldAnonymizePrerequisites(ctx context.Context) bool {
	if uc.deps.CompParamUC == nil {
		return false
	}

	return uc.deps.CompParamUC.GetBool(ctx, "challenge_prerequisite_anonymize", false)
}

// anonymizedChallengeDetail returns a placeholder ChallengeDetail for a challenge whose
// prerequisites are unmet and the anonymize flag is enabled. Title and description are
// replaced with "???", ConnectionInfo is cleared, and state is forced to Locked so the
// client shows a locked card rather than a 404.
func anonymizedChallengeDetail(c *domain.Challenge) *usecase.ChallengeDetail {
	hidden := "???"

	masked := *c
	masked.Title = hidden
	masked.Description = hidden
	masked.ConnectionInfo = ""
	masked.State = domain.ChallengeStateLocked

	return &usecase.ChallengeDetail{
		Challenge:  &masked,
		Tags:       []*domain.Tag{},
		Files:      []*domain.File{},
		Hints:      []*usecase.HintWithUnlockStatus{},
		SolvedByMe: false,
		SolveCount: c.SolveCount,
	}
}

func (uc *ChallengeUseCase) GetByID(ctx context.Context, challengeID uuid.UUID) (*domain.Challenge, error) {
	challenge, err := uc.deps.ChallengeRepo.GetByID(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetByID - ChallengeRepo.GetByID: %w", err)
	}

	return challenge, nil
}

// GetDetail is the singleflight-wrapped entry point for challenge detail. The
// key includes both challengeID and teamID so concurrent requests for the same
// pair share one bounded loader instead of binding shared work to the first caller.
func (uc *ChallengeUseCase) GetDetail(ctx context.Context, challengeID uuid.UUID, teamID *uuid.UUID) (*usecase.ChallengeDetail, error) {
	teamIDStr := ""

	if teamID != nil {
		teamIDStr = teamID.String()
	}

	key := fmt.Sprintf("challenge_detail:%s:%s", challengeID, teamIDStr)

	v, err, _ := uc.challengeDetailSf.Do(key, func() (any, error) {
		loadCtx, cancel := cacheutil.LoaderContext(ctx)
		defer cancel()

		return uc.getDetailInner(loadCtx, challengeID, teamID)
	})
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetDetail: %w", err)
	}

	d, ok := v.(*usecase.ChallengeDetail)
	if !ok {
		return nil, fmt.Errorf("ChallengeUseCase - GetDetail: unexpected type from singleflight")
	}

	return d, nil
}

// getDetailInner fetches a challenge's full detail, gating access by prerequisite
// satisfaction before spawning a fan-out errgroup that loads tags, challenge files,
// hints (with unlock status), first-blood entry, and the team's solved status in
// parallel. After the errgroup completes it resolves the freeze-aware solve count: if the
// competition freeze is active it queries solves up to the freeze timestamp instead of
// using the live counter stored on the challenge row. When prerequisites are unmet and the
// challenge_prerequisite_anonymize flag is enabled, it returns an anonymized placeholder
// instead of ErrChallengeNotFound so that the challenge list entry remains visible.
func (uc *ChallengeUseCase) getDetailInner(ctx context.Context, challengeID uuid.UUID, teamID *uuid.UUID) (*usecase.ChallengeDetail, error) {
	challenge, err := uc.deps.ChallengeRepo.GetByID(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetDetail - ChallengeRepo.GetByID: %w", err)
	}

	if err := guard.EnsureChallengeVisible(challenge); err != nil {
		return nil, err
	}

	reqs, err := uc.deps.ChallengeRepo.GetRequirements(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetDetail - GetRequirements: %w", err)
	}

	if len(reqs) > 0 {
		if teamID == nil || uc.deps.SolveRepo == nil {
			if uc.shouldAnonymizePrerequisites(ctx) {
				return anonymizedChallengeDetail(challenge), nil
			}

			return nil, apperr.ErrChallengeNotFound
		}

		met, err := requirementsMet(ctx, challengeID, *teamID, uc.deps.ChallengeRepo, uc.deps.SolveRepo)
		if err != nil {
			return nil, fmt.Errorf("ChallengeUseCase - GetDetail - requirementsMet: %w", err)
		}

		if !met {
			if uc.shouldAnonymizePrerequisites(ctx) {
				return anonymizedChallengeDetail(challenge), nil
			}

			return nil, apperr.ErrChallengeNotFound
		}
	}

	var (
		tags       []*domain.Tag
		files      []*domain.File
		hints      []*usecase.HintWithUnlockStatus
		firstBlood *domain.FirstBloodEntry
		solvedByMe bool
	)

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error

		tags, err = uc.getChallengeTags(gCtx, challengeID)
		if err != nil {
			return fmt.Errorf("ChallengeUseCase - GetDetail - getChallengeTags: %w", err)
		}

		return nil
	})
	g.Go(func() error {
		var err error

		files, err = uc.getChallengeFiles(gCtx, challengeID)
		if err != nil {
			return fmt.Errorf("ChallengeUseCase - GetDetail - getChallengeFiles: %w", err)
		}

		return nil
	})
	g.Go(func() error {
		var err error

		hints, err = uc.getChallengeHints(gCtx, challengeID, teamID)
		if err != nil {
			return fmt.Errorf("ChallengeUseCase - GetDetail - getChallengeHints: %w", err)
		}

		return nil
	})
	g.Go(func() error {
		var err error

		firstBlood, err = uc.getChallengeFirstBlood(gCtx, challengeID)
		if err != nil {
			return fmt.Errorf("ChallengeUseCase - GetDetail - getChallengeFirstBlood: %w", err)
		}

		return nil
	})
	g.Go(func() error {
		var err error

		solvedByMe, err = uc.checkChallengeSolved(gCtx, challengeID, teamID)
		if err != nil {
			return fmt.Errorf("ChallengeUseCase - GetDetail - checkChallengeSolved: %w", err)
		}

		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetDetail - errgroup.Wait: %w", err)
	}

	solveCount := challenge.SolveCount

	if uc.deps.SolveRepo != nil {
		comp := computil.Cached(ctx, uc.deps.CompUC, uc.deps.CompRepo)

		if comp != nil && comp.IsFreezeActive() {
			frozenSolves, err := uc.deps.SolveRepo.GetByChallengeID(ctx, challengeID, comp.FreezeTime)
			if err == nil {
				solveCount = len(frozenSolves)
			}
		}
	}

	return &usecase.ChallengeDetail{
		Challenge:  challenge,
		Tags:       tags,
		Files:      files,
		Hints:      hints,
		FirstBlood: firstBlood,
		SolvedByMe: solvedByMe,
		SolveCount: solveCount,
	}, nil
}

func (uc *ChallengeUseCase) getChallengeTags(ctx context.Context, challengeID uuid.UUID) ([]*domain.Tag, error) {
	if uc.deps.TagRepo == nil {
		return nil, nil
	}

	tags, err := uc.deps.TagRepo.GetByChallengeID(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetDetail - TagRepo.GetByChallengeID: %w", err)
	}

	return tags, nil
}

func (uc *ChallengeUseCase) getChallengeFiles(ctx context.Context, challengeID uuid.UUID) ([]*domain.File, error) {
	if uc.deps.FileRepo == nil {
		return nil, nil
	}

	files, err := uc.deps.FileRepo.GetByChallengeID(ctx, challengeID, domain.FileTypeChallenge)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetDetail - FileRepo.GetByChallengeID: %w", err)
	}

	return files, nil
}

// getChallengeHints returns hints with per-team unlock status via HintUC.
// Returns nil (not an error) when HintUC is not wired.
func (uc *ChallengeUseCase) getChallengeHints(ctx context.Context, challengeID uuid.UUID, teamID *uuid.UUID) ([]*usecase.HintWithUnlockStatus, error) {
	if uc.deps.HintUC == nil {
		return nil, nil
	}

	hints, err := uc.deps.HintUC.GetByChallengeID(ctx, challengeID, teamID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetDetail - HintRepo.GetByChallengeID: %w", err)
	}

	return hints, nil
}

// getChallengeFirstBlood returns the first-blood entry for a challenge with freeze
// awareness: when the competition freeze is active it queries solves up to FreezeTime
// via GetFirstBloodFrozen; otherwise uses the live GetFirstBlood. ErrSolveNotFound
// is treated as "no first blood yet" and returns nil without error.
func (uc *ChallengeUseCase) getChallengeFirstBlood(ctx context.Context, challengeID uuid.UUID) (*domain.FirstBloodEntry, error) {
	if uc.deps.SolveRepo == nil {
		return nil, nil
	}

	comp := computil.Cached(ctx, uc.deps.CompUC, uc.deps.CompRepo)

	var ft *time.Time

	if comp != nil && comp.IsFreezeActive() {
		ft = comp.FreezeTime
	}

	fb, err := uc.deps.SolveRepo.GetFirstBlood(ctx, challengeID, ft)
	if err != nil {
		if errors.Is(err, apperr.ErrSolveNotFound) {
			return nil, nil
		}

		return nil, fmt.Errorf("ChallengeUseCase - GetDetail - SolveRepo.GetFirstBlood: %w", err)
	}

	return fb, nil
}

// checkChallengeSolved returns true when the team has an existing solve for the challenge.
// ErrSolveNotFound is mapped to (false, nil); any other error is propagated.
func (uc *ChallengeUseCase) checkChallengeSolved(ctx context.Context, challengeID uuid.UUID, teamID *uuid.UUID) (bool, error) {
	if teamID == nil || uc.deps.SolveRepo == nil {
		return false, nil
	}

	_, err := uc.deps.SolveRepo.GetByTeamAndChallenge(ctx, *teamID, challengeID)
	if err == nil {
		return true, nil
	}

	if errors.Is(err, apperr.ErrSolveNotFound) {
		return false, nil
	}

	return false, fmt.Errorf("ChallengeUseCase - GetDetail - checkSolved: %w", err)
}

// GetSolves returns the solve list for a challenge, applying freeze-time filtering when active.
func (uc *ChallengeUseCase) GetSolves(ctx context.Context, challengeID uuid.UUID) ([]*domain.SolveWithDetails, error) {
	challenge, err := uc.deps.ChallengeRepo.GetByID(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetSolves - ChallengeRepo.GetByID: %w", err)
	}

	if err := guard.EnsureChallengeVisible(challenge); err != nil {
		return nil, err
	}

	comp, err := uc.deps.CompUC.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetSolves - CompUC.Get: %w", err)
	}

	var ft *time.Time

	if comp != nil && comp.IsFreezeActive() {
		ft = comp.FreezeTime
	}

	solves, err := uc.deps.SolveRepo.GetByChallengeID(ctx, challengeID, ft)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetSolves - SolveRepo.GetByChallengeID: %w", err)
	}

	return solves, nil
}
