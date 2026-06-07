package challenge

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-logkit"
	"golang.org/x/sync/errgroup"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/crypto"
)

func validateDynamicScoringRange(initialValue, minValue int) error {
	if initialValue < 0 {
		return apperr.NewValidationErrorf("dynamic scoring initial value must be non-negative")
	}

	if minValue < 0 {
		return apperr.NewValidationErrorf("dynamic scoring min value must be non-negative")
	}

	if initialValue > 0 && minValue > 0 && initialValue < minValue {
		return apperr.ErrInvalidScoringRange
	}

	return nil
}

const (
	maxFlagFormatRegexLen             = 1024
	deleteStorageFilesConcurrency     = 4
	deleteStorageFilesDetachedTimeout = 30 * time.Second
)

func validateFlagFormatRegex(flagFormatRegex *string) error {
	if flagFormatRegex == nil || *flagFormatRegex == "" {
		return nil
	}

	if len(*flagFormatRegex) > maxFlagFormatRegexLen {
		return apperr.ErrInvalidFlagFormat
	}

	if _, err := regexp.Compile(*flagFormatRegex); err != nil {
		return apperr.ErrInvalidFlagFormat
	}

	return nil
}

func (uc *ChallengeUseCase) ensureNextChallengeExists(ctx context.Context, nextID *uuid.UUID) error {
	if nextID == nil {
		return nil
	}

	if _, err := uc.deps.ChallengeRepo.GetByID(ctx, *nextID); err != nil {
		if errors.Is(err, apperr.ErrChallengeNotFound) {
			return apperr.NewValidationErrorf("next_id references unknown challenge")
		}

		return fmt.Errorf("ChallengeUseCase - ensureNextChallengeExists - ChallengeRepo.GetByID: %w", err)
	}

	return nil
}

func (uc *ChallengeUseCase) validateNextChallengeReference(ctx context.Context, currentID uuid.UUID, params usecase.ChallengeUpdateParams) error {
	if !params.NextChallengeSet {
		return nil
	}

	if params.NextChallengeID == nil {
		return nil
	}

	if *params.NextChallengeID == currentID {
		return apperr.NewValidationErrorf("next_id cannot reference the same challenge")
	}

	return uc.ensureNextChallengeExists(ctx, params.NextChallengeID)
}

// Create creates a new challenge, hashing or AES-encrypting the flag, persisting it with tags,
// and invalidating the challenge list cache.
func (uc *ChallengeUseCase) Create(ctx context.Context, params usecase.ChallengeCreateParams) (*domain.Challenge, error) {
	if err := validateDynamicScoringRange(params.InitialValue, params.MinValue); err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - Create - validateDynamicScoringRange: %w", err)
	}

	if err := uc.ensureNextChallengeExists(ctx, params.NextChallengeID); err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - Create - ensureNextChallengeExists: %w", err)
	}

	flagHash, flagRegex, err := uc.challengeCreateComputeFlagHash(params.Flag, params.IsRegex, params.IsCaseInsensitive)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - Create - challengeCreateComputeFlagHash: %w", err)
	}

	var flagRegexPtr *string

	if flagRegex != "" {
		flagRegexPtr = &flagRegex
	}

	challenge := &domain.Challenge{
		Title:             params.Title,
		Description:       params.Description,
		Category:          params.Category,
		Points:            params.Points,
		InitialValue:      params.InitialValue,
		MinValue:          params.MinValue,
		Decay:             params.Decay,
		SolveCount:        0,
		FlagHash:          flagHash,
		Attribution:       params.Attribution,
		ConnectionInfo:    params.ConnectionInfo,
		MaxAttempts:       params.MaxAttempts,
		MaxAttemptsWindow: params.MaxAttemptsWindow,
		Position:          params.Position,
		NextChallengeID:   params.NextChallengeID,
		State:             domain.ChallengeStateOrDefault(params.State),
		IsRegex:           params.IsRegex,
		IsCaseInsensitive: params.IsCaseInsensitive,
		FlagRegex:         flagRegexPtr,
		FlagFormatRegex:   params.FlagFormatRegex,
	}
	if err := validateFlagFormatRegex(params.FlagFormatRegex); err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - Create - validateFlagFormatRegex: %w", err)
	}

	if err := uc.challengeCreatePersist(ctx, challenge, params.TagIDs); err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - Create - challengeCreatePersist: %w", err)
	}

	uc.InvalidateChallengeListCache(ctx)
	uc.invalidateStatisticsCache(ctx, "Create")

	return challenge, nil
}

