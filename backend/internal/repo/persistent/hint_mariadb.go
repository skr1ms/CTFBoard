package persistent

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/repo"
)

type HintRepo struct {
	db *sql.DB
}

func NewHintRepo(db *sql.DB) *HintRepo {
	return &HintRepo{db: db}
}

func (r *HintRepo) Create(ctx context.Context, h *entity.Hint) error {
	id := uuid.New().String()
	h.Id = id

	challengeUUID, err := uuid.Parse(h.ChallengeId)
	if err != nil {
		return fmt.Errorf("HintRepo - Create - Parse ChallengeID: %w", err)
	}

	query := squirrel.Insert("hints").
		Columns("id", "challenge_id", "content", "cost", "order_index").
		Values(id, challengeUUID.String(), h.Content, h.Cost, h.OrderIndex)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("HintRepo - Create - BuildQuery: %w", err)
	}

	_, err = r.db.ExecContext(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("HintRepo - Create - ExecQuery: %w", err)
	}

	return nil
}

func (r *HintRepo) GetByID(ctx context.Context, id string) (*entity.Hint, error) {
	uuidID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("HintRepo - GetByID - ParseID: %w", err)
	}

	query := squirrel.Select("id", "challenge_id", "content", "cost", "order_index").
		From("hints").
		Where(squirrel.Eq{"id": uuidID})

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("HintRepo - GetByID - BuildQuery: %w", err)
	}

	var hint entity.Hint
	err = r.db.QueryRowContext(ctx, sqlQuery, args...).Scan(
		&hint.Id,
		&hint.ChallengeId,
		&hint.Content,
		&hint.Cost,
		&hint.OrderIndex,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, entityError.ErrHintNotFound
		}
		return nil, fmt.Errorf("HintRepo - GetByID - Scan: %w", err)
	}

	return &hint, nil
}

func (r *HintRepo) GetByChallengeID(ctx context.Context, challengeId string) ([]*entity.Hint, error) {
	challengeUUID, err := uuid.Parse(challengeId)
	if err != nil {
		return nil, fmt.Errorf("HintRepo - GetByChallengeID - Parse ChallengeID: %w", err)
	}

	query := squirrel.Select("id", "challenge_id", "content", "cost", "order_index").
		From("hints").
		Where(squirrel.Eq{"challenge_id": challengeUUID}).
		OrderBy("order_index ASC")

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("HintRepo - GetByChallengeID - BuildQuery: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("HintRepo - GetByChallengeID - Query: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

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
	uuidID, err := uuid.Parse(h.Id)
	if err != nil {
		return fmt.Errorf("HintRepo - Update - ParseID: %w", err)
	}

	query := squirrel.Update("hints").
		Set("content", h.Content).
		Set("cost", h.Cost).
		Set("order_index", h.OrderIndex).
		Where(squirrel.Eq{"id": uuidID})

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("HintRepo - Update - BuildQuery: %w", err)
	}

	_, err = r.db.ExecContext(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("HintRepo - Update - ExecQuery: %w", err)
	}

	return nil
}

func (r *HintRepo) Delete(ctx context.Context, id string) error {
	uuidID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("HintRepo - Delete - ParseID: %w", err)
	}

	query := squirrel.Delete("hints").
		Where(squirrel.Eq{"id": uuidID})

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("HintRepo - Delete - BuildQuery: %w", err)
	}

	_, err = r.db.ExecContext(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("HintRepo - Delete - ExecQuery: %w", err)
	}

	return nil
}

type HintUnlockRepo struct {
	db *sql.DB
}

func NewHintUnlockRepo(db *sql.DB) *HintUnlockRepo {
	return &HintUnlockRepo{db: db}
}

func (r *HintUnlockRepo) CreateTx(ctx context.Context, tx *sql.Tx, teamId, hintId string) error {
	teamUUID, err := uuid.Parse(teamId)
	if err != nil {
		return fmt.Errorf("HintUnlockRepo - CreateTx - Parse TeamID: %w", err)
	}
	hintUUID, err := uuid.Parse(hintId)
	if err != nil {
		return fmt.Errorf("HintUnlockRepo - CreateTx - Parse HintID: %w", err)
	}

	query := squirrel.Insert("hint_unlocks").
		Columns("id", "hint_id", "team_id").
		Values(uuid.New().String(), hintUUID.String(), teamUUID.String())

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("HintUnlockRepo - CreateTx - BuildQuery: %w", err)
	}

	_, err = tx.ExecContext(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("HintUnlockRepo - CreateTx - ExecQuery: %w", err)
	}

	return nil
}

