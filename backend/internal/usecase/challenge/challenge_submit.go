package challenge

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/semaphore"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/cache"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/scoring"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/computil"
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

var regexSem = semaphore.NewWeighted(maxConcurrentRegex)

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

// getCompiledRegex returns a compiled *regexp.Regexp for pattern, using an LRU cache to
// avoid redundant compilations. When multiple goroutines request the same pattern
// simultaneously, singleflight ensures only one compilation runs; the rest receive the
// shared result. A double-checked get inside the singleflight body prevents a race between
// the outer cache miss and the singleflight flight being started.
func (uc *ChallengeUseCase) getCompiledRegex(pattern string) (*regexp.Regexp, error) {
	if re, ok := uc.regexCache.Get(pattern); ok {
		return re, nil
	}

	v, err, _ := uc.regexSf.Do(pattern, func() (any, error) {
		if re, ok := uc.regexCache.Get(pattern); ok {
			return re, nil
		}

		compiled, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("ChallengeUseCase - getCompiledRegex - regexp.Compile: %w", err)
		}

		uc.regexCache.Set(pattern, compiled)

		return compiled, nil
	})
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - getCompiledRegex: %w", err)
	}

	re, ok := v.(*regexp.Regexp)
	if !ok {
		return nil, fmt.Errorf("ChallengeUseCase - getCompiledRegex: invalid type from singleflight")
	}

	return re, nil
}