// challengeCreateComputeFlagHash derives the stored flag representation from the plain-text flag.
// For regex challenges: AES-encrypts the pattern and sets FlagHash to the sentinel value.
// For plain challenges: SHA-256 hashes the (optionally lowercased) flag.
func (uc *ChallengeUseCase) challengeCreateComputeFlagHash(flag string, isRegex, isCaseInsensitive bool) (flagHash, flagRegex string, err error) {
	if isRegex {
		if uc.deps.Crypto == nil {
			return "", "", fmt.Errorf("ChallengeUseCase - Create - crypto.ErrServiceNotConfigured: %w", crypto.ErrServiceNotConfigured)
		}

		encrypted, err := uc.deps.Crypto.Encrypt(flag)
		if err != nil {
			return "", "", fmt.Errorf("ChallengeUseCase - Create - crypto.Encrypt: %w", err)
		}

		return domain.FlagHashRegexSentinel, encrypted, nil
	}

	userInput := crypto.NormalizeFlagInput(strings.TrimSpace(flag))

	if isCaseInsensitive {
		userInput = strings.ToLower(userInput)
	}

	return crypto.SHA256Hex(userInput), "", nil
}

func (uc *ChallengeUseCase) challengeCreatePersist(ctx context.Context, challenge *domain.Challenge, tagIDs []uuid.UUID) error {
	return uc.challengeCreatePersistTx(ctx, challenge, tagIDs)
}

// challengeCreatePersistTx persists a new challenge inside a transaction: generates a
// UUID, creates the row, and optionally attaches tags via SetTags. Requirements can be
// added after the challenge exists because SetRequirements is a separate call by the
// admin handler.
func (uc *ChallengeUseCase) challengeCreatePersistTx(ctx context.Context, challenge *domain.Challenge, tagIDs []uuid.UUID) error {
	return uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		challenge.ID = uuid.New()

		err := uc.deps.ChallengeRepo.Create(ctx, challenge)
		if err != nil {
			return fmt.Errorf("ChallengeUseCase - Create - ChallengeRepo.Create: %w", err)
		}

		if len(tagIDs) > 0 {
			err := uc.deps.ChallengeRepo.SetTags(ctx, challenge.ID, tagIDs)
			if err != nil {
				return fmt.Errorf("ChallengeUseCase - Create - ChallengeRepo.SetTags: %w", err)
			}
		}

		return nil
	})
}

func (uc *ChallengeUseCase) Update(ctx context.Context, ID uuid.UUID, params usecase.ChallengeUpdateParams) (*domain.Challenge, error) {
	if err := validateFlagFormatRegex(params.FlagFormatRegex); err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - Update - validateFlagFormatRegex: %w", err)
	}

	challenge, err := uc.challengeUpdatePersist(ctx, ID, params)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - Update - challengeUpdatePersist: %w", err)
	}

	if uc.deps.ScoreboardCache != nil {
		uc.deps.ScoreboardCache.InvalidateAll(ctx)
	}

	uc.InvalidateChallengeListCache(ctx)
	uc.invalidateStatisticsCache(ctx, "Update")

	return challenge, nil
}

