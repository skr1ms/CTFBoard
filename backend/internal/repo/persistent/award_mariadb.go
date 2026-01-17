package persistent

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/repo"
)

type AwardRepo struct {
	db *sql.DB
}

func NewAwardRepo(db *sql.DB) *AwardRepo {
	return &AwardRepo{db: db}
}

func (r *AwardRepo) CreateTx(ctx context.Context, tx *sql.Tx, a *entity.Award) error {
	id := uuid.New().String()
	a.Id = id

	teamUUID, err := uuid.Parse(a.TeamId)
	if err != nil {
		return fmt.Errorf("AwardRepo - CreateTx - Parse TeamID: %w", err)
	}

	query := squirrel.Insert("awards").
		Columns("id", "team_id", "value", "description", "created_at").
		Values(id, teamUUID.String(), a.Value, a.Description, time.Now())

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("AwardRepo - CreateTx - BuildQuery: %w", err)
	}

	_, err = tx.ExecContext(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("AwardRepo - CreateTx - ExecQuery: %w", err)
	}

	return nil
}

func (r *AwardRepo) GetTeamTotalAwards(ctx context.Context, teamId string) (int, error) {
	teamUUID, err := uuid.Parse(teamId)
	if err != nil {
		return 0, fmt.Errorf("AwardRepo - GetTeamTotalAwards - Parse TeamID: %w", err)
	}

	query := squirrel.Select("COALESCE(SUM(value), 0)").
		From("awards").
		Where(squirrel.Eq{"team_id": teamUUID})

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return 0, fmt.Errorf("AwardRepo - GetTeamTotalAwards - BuildQuery: %w", err)
	}

	var total int
	err = r.db.QueryRowContext(ctx, sqlQuery, args...).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("AwardRepo - GetTeamTotalAwards - Scan: %w", err)
	}

	return total, nil
}

type TxRepo struct {
	db *sql.DB
}

func NewTxRepo(db *sql.DB) *TxRepo {
	return &TxRepo{db: db}
}

func (r *TxRepo) BeginTx(ctx context.Context) (*sql.Tx, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("TxRepo - BeginTx: %w", err)
	}
	return tx, nil
}

var _ repo.AwardRepository = (*AwardRepo)(nil)
var _ repo.TxRepository = (*TxRepo)(nil)