// safeMatchString executes re.MatchString(input) with two layers of protection against
// ReDoS. First it acquires a slot from the global weighted semaphore (capacity
// maxConcurrentRegex) so that the number of simultaneously running regex goroutines is
// bounded. Then it launches the match in a separate goroutine and uses a select to honour
// the caller's context deadline or the explicit timeout, whichever fires first. The result
// channel is buffered (cap 1) so the goroutine can always send and call
// regexSem.Release even when the caller has already timed out and stopped receiving.
func safeMatchString(ctx context.Context, re *regexp.Regexp, input string, timeout time.Duration) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	err := regexSem.Acquire(ctx, 1)
	if err != nil {
		return false, fmt.Errorf("ChallengeUseCase - safeMatchString - acquire semaphore: %w", err)
	}

	ch := make(chan bool, 1)

	go func() {
		defer regexSem.Release(1)

		ch <- re.MatchString(input)
	}()
	// Buffered ch allows the goroutine to send and exit so Release runs even when caller times out
	select {
	case matched := <-ch:
		return matched, nil
	case <-ctx.Done():
		return false, fmt.Errorf("ChallengeUseCase - safeMatchString - match timed out: %w", ctx.Err())
	}
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

	challenge, err := uc.submitGetChallenge(sc)
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
		count, err := uc.countAttempts(sc.ctx, sc.teamID, sc.challengeID, challenge.MaxAttemptsWindow)
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

	uc.submitInvalidateCacheWithFrozenStatus(sc.ctx, sc.teamID, wasFrozen)
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
func (uc *ChallengeUseCase) submitGetChallenge(sc *submitContext) (*domain.Challenge, error) {
	key := sc.challengeID.String()

	v, err, _ := uc.challengeFetchSf.Do(key, func() (any, error) {
		return uc.deps.ChallengeRepo.GetByID(context.WithoutCancel(sc.ctx), sc.challengeID)
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

// submitCheckRequirementsInTx verifies that the team has solved all prerequisite challenges.
// Called inside the submit transaction so the check is consistent with the advisory lock.
func (uc *ChallengeUseCase) submitCheckRequirementsInTx(ctx context.Context, challengeID, teamID uuid.UUID) error {
	requirements, err := uc.deps.ChallengeRepo.GetRequirements(ctx, challengeID)
	if err != nil {
		return fmt.Errorf("ChallengeUseCase - submitRecordSolve - GetRequirements: %w", err)
	}

	if uc.deps.SolveRepo == nil || len(requirements) == 0 {
		return nil
	}

	requirementIDs := make([]uuid.UUID, 0, len(requirements))
	for _, req := range requirements {
		requirementIDs = append(requirementIDs, req.ChallengeID)
	}

	solvedIDs, err := uc.deps.SolveRepo.GetSolvedChallengeIDsByTeam(ctx, teamID, requirementIDs)
	if err != nil {
		return fmt.Errorf("ChallengeUseCase - submitRecordSolve - GetSolvedChallengeIDsByTeam: %w", err)
	}

	solvedSet := make(map[uuid.UUID]struct{}, len(solvedIDs))
	for _, id := range solvedIDs {
		solvedSet[id] = struct{}{}
	}

	for _, req := range requirements {
		if _, ok := solvedSet[req.ChallengeID]; !ok {
			return apperr.ErrRequirementsNotMet
		}
	}

	return nil
}

// countAttempts returns the number of submissions by the team for the challenge.
// When window > 0 only submissions within the rolling time window are counted
// (rate-limiting semantics); otherwise the total lifetime count is returned.
func (uc *ChallengeUseCase) countAttempts(ctx context.Context, teamID, challengeID uuid.UUID, window time.Duration) (int64, error) {
	if window > 0 {
		windowStart := time.Now().Add(-window)

		return uc.deps.SubmissionRepo.CountSubmissionsByTeamAndChallengeInWindow(ctx, teamID, challengeID, windowStart)
	}

	return uc.deps.SubmissionRepo.CountSubmissionsByTeamAndChallenge(ctx, teamID, challengeID)
}

// submitLogIncorrectAndEnforceMaxAttempts records an incorrect submission and enforces the
// per-team, per-challenge attempt limit. When MaxAttempts is configured it wraps both
// operations in a serializable transaction protected by an advisory lock so that the
// count read and the submission insert are atomic: a concurrent submission cannot sneak in
// between the read and the write. If the count already equals or exceeds MaxAttempts the
// transaction returns ErrMaxAttemptsReached without inserting. When MaxAttempts is not
// configured the submission is written directly without a transaction.
func (uc *ChallengeUseCase) submitLogIncorrectAndEnforceMaxAttempts(sc *submitContext, challenge *domain.Challenge) error {
	sub := &domain.Submission{
		UserID:        sc.userID,
		TeamID:        &sc.teamID,
		ChallengeID:   sc.challengeID,
		SubmittedFlag: sc.flag,
		IsCorrect:     false,
		Type:          domain.SubmissionTypeIncorrect,
		IP:            sc.clientIP,
		CreatedAt:     time.Now(),
	}

	if challenge.MaxAttempts > 0 && uc.deps.SubmissionRepo != nil && uc.deps.TM != nil {
		err := uc.deps.TM.Run(sc.ctx, func(ctx context.Context) error {
			if err := uc.deps.SubmissionRepo.AcquireAdvisoryLockForSubmit(ctx, sc.teamID, sc.challengeID); err != nil {
				return fmt.Errorf("ChallengeUseCase - submitLogIncorrectAndEnforceMaxAttempts - AcquireAdvisoryLockForSubmit: %w", err)
			}

			count, err := uc.countAttempts(ctx, sc.teamID, sc.challengeID, challenge.MaxAttemptsWindow)
			if err != nil {
				return fmt.Errorf("ChallengeUseCase - submitLogIncorrectAndEnforceMaxAttempts - countAttempts: %w", err)
			}

			if count >= int64(challenge.MaxAttempts) {
				return apperr.ErrMaxAttemptsReached
			}

			return uc.deps.SubmissionRepo.Create(ctx, sub)
		})
		if err != nil {
			return fmt.Errorf("ChallengeUseCase - submitLogIncorrectAndEnforceMaxAttempts - TM.Run: %w", err)
		}

		return nil
	}

	if uc.deps.SubmissionRepo != nil {
		return uc.deps.SubmissionRepo.Create(sc.ctx, sub)
	}

	return nil
}

// submitValidateFlagFormat validates the submitted flag against the challenge-level or competition-level
// format regex (if configured). Uses safeMatchString with ReDoS protection.
func (uc *ChallengeUseCase) submitValidateFlagFormat(sc *submitContext, challenge *domain.Challenge) error {
	formatRegex := ""

	if challenge.FlagFormatRegex != nil && *challenge.FlagFormatRegex != "" {
		formatRegex = *challenge.FlagFormatRegex
	} else if sc.comp != nil && sc.comp.FlagRegex != nil && *sc.comp.FlagRegex != "" {
		formatRegex = *sc.comp.FlagRegex
	}

	if formatRegex == "" {
		return nil
	}

	compiled, err := uc.getCompiledRegex(formatRegex)
	if err != nil {
		return fmt.Errorf("ChallengeUseCase - SubmitFlag - CompileFormatRegex: %w", err)
	}

	matched, err := safeMatchString(sc.ctx, compiled, sc.flag, regexMatchTimeout)
	if err != nil {
		return fmt.Errorf("ChallengeUseCase - SubmitFlag - format regex match: %w", err)
	}

	if !matched {
		return apperr.ErrInvalidFlagFormat
	}

	return nil
}

// submitCheckFlag dispatches to submitCheckRegexFlag or submitCheckHashFlag based on challenge type.
func (uc *ChallengeUseCase) submitCheckFlag(sc *submitContext, challenge *domain.Challenge) (bool, error) {
	if challenge.IsRegex {
		return uc.submitCheckRegexFlag(sc, challenge)
	}

	return uc.submitCheckHashFlag(sc, challenge), nil
}

// submitCheckRegexFlag decrypts the AES-encrypted flag_regex, optionally prepends (?i) for
// case-insensitive matching, then evaluates the submission via safeMatchString with semaphore
// and timeout protection against ReDoS.
func (uc *ChallengeUseCase) submitCheckRegexFlag(sc *submitContext, challenge *domain.Challenge) (bool, error) {
	if uc.deps.Crypto == nil {
		return false, fmt.Errorf("ChallengeUseCase - submitCheckRegexFlag - crypto not configured")
	}

	if challenge.FlagRegex == nil || *challenge.FlagRegex == "" {
		return false, fmt.Errorf("ChallengeUseCase - submitCheckRegexFlag - regex challenge has no flag_regex")
	}

	pattern, err := uc.deps.Crypto.Decrypt(*challenge.FlagRegex)
	if err != nil {
		return false, fmt.Errorf("ChallengeUseCase - submitCheckRegexFlag - crypto.Decrypt: %w", err)
	}

	if challenge.IsCaseInsensitive {
		pattern = "(?i)" + pattern
	}

	compiled, err := uc.getCompiledRegex(pattern)
	if err != nil {
		return false, fmt.Errorf("ChallengeUseCase - submitCheckRegexFlag - regexp.Compile: %w", err)
	}

	matched, err := safeMatchString(sc.ctx, compiled, sc.flag, regexMatchTimeout)
	if err != nil {
		return false, fmt.Errorf("ChallengeUseCase - submitCheckRegexFlag - match: %w", err)
	}

	return matched, nil
}

// submitCheckHashFlag compares the SHA-256 hex of the submitted flag against the stored hash
// using constant-time comparison to prevent timing-based side channels.
func (uc *ChallengeUseCase) submitCheckHashFlag(sc *submitContext, challenge *domain.Challenge) bool {
	userInput := sc.flag

	if challenge.IsCaseInsensitive {
		userInput = strings.ToLower(userInput)
	}

	hashStr := crypto.SHA256Hex(userInput)

	return subtle.ConstantTimeCompare([]byte(hashStr), []byte(challenge.FlagHash)) == 1
}

// submitRecordSolve atomically records a correct solve inside a SERIALIZABLE transaction
// The strategy inside the transaction is
//  1. Re-read the competition state (GetForUpdate) to detect freeze and re-check the
//     submission window; returns ErrSubmissionNotAllowed if the window has just closed
//  2. Lock and re-validate user (still in the team, not banned) and team (not banned,
//     mode/min-size constraints) rows with SELECT … FOR UPDATE to prevent concurrent
//     membership changes from producing an inconsistent solve record
//  3. Lock the challenge row (GetByIDForUpdate) and re-check hidden/locked state and
//     prerequisites - all inside the same transaction for idempotency
//  4. If MaxAttempts is set, acquires a second advisory lock keyed on (teamID, challengeID)
//     to prevent two concurrent correct submissions from both slipping under the cap
//     if the cap is exceeded and a solve already exists the call is treated as idempotent
//     (alreadySolved = true)
//  5. Inserts the correct submission record, then delegates to deps.SolveRecord
//     which handles idempotency (returns alreadySolved), dynamic score decay, and
//     first-blood detection
//
// Returns the locked challenge, the new solve count, an alreadySolved flag, a wasFrozen
// flag (used by the caller to decide which cache keys to invalidate), and any error.
func (uc *ChallengeUseCase) submitRecordSolve(sc *submitContext, _ *domain.Challenge) (*domain.Challenge, int, bool, bool, error) {
	var (
		solvedChallenge *domain.Challenge
		solveCount      int
		alreadySolved   bool
		wasFrozen       bool
	)

	err := uc.deps.TM.Run(sc.ctx, func(ctx context.Context) error {
		var (
			comp      *domain.Competition
			freshUser *domain.User
		)

		if uc.deps.CompRepo != nil {
			c, err := uc.deps.CompRepo.GetForUpdate(ctx)
			if err != nil && !errors.Is(err, apperr.ErrCompetitionNotFound) {
				return fmt.Errorf("ChallengeUseCase - submitRecordSolve - CompRepo.GetForUpdate: %w", err)
			}

			comp = c
			if comp != nil {
				wasFrozen = comp.IsFreezeActive()
			}
		}

		if comp != nil && !comp.IsSubmissionAllowed() {
			return apperr.ErrSubmissionNotAllowed
		}

		if uc.deps.UserRepo != nil {
			if err := uc.deps.UserRepo.Lock(ctx, sc.userID); err != nil {
				return fmt.Errorf("ChallengeUseCase - submitRecordSolve - UserRepo.Lock: %w", err)
			}

			freshUser, err := uc.deps.UserRepo.GetByID(ctx, sc.userID)
			if err != nil {
				return fmt.Errorf("ChallengeUseCase - submitRecordSolve - UserRepo.GetByID: %w", err)
			}

			if freshUser.TeamID == nil || *freshUser.TeamID != sc.teamID {
				return apperr.ErrTeamMemberNotFound
			}

			if freshUser.IsBanned {
				return apperr.ErrUserBanned
			}

			if freshUser.WasInBannedTeam && freshUser.Role != domain.RoleAdmin {
				return apperr.ErrUserWasInBannedTeam
			}
		}

		if uc.deps.TeamRepo != nil {
			if err := uc.deps.TeamRepo.Lock(ctx, sc.teamID); err != nil {
				return fmt.Errorf("ChallengeUseCase - submitRecordSolve - TeamRepo.Lock: %w", err)
			}

			freshTeam, err := uc.deps.TeamRepo.GetByID(ctx, sc.teamID)
			if err != nil {
				return fmt.Errorf("ChallengeUseCase - submitRecordSolve - TeamRepo.GetByID: %w", err)
			}

			if err := guard.ValidateSubmissionEligibility(ctx, freshUser, freshTeam, comp, uc.deps.TeamRepo); err != nil {
				return err
			}
		}

		freshChallenge, err := uc.deps.ChallengeRepo.GetByIDForUpdate(ctx, sc.challengeID)
		if err != nil {
			return fmt.Errorf("ChallengeUseCase - submitRecordSolve - ChallengeRepo.GetByIDForUpdate: %w", err)
		}

		if err := guard.EnsureChallengeVisible(freshChallenge); err != nil {
			return err
		}

		if freshChallenge.State == domain.ChallengeStateLocked {
			return apperr.ErrChallengeLocked
		}

		if err := uc.submitCheckRequirementsInTx(ctx, sc.challengeID, sc.teamID); err != nil {
			return err
		}

		if freshChallenge.MaxAttempts > 0 && uc.deps.SubmissionRepo != nil {
			if err := uc.deps.SubmissionRepo.AcquireAdvisoryLockForSubmit(ctx, sc.teamID, sc.challengeID); err != nil {
				return fmt.Errorf("ChallengeUseCase - submitRecordSolve - AcquireAdvisoryLockForSubmit: %w", err)
			}

			count, err := uc.countAttempts(ctx, sc.teamID, sc.challengeID, freshChallenge.MaxAttemptsWindow)
			if err != nil {
				return fmt.Errorf("ChallengeUseCase - submitRecordSolve - countAttempts: %w", err)
			}

			if count >= int64(freshChallenge.MaxAttempts) {
				if uc.deps.SolveRepo != nil {
					_, solveErr := uc.deps.SolveRepo.GetByTeamAndChallenge(ctx, sc.teamID, sc.challengeID)
					if solveErr == nil {
						solvedChallenge = freshChallenge
						alreadySolved = true

						return nil
					}
				}

				return apperr.ErrMaxAttemptsReached
			}
		}

		correctSub := &domain.Submission{
			UserID:        sc.userID,
			TeamID:        &sc.teamID,
			ChallengeID:   sc.challengeID,
			SubmittedFlag: sc.flag,
			IsCorrect:     true,
			Type:          domain.SubmissionTypeCorrect,
			IP:            sc.clientIP,
			CreatedAt:     time.Now(),
		}

		if uc.deps.SubmissionRepo != nil {
			err := uc.deps.SubmissionRepo.Create(ctx, correctSub)
			if err != nil {
				return fmt.Errorf("ChallengeUseCase - submitRecordSolve - SubmissionRepo.Create: %w", err)
			}
		}

		solvedChallenge = freshChallenge
		solve := &domain.Solve{UserID: sc.userID, TeamID: sc.teamID, ChallengeID: sc.challengeID}

		if uc.deps.SolveRecord == nil {
			return fmt.Errorf("ChallengeUseCase - submitRecordSolve: SolveRecord not configured")
		}

		solveCount, err = uc.deps.SolveRecord(ctx, solve, freshChallenge, uc.deps.ChallengeRepo, uc.deps.SolveRepo, scoring.GetDecayFn(ctx, uc.deps.CompParamUC))
		if err != nil {
			return fmt.Errorf("ChallengeUseCase - submitRecordSolve - RecordSolveInTx: %w", err)
		}

		return nil
	})
	if err != nil {
		var ve *apperr.ValidationError
		if errors.As(err, &ve) || errors.Is(err, apperr.ErrMaxAttemptsReached) || errors.Is(err, apperr.ErrAlreadySolved) {
			return nil, 0, false, false, err
		}

		return nil, 0, false, false, fmt.Errorf("ChallengeUseCase - submitRecordSolve - TM.Run: %w", err)
	}

	return solvedChallenge, solveCount, alreadySolved, wasFrozen, nil
}

// submitRecordSolveUpdatePointsIfDecay applies dynamic score decay after recording a solve.
// When the new score differs from the current score, updates the challenge points in the DB.
func (uc *ChallengeUseCase) submitRecordSolveUpdatePointsIfDecay(ctx context.Context, challengeID uuid.UUID, solvedChallenge *domain.Challenge, solveCount int) error {
	_, err := scoring.ApplySolveScore(ctx,
		solvedChallenge.InitialValue, solvedChallenge.MinValue, solvedChallenge.Decay, solvedChallenge.Points, solveCount,
		scoring.GetDecayFn(ctx, uc.deps.CompParamUC),
		func(ctx context.Context, pts int) error {
			err := uc.deps.ChallengeRepo.UpdatePoints(ctx, challengeID, pts)
			if err != nil {
				return fmt.Errorf("ChallengeUseCase - submitRecordSolveUpdatePointsIfDecay - ChallengeRepo.UpdatePoints: %w", err)
			}

			solvedChallenge.Points = pts

			return nil
		},
	)
	if err != nil {
		return fmt.Errorf("ChallengeUseCase - submitRecordSolveUpdatePointsIfDecay - ApplySolveScore: %w", err)
	}

	return nil
}

// submitInvalidateCache invalidates scoreboard and challenge caches after a solve.
// Delegates to submitInvalidateCacheWithFrozenStatus after checking current freeze state.
func (uc *ChallengeUseCase) submitInvalidateCache(ctx context.Context, teamID uuid.UUID) {
	comp := computil.Fresh(ctx, uc.deps.CompRepo, uc.deps.CompUC)
	wasFrozen := comp != nil && comp.IsFreezeActive()
	uc.submitInvalidateCacheWithFrozenStatus(ctx, teamID, wasFrozen)
}

// submitInvalidateCacheWithFrozenStatus invalidates scoreboard and challenge list caches
// with awareness of freeze state. When frozen, only live scoreboard is invalidated
// (frozen snapshot is preserved). Challenge list cache is always invalidated.
func (uc *ChallengeUseCase) submitInvalidateCacheWithFrozenStatus(ctx context.Context, teamID uuid.UUID, wasFrozen bool) {
	cache.InvalidateWithFreezeAwareness(ctx, uc.deps.ScoreboardCache, teamID, wasFrozen)
	uc.InvalidateChallengeListCacheForTeam(ctx, teamID)
}

// submitNotifySolve broadcasts a solve event (and optionally a first-blood event) to WebSocket/SSE clients.
// wasFrozen is forwarded from submitRecordSolve to skip the redundant CompRepo hit.
func (uc *ChallengeUseCase) submitNotifySolve(sc *submitContext, challenge *domain.Challenge, isFirstBlood, wasFrozen bool) {
	if uc.deps.Broadcaster == nil || challenge == nil || wasFrozen {
		return
	}

	uc.deps.Broadcaster.NotifySolve(sc.teamID, challenge.Title, challenge.Points, isFirstBlood)
}