// challengeUpdatePersist applies an update to a challenge inside a single transaction.
// It reads the current row with GetByID, computes the effective initialValue/minValue
// after merging nil-able overrides, and validates the dynamic scoring range. Flag handling
// covers three cases: (a) no flag change - checks for a mode switch (plain↔regex) and
// returns an error if a new flag value was not provided; (b) switching to regex - encrypts
// the new pattern and stores the sentinel hash; (c) switching to plain - hashes the new
// value (lowercased if case-insensitive) and clears the regex column. Tags are replaced
// atomically via SetTags in the same transaction.
func (uc *ChallengeUseCase) challengeUpdatePersist(ctx context.Context, ID uuid.UUID, params usecase.ChallengeUpdateParams) (*domain.Challenge, error) {
	var challenge *domain.Challenge

	err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		var errTx error

		challenge, errTx = uc.deps.ChallengeRepo.GetByID(ctx, ID)
		if errTx != nil {
			return fmt.Errorf("ChallengeUseCase - Update - ChallengeRepo.GetByID: %w", errTx)
		}

		effectiveIV, effectiveMV := challenge.InitialValue, challenge.MinValue

		if params.InitialValue != nil {
			effectiveIV = *params.InitialValue
		}

		if params.MinValue != nil {
			effectiveMV = *params.MinValue
		}

		if errTx = validateDynamicScoringRange(effectiveIV, effectiveMV); errTx != nil {
			return fmt.Errorf("ChallengeUseCase - Update - validateDynamicScoringRange: %w", errTx)
		}

		if errTx = uc.validateNextChallengeReference(ctx, ID, params); errTx != nil {
			return fmt.Errorf("ChallengeUseCase - Update - validateNextChallengeReference: %w", errTx)
		}

		uc.challengeUpdateApplyBasic(challenge, params)
		applyRegex, applyCaseInsensitive := challenge.IsRegex, challenge.IsCaseInsensitive

		if params.IsRegex != nil {
			applyRegex = *params.IsRegex
		}

		if params.IsCaseInsensitive != nil {
			applyCaseInsensitive = *params.IsCaseInsensitive
		}

		if errTx = uc.challengeUpdateApplyFlag(challenge, params.Flag, applyRegex, applyCaseInsensitive); errTx != nil {
			return fmt.Errorf("ChallengeUseCase - Update - challengeUpdateApplyFlag: %w", errTx)
		}

		if errTx = uc.deps.ChallengeRepo.Update(ctx, challenge); errTx != nil {
			return fmt.Errorf("ChallengeUseCase - Update - ChallengeRepo.Update: %w", errTx)
		}

		if errTx = uc.deps.ChallengeRepo.SetTags(ctx, ID, params.TagIDs); errTx != nil {
			return fmt.Errorf("ChallengeUseCase - Update - ChallengeRepo.SetTags: %w", errTx)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - challengeUpdatePersist - TM.Run: %w", err)
	}

	return challenge, nil
}

// challengeUpdateApplyBasic applies all non-flag scalar fields to the challenge struct.
// Pointer parameters are applied only when non-nil (partial-update semantics); string
// and int parameters that are always present overwrite unconditionally.
func (uc *ChallengeUseCase) challengeUpdateApplyBasic(c *domain.Challenge, params usecase.ChallengeUpdateParams) {
	c.Title = params.Title
	c.Description = params.Description
	c.Category = params.Category
	c.Points = params.Points

	if params.Attribution != nil {
		c.Attribution = *params.Attribution
	}

	if params.ConnectionInfo != nil {
		c.ConnectionInfo = *params.ConnectionInfo
	}

	if params.MaxAttempts != nil {
		c.MaxAttempts = *params.MaxAttempts
	}

	if params.MaxAttemptsWindow != nil {
		c.MaxAttemptsWindow = *params.MaxAttemptsWindow
	}

	if params.Position != nil {
		c.Position = *params.Position
	}

	if params.NextChallengeSet {
		c.NextChallengeID = params.NextChallengeID
	}

	if params.State != "" {
		c.State = domain.ChallengeStateOrDefault(params.State)
	}

	if params.InitialValue != nil {
		c.InitialValue = *params.InitialValue
	}

	if params.MinValue != nil {
		c.MinValue = *params.MinValue
	}

	if params.Decay != nil {
		c.Decay = *params.Decay
	}

	if params.IsRegex != nil {
		c.IsRegex = *params.IsRegex
	}

	if params.IsCaseInsensitive != nil {
		c.IsCaseInsensitive = *params.IsCaseInsensitive
	}

	c.FlagFormatRegex = params.FlagFormatRegex
}

// challengeUpdateApplyFlag applies a flag change during challenge update.
// Enforces that a new flag value is provided when switching between regex and hash modes.
// For regex: AES-encrypts the new pattern. For plain: SHA-256 hashes it.
func (uc *ChallengeUseCase) challengeUpdateApplyFlag(c *domain.Challenge, flag string, isRegex, isCaseInsensitive bool) error {
	if flag == "" {
		wasRegex := c.FlagHash == domain.FlagHashRegexSentinel
		if isRegex && !wasRegex {
			return apperr.ErrChallengeFlagRequiredWhenSwitchingMode
		}

		if !isRegex && wasRegex {
			return apperr.ErrChallengeFlagRequiredWhenSwitchingMode
		}

		return nil
	}

	if isRegex {
		if uc.deps.Crypto == nil {
			return fmt.Errorf("ChallengeUseCase - Update - crypto.ErrServiceNotConfigured: %w", crypto.ErrServiceNotConfigured)
		}

		encrypted, err := uc.deps.Crypto.Encrypt(flag)
		if err != nil {
			return fmt.Errorf("ChallengeUseCase - Update - crypto.Encrypt: %w", err)
		}

		c.FlagRegex = &encrypted
		c.FlagHash = domain.FlagHashRegexSentinel

		return nil
	}

	userInput := crypto.NormalizeFlagInput(strings.TrimSpace(flag))

	if isCaseInsensitive {
		userInput = strings.ToLower(userInput)
	}

	c.FlagHash = crypto.SHA256Hex(userInput)
	c.FlagRegex = nil

	return nil
}

