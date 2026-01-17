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

type ChallengeRepo struct {
	db *sql.DB
}

func NewChallengeRepo(db *sql.DB) *ChallengeRepo {
	return &ChallengeRepo{db: db}
}

func (r *ChallengeRepo) Create(ctx context.Context, c *entity.Challenge) error {
	id := uuid.New().String()
	c.Id = id

	query := squirrel.Insert("challenges").
		Columns("id", "title", "description", "category", "points", "initial_value", "min_value", "decay", "solve_count", "flag_hash", "is_hidden").
		Values(id, c.Title, c.Description, c.Category, c.Points, c.InitialValue, c.MinValue, c.Decay, c.SolveCount, c.FlagHash, c.IsHidden)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("ChallengeRepo - Create - BuildQuery: %w", err)
	}

	_, err = r.db.ExecContext(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("ChallengeRepo - Create - ExecQuery: %w", err)
	}

	return nil
}

func (r *ChallengeRepo) GetByID(ctx context.Context, id string) (*entity.Challenge, error) {
	uuidID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("ChallengeRepo - GetByID - ParseID: %w", err)
	}

	query := squirrel.Select("id", "title", "description", "category", "points", "initial_value", "min_value", "decay", "solve_count", "flag_hash", "is_hidden").
		From("challenges").
		Where(squirrel.Eq{"id": uuidID})

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("ChallengeRepo - GetByID - BuildQuery: %w", err)
	}

	var challenge entity.Challenge
	err = r.db.QueryRowContext(ctx, sqlQuery, args...).Scan(
		&challenge.Id,
		&challenge.Title,
		&challenge.Description,
		&challenge.Category,
		&challenge.Points,
		&challenge.InitialValue,
		&challenge.MinValue,
		&challenge.Decay,
		&challenge.SolveCount,
		&challenge.FlagHash,
		&challenge.IsHidden,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, entityError.ErrChallengeNotFound
		}
		return nil, fmt.Errorf("ChallengeRepo - GetByID - Scan: %w", err)
	}

	return &challenge, nil
}

func (r *ChallengeRepo) GetAll(ctx context.Context, teamId *string) ([]*repo.ChallengeWithSolved, error) {
	var query squirrel.SelectBuilder

	if teamId != nil {
		teamUUID, err := uuid.Parse(*teamId)
		if err != nil {
			return nil, fmt.Errorf("ChallengeRepo - GetAll - Parse TeamID: %w", err)
		}
		query = squirrel.Select(
			"c.id", "c.title", "c.description", "c.category", "c.points", "c.initial_value", "c.min_value", "c.decay", "c.solve_count", "c.flag_hash", "c.is_hidden",
			"CASE WHEN s.id IS NOT NULL THEN 1 ELSE 0 END as solved",
		).
			From("challenges c").
			LeftJoin("solves s ON c.id = s.challenge_id AND s.team_id = ?", teamUUID.String()).
			Where(squirrel.Eq{"c.is_hidden": false})
	} else {
		query = squirrel.Select(
			"c.id", "c.title", "c.description", "c.category", "c.points", "c.initial_value", "c.min_value", "c.decay", "c.solve_count", "c.flag_hash", "c.is_hidden",
			"0 as solved",
		).
			From("challenges c").
			Where(squirrel.Eq{"c.is_hidden": false})
	}

	sqlQuery, args, err := query.PlaceholderFormat(squirrel.Question).ToSql()
	if err != nil {
		return nil, fmt.Errorf("ChallengeRepo - GetAll - BuildQuery: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("ChallengeRepo - GetAll - Query: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	result := make([]*repo.ChallengeWithSolved, 0)
	for rows.Next() {
		var challenge entity.Challenge
		var solved int
		if err := rows.Scan(
			&challenge.Id,
			&challenge.Title,
			&challenge.Description,
			&challenge.Category,
			&challenge.Points,
			&challenge.InitialValue,
			&challenge.MinValue,
			&challenge.Decay,
			&challenge.SolveCount,
			&challenge.FlagHash,
			&challenge.IsHidden,
			&solved,
		); err != nil {
			return nil, fmt.Errorf("ChallengeRepo - GetAll - Scan: %w", err)
		}
		result = append(result, &repo.ChallengeWithSolved{
			Challenge: &challenge,
			Solved:    solved == 1,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ChallengeRepo - GetAll - Rows: %w", err)
	}

	return result, nil
}

func (r *ChallengeRepo) Update(ctx context.Context, c *entity.Challenge) error {
	uuidID, err := uuid.Parse(c.Id)
	if err != nil {
		return fmt.Errorf("ChallengeRepo - Update - ParseID: %w", err)
	}

	query := squirrel.Update("challenges").
		Set("title", c.Title).
		Set("description", c.Description).
		Set("category", c.Category).
		Set("points", c.Points).
		Set("initial_value", c.InitialValue).
		Set("min_value", c.MinValue).
		Set("decay", c.Decay).
		Set("flag_hash", c.FlagHash).
		Set("is_hidden", c.IsHidden).
		Where(squirrel.Eq{"id": uuidID})

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("ChallengeRepo - Update - BuildQuery: %w", err)
	}

	_, err = r.db.ExecContext(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("ChallengeRepo - Update - ExecQuery: %w", err)
	}

	return nil
}

func (r *ChallengeRepo) Delete(ctx context.Context, id string) error {
	uuidID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("ChallengeRepo - Delete - ParseID: %w", err)
	}

	query := squirrel.Delete("challenges").
		Where(squirrel.Eq{"id": uuidID})

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("ChallengeRepo - Delete - BuildQuery: %w", err)
	}

	_, err = r.db.ExecContext(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("ChallengeRepo - Delete - ExecQuery: %w", err)
	}

	return nil
}

func (r *ChallengeRepo) IncrementSolveCount(ctx context.Context, id string) (int, error) {
	uuidID, err := uuid.Parse(id)
	if err != nil {
		return 0, fmt.Errorf("ChallengeRepo - IncrementSolveCount - ParseID: %w", err)
	}

	updateQuery := squirrel.Update("challenges").
		Set("solve_count", squirrel.Expr("solve_count + 1")).
		Where(squirrel.Eq{"id": uuidID.String()})

	updateSQL, updateArgs, err := updateQuery.ToSql()
	if err != nil {
		return 0, fmt.Errorf("ChallengeRepo - IncrementSolveCount - BuildUpdateQuery: %w", err)
	}

	_, err = r.db.ExecContext(ctx, updateSQL, updateArgs...)
	if err != nil {
		return 0, fmt.Errorf("ChallengeRepo - IncrementSolveCount - Exec: %w", err)
	}

	selectQuery := squirrel.Select("solve_count").
		From("challenges").
		Where(squirrel.Eq{"id": uuidID.String()})

	selectSQL, selectArgs, err := selectQuery.ToSql()
	if err != nil {
		return 0, fmt.Errorf("ChallengeRepo - IncrementSolveCount - BuildSelectQuery: %w", err)
	}

	var solveCount int
	err = r.db.QueryRowContext(ctx, selectSQL, selectArgs...).Scan(&solveCount)
	if err != nil {
		return 0, fmt.Errorf("ChallengeRepo - IncrementSolveCount - Query: %w", err)
	}

	return solveCount, nil
}

func (r *ChallengeRepo) UpdatePoints(ctx context.Context, id string, points int) error {
	uuidID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("ChallengeRepo - UpdatePoints - ParseID: %w", err)
	}

	query := squirrel.Update("challenges").
		Set("points", points).
		Where(squirrel.Eq{"id": uuidID.String()})

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("ChallengeRepo - UpdatePoints - BuildQuery: %w", err)
	}

	_, err = r.db.ExecContext(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("ChallengeRepo - UpdatePoints - Exec: %w", err)
	}

	return nil
}

