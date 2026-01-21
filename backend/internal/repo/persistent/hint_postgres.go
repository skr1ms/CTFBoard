package persistent

import (
	"context"
	"errors"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/repo"
)

type HintRepo struct {
	pool *pgxpool.Pool
}

func NewHintRepo(pool *pgxpool.Pool) *HintRepo {
	return &HintRepo{pool: pool}
}

func (r *HintRepo) Create(ctx context.Context, h *entity.Hint) error {
	h.Id = uuid.New()

	query := squirrel.Insert("hints").
		Columns("id", "challenge_id", "content", "cost", "order_index").
		Values(h.Id, h.ChallengeId, h.Content, h.Cost, h.OrderIndex).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("HintRepo - Create - BuildQuery: %w", err)
	}

	_, err = r.pool.Exec(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("HintRepo - Create - ExecQuery: %w", err)
	}

	return nil
}

func (r *HintRepo) GetByID(ctx context.Context, id uuid.UUID) (*entity.Hint, error) {
	query := squirrel.Select("id", "challenge_id", "content", "cost", "order_index").
		From("hints").
		Where(squirrel.Eq{"id": id}).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("HintRepo - GetByID - BuildQuery: %w", err)
	}

	var hint entity.Hint
	err = r.pool.QueryRow(ctx, sqlQuery, args...).Scan(
		&hint.Id,
		&hint.ChallengeId,
		&hint.Content,
		&hint.Cost,
		&hint.OrderIndex,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entityError.ErrHintNotFound
		}
		return nil, fmt.Errorf("HintRepo - GetByID - Scan: %w", err)
	}

	return &hint, nil
}

func (r *HintRepo) GetByChallengeID(ctx context.Context, challengeId uuid.UUID) ([]*entity.Hint, error) {
	query := squirrel.Select("id", "challenge_id", "content", "cost", "order_index").
		From("hints").
		Where(squirrel.Eq{"challenge_id": challengeId}).
		OrderBy("order_index ASC").
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("HintRepo - GetByChallengeID - BuildQuery: %w", err)
	}

	rows, err := r.pool.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("HintRepo - GetByChallengeID - Query: %w", err)
	}
	defer rows.Close()

	hints := make([]*entity.Hint, 0)
	for rows.Next() {
		var hint entity.Hint
		if err := rows.Scan(
			&hint.Id,
			&hint.ChallengeId,
			&hint.Content,
			&hint.Cost,
			&hint.OrderIndex,
		); err != nil {
			return nil, fmt.Errorf("HintRepo - GetByChallengeID - Scan: %w", err)
		}
		hints = append(hints, &hint)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("HintRepo - GetByChallengeID - Rows: %w", err)
	}

	return hints, nil
}

func (r *HintRepo) Update(ctx context.Context, h *entity.Hint) error {
	query := squirrel.Update("hints").
		Set("content", h.Content).
		Set("cost", h.Cost).
		Set("order_index", h.OrderIndex).
		Where(squirrel.Eq{"id": h.Id}).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("HintRepo - Update - BuildQuery: %w", err)
	}

	_, err = r.pool.Exec(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("HintRepo - Update - ExecQuery: %w", err)
	}

	return nil
}

func (r *HintRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := squirrel.Delete("hints").
		Where(squirrel.Eq{"id": id}).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("HintRepo - Delete - BuildQuery: %w", err)
	}

	_, err = r.pool.Exec(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("HintRepo - Delete - ExecQuery: %w", err)
	}

	return nil
}

type HintUnlockRepo struct {
	pool *pgxpool.Pool
}

func NewHintUnlockRepo(pool *pgxpool.Pool) *HintUnlockRepo {
	return &HintUnlockRepo{pool: pool}
}

func (r *HintUnlockRepo) GetByTeamAndHint(ctx context.Context, teamId, hintId uuid.UUID) (*entity.HintUnlock, error) {
	query := squirrel.Select("id", "hint_id", "team_id", "unlocked_at").
		From("hint_unlocks").
		Where(squirrel.Eq{"team_id": teamId, "hint_id": hintId}).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("HintUnlockRepo - GetByTeamAndHint - BuildQuery: %w", err)
	}

	var unlock entity.HintUnlock
	err = r.pool.QueryRow(ctx, sqlQuery, args...).Scan(
		&unlock.Id,
		&unlock.HintId,
		&unlock.TeamId,
		&unlock.UnlockedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entityError.ErrHintNotFound
		}
		return nil, fmt.Errorf("HintUnlockRepo - GetByTeamAndHint - Scan: %w", err)
	}

	return &unlock, nil
}

func (r *HintUnlockRepo) GetUnlockedHintIDs(ctx context.Context, teamId, challengeId uuid.UUID) ([]uuid.UUID, error) {
	query := squirrel.Select("hu.hint_id").
		From("hint_unlocks hu").
		Join("hints h ON h.id = hu.hint_id").
		Where(squirrel.Eq{"hu.team_id": teamId, "h.challenge_id": challengeId}).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("HintUnlockRepo - GetUnlockedHintIDs - BuildQuery: %w", err)
	}

	rows, err := r.pool.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("HintUnlockRepo - GetUnlockedHintIDs - Query: %w", err)
	}
	defer rows.Close()

	hintIDs := make([]uuid.UUID, 0)
	for rows.Next() {
		var hintId uuid.UUID
		if err := rows.Scan(&hintId); err != nil {
			return nil, fmt.Errorf("HintUnlockRepo - GetUnlockedHintIDs - Scan: %w", err)
		}
		hintIDs = append(hintIDs, hintId)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("HintUnlockRepo - GetUnlockedHintIDs - Rows: %w", err)
	}

	return hintIDs, nil
}

var _ repo.HintRepository = (*HintRepo)(nil)
var _ repo.HintUnlockRepository = (*HintUnlockRepo)(nil)