// Delete removes a challenge and all of its associated data in a single transaction, then
// cleans up object-storage files after the commit. Inside the transaction it collects all
// file storage locations, deletes the challenge row (which cascades to solves, submissions,
// hints, unlocks, tags, requirements, and solution rows via foreign-key constraints), and
// writes an audit-log entry. Storage deletion happens outside the transaction via
// deleteStorageFiles so that a storage failure does not roll back the DB deletion; the
// call uses context.WithoutCancel so that an HTTP request cancellation does not abort the
// best-effort cleanup. Scoreboard and challenge-list caches are invalidated after the
// commit.
func (uc *ChallengeUseCase) Delete(ctx context.Context, ID, actorID uuid.UUID, clientIP string) error {
	var fileLocations []string

	err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		if _, err := uc.deps.ChallengeRepo.GetByID(ctx, ID); err != nil {
			return fmt.Errorf("ChallengeUseCase - Delete - ChallengeRepo.GetByID: %w", err)
		}

		fileLocations = uc.collectFileLocations(ctx, ID)

		err := uc.deps.ChallengeRepo.Delete(ctx, ID)
		if err != nil {
			return fmt.Errorf("ChallengeUseCase - Delete - ChallengeRepo.Delete: %w", err)
		}

		auditLog := &domain.AuditLog{
			UserID:     &actorID,
			Action:     domain.AuditActionDelete,
			EntityType: domain.AuditEntityChallenge,
			EntityID:   ID.String(),
			IP:         clientIP,
		}

		err = uc.deps.AuditLogRepo.Create(ctx, auditLog)
		if err != nil {
			return fmt.Errorf("ChallengeUseCase - Delete - AuditLogRepo.Create: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("ChallengeUseCase - Delete - TM.Run: %w", err)
	}

	cleanupCtx, cleanupCancel := storageCleanupContext(ctx)
	defer cleanupCancel()

	uc.deleteStorageFiles(cleanupCtx, fileLocations)

	if uc.deps.ScoreboardCache != nil {
		uc.deps.ScoreboardCache.InvalidateAll(ctx)
	}

	uc.InvalidateChallengeListCache(ctx)
	uc.invalidateStatisticsCache(ctx, "Delete")

	return nil
}

// collectFileLocations fetches all file rows for a challenge and extracts their storage
// location strings. Errors are logged and swallowed so that a missing file record does
// not prevent challenge deletion; the caller handles storage cleanup best-effort.
func (uc *ChallengeUseCase) collectFileLocations(ctx context.Context, challengeID uuid.UUID) []string {
	if uc.deps.FileRepo == nil || uc.deps.Storage == nil {
		return nil
	}

	files, err := uc.deps.FileRepo.GetAllByChallengeID(ctx, challengeID)
	if err != nil {
		uc.deps.Logger.WithError(err).Warn("ChallengeUseCase - collectFileLocations - GetAllByChallengeID")

		return nil
	}

	locations := make([]string, 0, len(files))
	for _, f := range files {
		locations = append(locations, f.Location)
	}

	return locations
}

// deleteStorageFiles removes files from storage. Failures are logged only and not returned
// the caller is not notified so this is best-effort cleanup (e.g. when deleting a challenge).
func (uc *ChallengeUseCase) deleteStorageFiles(ctx context.Context, locations []string) {
	if uc.deps.Storage == nil || len(locations) == 0 {
		return
	}

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(deleteStorageFilesConcurrency)

	for _, loc := range locations {
		g.Go(func() error {
			if err := uc.deps.Storage.Delete(ctx, loc); err != nil {
				uc.deps.Logger.WithError(err).Warn("ChallengeUseCase - deleteStorageFiles", logkit.Fields{"location": loc})
			}

			return nil
		})
	}

	_ = g.Wait()
}

func storageCleanupContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), deleteStorageFilesDetachedTimeout)
}
