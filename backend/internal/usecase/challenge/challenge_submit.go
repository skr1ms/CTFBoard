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

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/competition"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/crypto"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/scoring"
)

const (
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

func safeMatchString(ctx context.Context, re *regexp.Regexp, input string, timeout time.Duration) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	err := regexSem.Acquire(ctx, 1)
	if err != nil {
		return false, fmt.Errorf("regex semaphore: %w", err)
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
		return false, fmt.Errorf("regex match timed out: %w", ctx.Err())
	}
}

//nolint:funlen // submission flow: validation, lock, solve check, create, broadcast
func (uc *ChallengeUseCase) SubmitFlag(ctx context.Context, challengeID uuid.UUID, flag string, userID uuid.UUID, teamID *uuid.UUID, clientIP string) (bool, error) {
	if teamID == nil {
		return false, httperr.ErrUserMustBeInTeam
	}

	comp, err := uc.submitCheckCompetitionTime(ctx)
	if err != nil {
		return false, fmt.Errorf("ChallengeUseCase - SubmitFlag - submitCheckCompetitionTime: %w", err)
	}

	sc := &submitContext{ctx: ctx, challengeID: challengeID, flag: strings.TrimSpace(flag), userID: userID, teamID: *teamID, clientIP: clientIP, comp: comp}
	if sc.flag == "" {
		return false, httperr.ErrInvalidFlagFormat
	}

	if len(sc.flag) > MaxFlagLength {
		return false, httperr.ErrInvalidFlagFormat
	}

	if uc.deps.UserRepo != nil {
		user, err := uc.deps.UserRepo.GetByID(ctx, userID)
		if err != nil {
			return false, fmt.Errorf("ChallengeUseCase - SubmitFlag - UserRepo.GetByID: %w", err)
		}

		if user.IsBanned {
			return false, httperr.ErrUserBanned
		}
	}

	if uc.deps.TeamRepo != nil {
		team, err := uc.deps.TeamRepo.GetByID(ctx, sc.teamID)
		if err != nil {
			return false, fmt.Errorf("ChallengeUseCase - SubmitFlag - TeamRepo.GetByID: %w", err)
		}

		sc.team = team
		if team.IsBanned {
			return false, httperr.ErrTeamBanned
		}

		if comp != nil {
			if comp.Mode == domain.ModeTeamsOnly && team.IsSolo {
				return false, httperr.ErrTeamModeRequired
			}

			if comp.Mode == domain.ModeSoloOnly && !team.IsSolo {
				return false, httperr.ErrSoloModeRequired
			}

			if comp.MinTeamSize > 0 && !team.IsSolo {
				count, err := uc.deps.TeamRepo.CountTeamMembers(ctx, sc.teamID)
				if err != nil {
					return false, fmt.Errorf("ChallengeUseCase - SubmitFlag - TeamRepo.CountTeamMembers: %w", err)
				}

				if count < comp.MinTeamSize {
					return false, httperr.ErrTeamBelowMinSize
				}
			}
		}
	}

	challenge, err := uc.submitGetChallenge(sc)
	if err != nil {
		if errors.Is(err, httperr.ErrChallengeNotFound) {
			return false, httperr.ErrChallengeNotFound
		}

		return false, fmt.Errorf("ChallengeUseCase - SubmitFlag - submitGetChallenge: %w", err)
	}

	if challenge.State == domain.ChallengeStateLocked {
		return false, httperr.ErrChallengeLocked
	}

	met, err := requirementsMet(ctx, sc.challengeID, sc.teamID, uc.deps.ChallengeRepo, uc.deps.SolveRepo)
	if err != nil {
		return false, fmt.Errorf("ChallengeUseCase - SubmitFlag - requirementsMet: %w", err)
	}

	if !met {
		return false, httperr.ErrRequirementsNotMet
	}

	if challenge.MaxAttempts > 0 && uc.deps.SubmissionRepo != nil {
		count, err := uc.deps.SubmissionRepo.CountSubmissionsByTeamAndChallenge(sc.ctx, sc.teamID, sc.challengeID)
		if err != nil {
			return false, fmt.Errorf("ChallengeUseCase - SubmitFlag - CountSubmissionsByTeamAndChallenge: %w", err)
		}

		if count >= int64(challenge.MaxAttempts) {
			return false, httperr.ErrMaxAttemptsReached
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
			if errors.Is(err, httperr.ErrMaxAttemptsReached) {
				return false, httperr.ErrMaxAttemptsReached
			}

			return false, fmt.Errorf("ChallengeUseCase - SubmitFlag - submitLogIncorrectAndEnforceMaxAttempts: %w", err)
		}

		return false, nil
	}

	solvedChallenge, solveCount, alreadySolved, wasFrozen, err := uc.submitRecordSolve(sc, challenge)
	if err != nil {
		if errors.Is(err, httperr.ErrRequirementsNotMet) {
			return false, httperr.ErrRequirementsNotMet
		}

		if errors.Is(err, httperr.ErrMaxAttemptsReached) {
			return false, httperr.ErrMaxAttemptsReached
		}

		return false, fmt.Errorf("ChallengeUseCase - SubmitFlag - submitRecordSolve: %w", err)
	}

	if alreadySolved {
		return true, nil
	}

	uc.submitInvalidateCacheWithFrozenStatus(sc.ctx, sc.teamID, wasFrozen)
	uc.submitNotifySolve(sc, solvedChallenge, solveCount == 1)

	return true, nil
}

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
		return nil, httperr.ErrSubmissionNotAllowed
	}

	return comp, nil
}

