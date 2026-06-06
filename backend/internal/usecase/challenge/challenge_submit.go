package challenge

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/cacheutil"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/guard"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/crypto"
)

const (
	// submitMinCheckDuration is the minimum time spent on flag validation to mitigate timing attacks on flag comparison.
	submitMinCheckDuration = 75 * time.Millisecond
	MaxFlagLength          = 200
	regexMatchTimeout      = 500 * time.Millisecond
	maxConcurrentRegex     = 100
)

type submitContext struct {
	ctx         context.Context
	challengeID uuid.UUID
	flag        string
	userID      uuid.UUID
	teamID      uuid.UUID
	clientIP    string
	team        *domain.Team
	comp        *domain.Competition
}

// SubmitFlag orchestrates the full flag-submission pipeline
//  1. Verifies competition time window and that the caller belongs to a team
//  2. Checks user and team ban status, competition mode constraints (solo/teams/min-size)
//  3. Fetches the challenge via a singleflight-deduplicated cache read and validates its
//     state (hidden -> not found, locked -> locked error)
//  4. Ensures all prerequisite challenges are solved by the team
//  5. Short-circuits early when the per-window attempt cap is already reached
//  6. Validates the submitted flag against the challenge's (or competition's) flag-format
//     regex if one is configured
//  7. Checks the flag: HMAC-SHA256 comparison for plain flags, decrypted-regex match for
//     regex challenges; timing-attack mitigation pads the check to submitMinCheckDuration
//  8. On an incorrect flag, records the attempt and re-enforces the attempt cap atomically
//  9. On a correct flag, runs submitRecordSolve to atomically record the solve, then
//     invalidates scoreboard/challenge-list caches (freeze-aware) and broadcasts the event
//
//nolint:funlen // submission flow: validation, lock, solve check, create, broadcast
func (uc *ChallengeUseCase) SubmitFlag(ctx context.Context, challengeID uuid.UUID, flag string, userID uuid.UUID, teamID *uuid.UUID, clientIP string) (bool, error) {
	if teamID == nil {
		return false, apperr.ErrUserNotInTeam
	}

	comp, err := uc.submitCheckCompetitionTime(ctx)
	if err != nil {
		return false, fmt.Errorf("ChallengeUseCase - SubmitFlag - submitCheckCompetitionTime: %w", err)
	}

	sc := &submitContext{ctx: ctx, challengeID: challengeID, flag: crypto.NormalizeFlagInput(strings.TrimSpace(flag)), userID: userID, teamID: *teamID, clientIP: clientIP, comp: comp}
	if sc.flag == "" {
		return false, apperr.ErrInvalidFlagFormat
	}

	if len(sc.flag) > MaxFlagLength {
		return false, apperr.ErrInvalidFlagFormat
	}

	if uc.deps.UserRepo != nil {
		user, err := uc.deps.UserRepo.GetByID(ctx, userID)
		if err != nil {
			return false, fmt.Errorf("ChallengeUseCase - SubmitFlag - UserRepo.GetByID: %w", err)
		}

		var team *domain.Team

		if uc.deps.TeamRepo != nil {
			team, err = uc.deps.TeamRepo.GetByID(ctx, sc.teamID)
			if err != nil {
				return false, fmt.Errorf("ChallengeUseCase - SubmitFlag - TeamRepo.GetByID: %w", err)
			}

			sc.team = team
		}

		if err := guard.ValidateSubmissionEligibility(ctx, user, team, comp, uc.deps.TeamRepo); err != nil {
			return false, err
		}
	} else if uc.deps.TeamRepo != nil {
		team, err := uc.deps.TeamRepo.GetByID(ctx, sc.teamID)
		if err != nil {
			return false, fmt.Errorf("ChallengeUseCase - SubmitFlag - TeamRepo.GetByID: %w", err)
		}

		sc.team = team

		if err := guard.ValidateSubmissionEligibility(ctx, nil, team, comp, uc.deps.TeamRepo); err != nil {
			return false, err
		}
	}

	challenge, err := uc.submitGetChallenge(ctx, sc)
	if err != nil {
		if errors.Is(err, apperr.ErrChallengeNotFound) {
			return false, apperr.ErrChallengeNotFound
		}

		return false, fmt.Errorf("ChallengeUseCase - SubmitFlag - submitGetChallenge: %w", err)
	}

	if challenge.State == domain.ChallengeStateLocked {
		return false, apperr.ErrChallengeLocked
	}

	met, err := requirementsMet(ctx, sc.challengeID, sc.teamID, uc.deps.ChallengeRepo, uc.deps.SolveRepo)
	if err != nil {
		return false, fmt.Errorf("ChallengeUseCase - SubmitFlag - requirementsMet: %w", err)
	}

	if !met {
		return false, apperr.ErrRequirementsNotMet
	}

	if challenge.MaxAttempts > 0 && uc.deps.SubmissionRepo != nil {
		count, err := uc.countAttempts(ctx, sc.teamID, sc.challengeID, challenge.MaxAttemptsWindow)
		if err != nil {
			return false, fmt.Errorf("ChallengeUseCase - SubmitFlag - countAttempts: %w", err)
		}

		if count >= int64(challenge.MaxAttempts) {
			return false, apperr.ErrMaxAttemptsReached
		}
	}

	if err := uc.submitValidateFlagFormat(sc, challenge); err != nil {
		return false, fmt.Errorf("ChallengeUseCase - SubmitFlag - submitValidateFlagFormat: %w", err)
	}

	checkStart := time.Now()
	correct, err := uc.submitCheckFlag(sc, challenge)

	if elapsed := time.Since(checkStart); elapsed < submitMinCheckDuration {
		select {
		case <-time.After(submitMinCheckDuration - elapsed):
		case <-sc.ctx.Done():
			return false, fmt.Errorf("ChallengeUseCase - SubmitFlag - context: %w", sc.ctx.Err())
		}
	}

	if err != nil {
		return false, fmt.Errorf("ChallengeUseCase - SubmitFlag - checkFlag: %w", err)
	}

	if !correct {
		err := uc.submitLogIncorrectAndEnforceMaxAttempts(sc, challenge)
		if err != nil {
			if errors.Is(err, apperr.ErrMaxAttemptsReached) {
				return false, apperr.ErrMaxAttemptsReached
			}

			return false, fmt.Errorf("ChallengeUseCase - SubmitFlag - submitLogIncorrectAndEnforceMaxAttempts: %w", err)
		}

		return false, nil
	}

	solvedChallenge, solveCount, alreadySolved, wasFrozen, err := uc.submitRecordSolve(sc, challenge)
	if err != nil {
		if errors.Is(err, apperr.ErrRequirementsNotMet) {
			return false, apperr.ErrRequirementsNotMet
		}

		if errors.Is(err, apperr.ErrMaxAttemptsReached) {
			return false, apperr.ErrMaxAttemptsReached
		}

		if errors.Is(err, apperr.ErrAlreadySolved) {
			return true, nil
		}

		return false, fmt.Errorf("ChallengeUseCase - SubmitFlag - submitRecordSolve: %w", err)
	}

	if alreadySolved {
		return true, nil
	}

	uc.submitInvalidateCacheWithFrozenStatus(ctx, sc.teamID, wasFrozen)
	uc.submitNotifySolve(sc, solvedChallenge, solveCount == 1, wasFrozen)

	return true, nil
}

