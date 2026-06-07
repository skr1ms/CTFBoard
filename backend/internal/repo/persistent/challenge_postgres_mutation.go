package persistent

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-pgkit/pgutil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
)

func (r *ChallengeRepo) Create(ctx context.Context, c *domain.Challenge) error {
	pts, err := intToInt32Safe(c.Points)
	if err != nil {
		return fmt.Errorf("ChallengeRepo - Create - Points: %w", err)
	}

	initialValue, err := intToInt32Safe(c.InitialValue)
	if err != nil {
		return fmt.Errorf("ChallengeRepo - Create - InitialValue: %w", err)
	}

	minValue, err := intToInt32Safe(c.MinValue)
	if err != nil {
		return fmt.Errorf("ChallengeRepo - Create - MinValue: %w", err)
	}

	decay, err := intToInt32Safe(c.Decay)
	if err != nil {
		return fmt.Errorf("ChallengeRepo - Create - Decay: %w", err)
	}

	solveCount, err := intToInt32Safe(c.SolveCount)
	if err != nil {
		return fmt.Errorf("ChallengeRepo - Create - SolveCount: %w", err)
	}

	maxAttempts, err := intToInt32Safe(c.MaxAttempts)
	if err != nil {
		return fmt.Errorf("ChallengeRepo - Create - MaxAttempts: %w", err)
	}

	position, err := intToInt32Safe(c.Position)
	if err != nil {
		return fmt.Errorf("ChallengeRepo - Create - Position: %w", err)
	}

	state := c.State
	if state == "" {
		state = domain.ChallengeStateHidden
	}

	now := time.Now()
	if err := r.Q(ctx).CreateChallenge(ctx, sqlc.CreateChallengeParams{
		ID:                c.ID,
		Title:             c.Title,
		Description:       c.Description,
		Category:          c.Category,
		Points:            pts,
		InitialValue:      initialValue,
		MinValue:          minValue,
		Decay:             decay,
		SolveCount:        solveCount,
		FlagHash:          c.FlagHash,
		Attribution:       c.Attribution,
		ConnectionInfo:    c.ConnectionInfo,
		MaxAttempts:       maxAttempts,
		MaxAttemptsWindow: int64(c.MaxAttemptsWindow),
		Position:          position,
		NextChallengeID:   c.NextChallengeID,
		State:             state,
		IsRegex:           c.IsRegex,
		IsCaseInsensitive: c.IsCaseInsensitive,
		FlagRegex:         c.FlagRegex,
		FlagFormatRegex:   c.FlagFormatRegex,
		CreatedAt:         pgutil.TimeToTimestamptz(&now),
		UpdatedAt:         pgutil.TimeToTimestamptz(&now),
	}); err != nil {
		return fmt.Errorf("ChallengeRepo - Create: %w", err)
	}

	return nil
}

func (r *ChallengeRepo) Update(ctx context.Context, c *domain.Challenge) error {
	pts, err := intToInt32Safe(c.Points)
	if err != nil {
		return fmt.Errorf("ChallengeRepo - Update - Points: %w", err)
	}

	initialValue, err := intToInt32Safe(c.InitialValue)
	if err != nil {
		return fmt.Errorf("ChallengeRepo - Update - InitialValue: %w", err)
	}

	minValue, err := intToInt32Safe(c.MinValue)
	if err != nil {
		return fmt.Errorf("ChallengeRepo - Update - MinValue: %w", err)
	}

	decay, err := intToInt32Safe(c.Decay)
	if err != nil {
		return fmt.Errorf("ChallengeRepo - Update - Decay: %w", err)
	}

	maxAttempts, err := intToInt32Safe(c.MaxAttempts)
	if err != nil {
		return fmt.Errorf("ChallengeRepo - Update - MaxAttempts: %w", err)
	}

	position, err := intToInt32Safe(c.Position)
	if err != nil {
		return fmt.Errorf("ChallengeRepo - Update - Position: %w", err)
	}

	state := c.State
	if state == "" {
		state = domain.ChallengeStateHidden
	}

	c.UpdatedAt = time.Now()
	if err := r.Q(ctx).UpdateChallenge(ctx, sqlc.UpdateChallengeParams{
		ID:                c.ID,
		Title:             c.Title,
		Description:       c.Description,
		Category:          c.Category,
		Points:            pts,
		InitialValue:      initialValue,
		MinValue:          minValue,
		Decay:             decay,
		FlagHash:          c.FlagHash,
		Attribution:       c.Attribution,
		ConnectionInfo:    c.ConnectionInfo,
		MaxAttempts:       maxAttempts,
		MaxAttemptsWindow: int64(c.MaxAttemptsWindow),
		Position:          position,
		NextChallengeID:   c.NextChallengeID,
		State:             state,
		IsRegex:           c.IsRegex,
		IsCaseInsensitive: c.IsCaseInsensitive,
		FlagRegex:         c.FlagRegex,
		FlagFormatRegex:   c.FlagFormatRegex,
		UpdatedAt:         pgutil.TimeToTimestamptz(&c.UpdatedAt),
	}); err != nil {
		return fmt.Errorf("ChallengeRepo - Update: %w", err)
	}

	return nil
}

