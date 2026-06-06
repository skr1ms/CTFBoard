package challenge

import (
	"context"
	"crypto/subtle"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/crypto"
)

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
// ReDoS. First it acquires a slot from the usecase-owned weighted semaphore (capacity
// maxConcurrentRegex) so that the number of simultaneously running regex goroutines is
// bounded. Then it launches the match in a separate goroutine and uses a select to honour
// the caller's context deadline or the explicit timeout, whichever fires first. The result
// channel is buffered (cap 1) so the goroutine can always send and call
// semaphore.Release even when the caller has already timed out and stopped receiving.
func (uc *ChallengeUseCase) safeMatchString(ctx context.Context, re *regexp.Regexp, input string, timeout time.Duration) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	err := uc.regexSem.Acquire(ctx, 1)
	if err != nil {
		return false, fmt.Errorf("ChallengeUseCase - safeMatchString - acquire semaphore: %w", err)
	}

	ch := make(chan bool, 1)

	go func() {
		defer uc.regexSem.Release(1)

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

	matched, err := uc.safeMatchString(sc.ctx, compiled, sc.flag, regexMatchTimeout)
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

	matched, err := uc.safeMatchString(sc.ctx, compiled, sc.flag, regexMatchTimeout)
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