// submitCheckCompetitionTime loads the current competition and returns an error if submissions are not allowed.
// Uses CompUC when available (cached), falling back to CompRepo for direct DB access.
func (uc *ChallengeUseCase) submitCheckCompetitionTime(ctx context.Context) (*domain.Competition, error) {
	if uc.deps.CompUC == nil && uc.deps.CompRepo == nil {
		return nil, nil
	}

	var (
		comp *domain.Competition
		err  error
	)

	if uc.deps.CompUC != nil {
		comp, err = uc.deps.CompUC.Get(ctx)
	} else {
		comp, err = uc.deps.CompRepo.Get(ctx)
	}

	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - SubmitFlag - CompetitionRepo.Get: %w", err)
	}

	if !comp.IsSubmissionAllowed() {
		return nil, apperr.ErrSubmissionNotAllowed
	}

	return comp, nil
}

// submitGetChallenge loads the challenge by ID using singleflight deduplication to avoid
// stampeding the DB under concurrent submissions for the same challenge.
func (uc *ChallengeUseCase) submitGetChallenge(ctx context.Context, sc *submitContext) (*domain.Challenge, error) {
	key := sc.challengeID.String()

	v, err, _ := uc.challengeFetchSf.Do(key, func() (any, error) {
		loadCtx, cancel := cacheutil.LoaderContext(ctx)
		defer cancel()

		return uc.deps.ChallengeRepo.GetByID(loadCtx, sc.challengeID)
	})
	if err != nil {
		if errors.Is(err, apperr.ErrChallengeNotFound) {
			return nil, apperr.ErrChallengeNotFound
		}

		return nil, fmt.Errorf("ChallengeUseCase - SubmitFlag - ChallengeRepo.GetByID: %w", err)
	}

	challenge, ok := v.(*domain.Challenge)
	if !ok {
		return nil, fmt.Errorf("ChallengeUseCase - SubmitFlag - ChallengeRepo.GetByID: unexpected type")
	}

	if err := guard.EnsureChallengeVisible(challenge); err != nil {
		return nil, err
	}

	return challenge, nil
}