func (uc *ChallengeUseCase) submitGetChallenge(sc *submitContext) (*domain.Challenge, error) {
	key := sc.challengeID.String()

	v, err, _ := uc.challengeFetchSf.Do(key, func() (any, error) {
		return uc.deps.ChallengeRepo.GetByID(context.WithoutCancel(sc.ctx), sc.challengeID)
	})
	if err != nil {
		if errors.Is(err, httperr.ErrChallengeNotFound) {
			return nil, httperr.ErrChallengeNotFound
		}

		return nil, fmt.Errorf("ChallengeUseCase - SubmitFlag - ChallengeRepo.GetByID: %w", err)
	}

	challenge, ok := v.(*domain.Challenge)
	if !ok {
		return nil, fmt.Errorf("ChallengeUseCase - SubmitFlag - ChallengeRepo.GetByID: unexpected type")
	}

	if challenge.State == domain.ChallengeStateHidden {
		return nil, httperr.ErrChallengeNotFound
	}

	return challenge, nil
}

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
			return httperr.ErrRequirementsNotMet
		}
	}

	return nil
}

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
				return err
			}

			count, err := uc.deps.SubmissionRepo.CountSubmissionsByTeamAndChallenge(ctx, sc.teamID, sc.challengeID)
			if err != nil {
				return fmt.Errorf("CountSubmissionsByTeamAndChallenge: %w", err)
			}

			if count >= int64(challenge.MaxAttempts) {
				return httperr.ErrMaxAttemptsReached
			}

			return uc.deps.SubmissionRepo.Create(ctx, sub)
		})
		if err != nil {
			return err
		}

		return nil
	}

	if uc.deps.SubmissionRepo != nil {
		return uc.deps.SubmissionRepo.Create(sc.ctx, sub)
	}

	return nil
}

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
		return httperr.ErrInvalidFlagFormat
	}

	return nil
}

func (uc *ChallengeUseCase) submitCheckFlag(sc *submitContext, challenge *domain.Challenge) (bool, error) {
	if challenge.IsRegex {
		return uc.submitCheckRegexFlag(sc, challenge)
	}

	return uc.submitCheckHashFlag(sc, challenge), nil
}

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

func (uc *ChallengeUseCase) submitCheckHashFlag(sc *submitContext, challenge *domain.Challenge) bool {
	userInput := sc.flag

	if challenge.IsCaseInsensitive {
		userInput = strings.ToLower(userInput)
	}

	hashStr := crypto.SHA256Hex(userInput)

	return subtle.ConstantTimeCompare([]byte(hashStr), []byte(challenge.FlagHash)) == 1
}