func (r *HintUnlockRepo) GetByTeamAndHint(ctx context.Context, teamId, hintId string) (*entity.HintUnlock, error) {
	teamUUID, err := uuid.Parse(teamId)
	if err != nil {
		return nil, fmt.Errorf("HintUnlockRepo - GetByTeamAndHint - Parse TeamID: %w", err)
	}
	hintUUID, err := uuid.Parse(hintId)
	if err != nil {
		return nil, fmt.Errorf("HintUnlockRepo - GetByTeamAndHint - Parse HintID: %w", err)
	}

	query := squirrel.Select("id", "hint_id", "team_id", "unlocked_at").
		From("hint_unlocks").
		Where(squirrel.Eq{"team_id": teamUUID, "hint_id": hintUUID})

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("HintUnlockRepo - GetByTeamAndHint - BuildQuery: %w", err)
	}

	var unlock entity.HintUnlock
	err = r.db.QueryRowContext(ctx, sqlQuery, args...).Scan(
		&unlock.Id,
		&unlock.HintId,
		&unlock.TeamId,
		&unlock.UnlockedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, entityError.ErrHintNotFound
		}
		return nil, fmt.Errorf("HintUnlockRepo - GetByTeamAndHint - Scan: %w", err)
	}

	return &unlock, nil
}

func (r *HintUnlockRepo) GetByTeamAndHintTx(ctx context.Context, tx *sql.Tx, teamId, hintId string) (*entity.HintUnlock, error) {
	teamUUID, err := uuid.Parse(teamId)
	if err != nil {
		return nil, fmt.Errorf("HintUnlockRepo - GetByTeamAndHintTx - Parse TeamID: %w", err)
	}
	hintUUID, err := uuid.Parse(hintId)
	if err != nil {
		return nil, fmt.Errorf("HintUnlockRepo - GetByTeamAndHintTx - Parse HintID: %w", err)
	}

	query := squirrel.Select("id", "hint_id", "team_id", "unlocked_at").
		From("hint_unlocks").
		Where(squirrel.Eq{"team_id": teamUUID, "hint_id": hintUUID}).
		Suffix("FOR UPDATE")

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("HintUnlockRepo - GetByTeamAndHintTx - BuildQuery: %w", err)
	}

	var unlock entity.HintUnlock
	err = tx.QueryRowContext(ctx, sqlQuery, args...).Scan(
		&unlock.Id,
		&unlock.HintId,
		&unlock.TeamId,
		&unlock.UnlockedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, entityError.ErrHintNotFound
		}
		return nil, fmt.Errorf("HintUnlockRepo - GetByTeamAndHintTx - Scan: %w", err)
	}

	return &unlock, nil
}

func (r *HintUnlockRepo) GetUnlockedHintIDs(ctx context.Context, teamId, challengeId string) ([]string, error) {
	teamUUID, err := uuid.Parse(teamId)
	if err != nil {
		return nil, fmt.Errorf("HintUnlockRepo - GetUnlockedHintIDs - Parse TeamID: %w", err)
	}
	challengeUUID, err := uuid.Parse(challengeId)
	if err != nil {
		return nil, fmt.Errorf("HintUnlockRepo - GetUnlockedHintIDs - Parse ChallengeID: %w", err)
	}

	query := squirrel.Select("hu.hint_id").
		From("hint_unlocks hu").
		Join("hints h ON h.id = hu.hint_id").
		Where(squirrel.Eq{"hu.team_id": teamUUID, "h.challenge_id": challengeUUID})

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("HintUnlockRepo - GetUnlockedHintIDs - BuildQuery: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("HintUnlockRepo - GetUnlockedHintIDs - Query: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	hintIDs := make([]string, 0)
	for rows.Next() {
		var hintId string
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