func (r *ChallengeRepo) GetByIDTx(ctx context.Context, tx *sql.Tx, id string) (*entity.Challenge, error) {
	uuidID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("ChallengeRepo - GetByIDTx - ParseID: %w", err)
	}

	query := squirrel.Select("id", "title", "description", "category", "points", "initial_value", "min_value", "decay", "solve_count", "flag_hash", "is_hidden").
		From("challenges").
		Where(squirrel.Eq{"id": uuidID}).
		Suffix("FOR UPDATE")

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("ChallengeRepo - GetByIDTx - BuildQuery: %w", err)
	}

	var challenge entity.Challenge
	err = tx.QueryRowContext(ctx, sqlQuery, args...).Scan(
		&challenge.Id,
		&challenge.Title,
		&challenge.Description,
		&challenge.Category,
		&challenge.Points,
		&challenge.InitialValue,
		&challenge.MinValue,
		&challenge.Decay,
		&challenge.SolveCount,
		&challenge.FlagHash,
		&challenge.IsHidden,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, entityError.ErrChallengeNotFound
		}
		return nil, fmt.Errorf("ChallengeRepo - GetByIDTx - Scan: %w", err)
	}

	return &challenge, nil
}

func (r *ChallengeRepo) IncrementSolveCountTx(ctx context.Context, tx *sql.Tx, id string) (int, error) {
	uuidID, err := uuid.Parse(id)
	if err != nil {
		return 0, fmt.Errorf("ChallengeRepo - IncrementSolveCountTx - ParseID: %w", err)
	}

	updateQuery := squirrel.Update("challenges").
		Set("solve_count", squirrel.Expr("solve_count + 1")).
		Where(squirrel.Eq{"id": uuidID.String()})

	updateSQL, updateArgs, err := updateQuery.ToSql()
	if err != nil {
		return 0, fmt.Errorf("ChallengeRepo - IncrementSolveCountTx - BuildUpdateQuery: %w", err)
	}

	_, err = tx.ExecContext(ctx, updateSQL, updateArgs...)
	if err != nil {
		return 0, fmt.Errorf("ChallengeRepo - IncrementSolveCountTx - Exec: %w", err)
	}

	selectQuery := squirrel.Select("solve_count").
		From("challenges").
		Where(squirrel.Eq{"id": uuidID.String()})

	selectSQL, selectArgs, err := selectQuery.ToSql()
	if err != nil {
		return 0, fmt.Errorf("ChallengeRepo - IncrementSolveCountTx - BuildSelectQuery: %w", err)
	}

	var solveCount int
	err = tx.QueryRowContext(ctx, selectSQL, selectArgs...).Scan(&solveCount)
	if err != nil {
		return 0, fmt.Errorf("ChallengeRepo - IncrementSolveCountTx - Query: %w", err)
	}

	return solveCount, nil
}

func (r *ChallengeRepo) UpdatePointsTx(ctx context.Context, tx *sql.Tx, id string, points int) error {
	uuidID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("ChallengeRepo - UpdatePointsTx - ParseID: %w", err)
	}

	query := squirrel.Update("challenges").
		Set("points", points).
		Where(squirrel.Eq{"id": uuidID.String()})

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("ChallengeRepo - UpdatePointsTx - BuildQuery: %w", err)
	}

	_, err = tx.ExecContext(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("ChallengeRepo - UpdatePointsTx - Exec: %w", err)
	}

	return nil
}
