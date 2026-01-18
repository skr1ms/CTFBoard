package persistent

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
)

type AwardRepo struct {
	db *sql.DB
}

func NewAwardRepo(db *sql.DB) *AwardRepo {
	return &AwardRepo{db: db}
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