func (r *ChallengeRepo) GetByIDForUpdate(ctx context.Context, ID uuid.UUID) (*domain.Challenge, error) {
	row, err := GetOrNotFound(func() (sqlc.GetChallengeByIDForUpdateRow, error) { return r.Q(ctx).GetChallengeByIDForUpdate(ctx, ID) },
		apperr.ErrChallengeNotFound, "ChallengeRepo - GetByIDForUpdate")
	if err != nil {
		return nil, err
	}

	return toDomainChallenge(fieldsFromGetByIDForUpdate(row).toChallengeRow()), nil
}

func (r *ChallengeRepo) DecrementSolveCount(ctx context.Context, ID uuid.UUID) (int, error) {
	n, err := GetOrNotFound(func() (int32, error) { return r.Q(ctx).DecrementChallengeSolveCount(ctx, ID) },
		apperr.ErrChallengeNotFound, "ChallengeRepo - DecrementSolveCount")
	if err != nil {
		return 0, err
	}

	return int(n), nil
}

func (r *ChallengeRepo) BatchDecrementSolveCount(ctx context.Context, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}

	err := r.Q(ctx).BatchDecrementChallengeSolveCount(ctx, ids)
	if err != nil {
		return fmt.Errorf("ChallengeRepo - BatchDecrementSolveCount: %w", err)
	}

	return nil
}

func (r *ChallengeRepo) BatchIncrementSolveCount(ctx context.Context, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}

	err := r.Q(ctx).BatchIncrementChallengeSolveCount(ctx, ids)
	if err != nil {
		return fmt.Errorf("ChallengeRepo - BatchIncrementSolveCount: %w", err)
	}

	return nil
}

// BatchUpdatePoints updates the points column for multiple challenges in a
// single query. Each int value is range-checked and converted to int32 before
// the call; a length mismatch between ids and points is returned as an error.
func (r *ChallengeRepo) BatchUpdatePoints(ctx context.Context, ids []uuid.UUID, points []int) error {
	if len(ids) == 0 {
		return nil
	}

	if len(ids) != len(points) {
		return fmt.Errorf("ChallengeRepo - BatchUpdatePoints: ids and points length mismatch (%d != %d)", len(ids), len(points))
	}

	pts := make([]int32, len(points))
	for i, p := range points {
		v, err := intToInt32Safe(p)
		if err != nil {
			return fmt.Errorf("ChallengeRepo - BatchUpdatePoints: %w", err)
		}

		pts[i] = v
	}

	err := r.Q(ctx).BatchUpdateChallengePoints(ctx, sqlc.BatchUpdateChallengePointsParams{Column1: ids, Column2: pts})
	if err != nil {
		return fmt.Errorf("ChallengeRepo - BatchUpdatePoints: %w", err)
	}

	return nil
}

// RecalculateSolveCounts refreshes the solve_count column for the given
// challenge IDs in two steps: first recomputes counts from the solves table,
// then explicitly zeros out any challenge whose count would remain stale at a
// positive value despite having no solves.
func (r *ChallengeRepo) RecalculateSolveCounts(ctx context.Context, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}

	if err := r.Q(ctx).RecalculateChallengeSolveCounts(ctx, ids); err != nil {
		return fmt.Errorf("ChallengeRepo - RecalculateSolveCounts: %w", err)
	}

	if err := r.Q(ctx).ZeroChallengeSolveCountsIfNoSolves(ctx, ids); err != nil {
		return fmt.Errorf("ChallengeRepo - RecalculateSolveCounts - ZeroIfNoSolves: %w", err)
	}

	return nil
}

// GetAllDynamicIDs returns the IDs of all challenges that use dynamic scoring
// (initial_value > 0 and decay > 0). Used by admin recalc to rebuild points for
// every dynamic challenge in one pass.
func (r *ChallengeRepo) GetAllDynamicIDs(ctx context.Context) ([]uuid.UUID, error) {
	ids, err := r.Q(ctx).GetAllDynamicChallengeIDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("ChallengeRepo - GetAllDynamicIDs: %w", err)
	}

	return ids, nil
}

// SetTags replaces the tag set for a challenge: deletes all existing tag
// associations, then inserts tagIDs with ON CONFLICT DO NOTHING.
func (r *ChallengeRepo) SetTags(ctx context.Context, challengeID uuid.UUID, tagIDs []uuid.UUID) error {
	if err := requireTx(ctx, "ChallengeRepo - SetTags"); err != nil {
		return err
	}

	if err := r.Q(ctx).DeleteChallengeTags(ctx, challengeID); err != nil {
		return fmt.Errorf("ChallengeRepo - SetTags - Delete: %w", err)
	}

	if len(tagIDs) == 0 {
		return nil
	}

	if err := r.Q(ctx).InsertChallengeTags(ctx, sqlc.InsertChallengeTagsParams{
		ChallengeID: challengeID,
		TagIds:      tagIDs,
	}); err != nil {
		return fmt.Errorf("ChallengeRepo - SetTags - Insert: %w", err)
	}

	return nil
}