func (uc *ChallengeUseCase) submitRecordSolve(sc *submitContext, _ *domain.Challenge) (*domain.Challenge, int, bool, bool, error) {
	var (
		solvedChallenge *domain.Challenge
		solveCount      int
		alreadySolved   bool
		wasFrozen       bool
	)

	err := uc.deps.TM.Run(sc.ctx, func(ctx context.Context) error {
		var comp *domain.Competition

		if uc.deps.CompRepo != nil {
			c, err := uc.deps.CompRepo.GetForUpdate(ctx)
			if err != nil && !errors.Is(err, httperr.ErrCompetitionNotFound) {
				return fmt.Errorf("ChallengeUseCase - submitRecordSolve - CompRepo.GetForUpdate: %w", err)
			}

			comp = c
			if comp != nil {
				wasFrozen = comp.IsFreezeActive()
			}
		}

		if comp != nil && !comp.IsSubmissionAllowed() {
			return httperr.ErrSubmissionNotAllowed
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
				return httperr.ErrTeamMemberNotFound
			}

			if freshUser.IsBanned {
				return httperr.ErrUserBanned
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

			if freshTeam.IsBanned {
				return httperr.ErrTeamBanned
			}

			if comp != nil {
				if comp.Mode == domain.ModeTeamsOnly && freshTeam.IsSolo {
					return httperr.ErrTeamModeRequired
				}

				if comp.Mode == domain.ModeSoloOnly && !freshTeam.IsSolo {
					return httperr.ErrSoloModeRequired
				}

				if comp.MinTeamSize > 0 && !freshTeam.IsSolo {
					count, err := uc.deps.TeamRepo.CountTeamMembers(ctx, sc.teamID)
					if err != nil {
						return fmt.Errorf("ChallengeUseCase - submitRecordSolve - TeamRepo.CountTeamMembers: %w", err)
					}

					if count < comp.MinTeamSize {
						return httperr.ErrTeamBelowMinSize
					}
				}
			}
		}

		freshChallenge, err := uc.deps.ChallengeRepo.GetByIDForUpdate(ctx, sc.challengeID)
		if err != nil {
			return fmt.Errorf("ChallengeUseCase - submitRecordSolve - ChallengeRepo.GetByIDForUpdate: %w", err)
		}

		if freshChallenge.State == domain.ChallengeStateHidden {
			return httperr.ErrChallengeNotFound
		}

		if freshChallenge.State == domain.ChallengeStateLocked {
			return httperr.ErrChallengeLocked
		}

		if err := uc.submitCheckRequirementsInTx(ctx, sc.challengeID, sc.teamID); err != nil {
			return err
		}

		if freshChallenge.MaxAttempts > 0 && uc.deps.SubmissionRepo != nil {
			if err := uc.deps.SubmissionRepo.AcquireAdvisoryLockForSubmit(ctx, sc.teamID, sc.challengeID); err != nil {
				return fmt.Errorf("AcquireAdvisoryLockForSubmit: %w", err)
			}

			count, err := uc.deps.SubmissionRepo.CountSubmissionsByTeamAndChallenge(ctx, sc.teamID, sc.challengeID)
			if err != nil {
				return fmt.Errorf("CountSubmissionsByTeamAndChallenge: %w", err)
			}

			if count >= int64(freshChallenge.MaxAttempts) {
				if uc.deps.SolveRepo != nil {
					_, solveErr := uc.deps.SolveRepo.GetByTeamAndChallenge(ctx, sc.teamID, sc.challengeID)
					if solveErr == nil {
						solvedChallenge = freshChallenge
						alreadySolved = true
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
								return fmt.Errorf("SubmissionRepo.Create: %w", err)
							}
						}

						return nil
					}
				}

				return httperr.ErrMaxAttemptsReached
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
				return fmt.Errorf("SubmissionRepo.Create: %w", err)
			}
		}

		solvedChallenge = freshChallenge
		solve := &domain.Solve{UserID: sc.userID, TeamID: sc.teamID, ChallengeID: sc.challengeID}

		solveCount, err = competition.RecordSolveInTx(ctx, solve, freshChallenge, uc.deps.ChallengeRepo, uc.deps.SolveRepo)
		if err != nil {
			return fmt.Errorf("ChallengeUseCase - submitRecordSolve - RecordSolveInTx: %w", err)
		}

		return nil
	})
	if err != nil {
		var httpErr *httperr.HTTPError
		if errors.As(err, &httpErr) {
			return nil, 0, false, false, err
		}

		return nil, 0, false, false, fmt.Errorf("ChallengeUseCase - submitRecordSolve - TM.Run: %w", err)
	}

	return solvedChallenge, solveCount, alreadySolved, wasFrozen, nil
}

func (uc *ChallengeUseCase) submitRecordSolveUpdatePointsIfDecay(ctx context.Context, challengeID uuid.UUID, solvedChallenge *domain.Challenge, solveCount int) error {
	_, err := scoring.ApplySolveScore(ctx,
		solvedChallenge.InitialValue, solvedChallenge.MinValue, solvedChallenge.Decay, solvedChallenge.Points, solveCount,
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

func (uc *ChallengeUseCase) submitInvalidateCache(ctx context.Context, teamID uuid.UUID) {
	comp := uc.submitGetFreshCompetition(ctx)
	wasFrozen := comp != nil && comp.IsFreezeActive()
	uc.submitInvalidateCacheWithFrozenStatus(ctx, teamID, wasFrozen)
}

func (uc *ChallengeUseCase) submitInvalidateCacheWithFrozenStatus(ctx context.Context, teamID uuid.UUID, wasFrozen bool) {
	if uc.deps.ScoreboardCache != nil {
		if wasFrozen {
			uc.deps.ScoreboardCache.InvalidateLiveOnly(ctx, teamID)
			uc.InvalidateChallengeListCacheForTeam(ctx, teamID)

			return
		}

		uc.deps.ScoreboardCache.InvalidateForTeam(ctx, teamID)
	}

	uc.InvalidateChallengeListCacheForTeam(ctx, teamID)
}

func (uc *ChallengeUseCase) submitGetFreshCompetition(ctx context.Context) *domain.Competition {
	if uc.deps.CompRepo != nil {
		comp, err := uc.deps.CompRepo.Get(ctx)
		if err == nil {
			return comp
		}
	}

	if uc.deps.CompUC != nil {
		comp, err := uc.deps.CompUC.Get(ctx)
		if err == nil {
			return comp
		}
	}

	return nil
}

func (uc *ChallengeUseCase) submitNotifySolve(sc *submitContext, challenge *domain.Challenge, isFirstBlood bool) {
	if uc.deps.Broadcaster == nil || challenge == nil {
		return
	}

	comp := uc.submitGetFreshCompetition(sc.ctx)
	if comp != nil && comp.IsFreezeActive() {
		return
	}

	uc.deps.Broadcaster.NotifySolve(sc.teamID, challenge.Title, challenge.Points, isFirstBlood)
}
