package persistent

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/repo"
)

// TxRepo

type TxRepo struct {
	pool *pgxpool.Pool
}

func NewTxRepo(pool *pgxpool.Pool) *TxRepo {
	return &TxRepo{pool: pool}
}

func (r *TxRepo) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return r.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.ReadCommitted,
	})
}

func (r *TxRepo) BeginSerializableTx(ctx context.Context) (pgx.Tx, error) {
	return r.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable,
	})
}

func (r *TxRepo) RunTransaction(ctx context.Context, fn func(context.Context, pgx.Tx) error) error {
	tx, err := r.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("RunTransaction - BeginTx: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
	}()

	if err := fn(ctx, tx); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("RunTransaction - FnError: %w, RollbackError: %v", err, rbErr)
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("RunTransaction - Commit: %w", err)
	}

	return nil
}

// UserRepo Tx Methods

func (r *TxRepo) CreateUserTx(ctx context.Context, tx pgx.Tx, user *entity.User) error {
	user.CreatedAt = time.Now()

	query := squirrel.Insert("users").
		Columns("username", "email", "password_hash", "role", "created_at").
		Values(user.Username, user.Email, user.PasswordHash, user.Role, user.CreatedAt).
		Suffix("RETURNING id").
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("TxRepo - CreateUserTx - BuildQuery: %w", err)
	}

	err = tx.QueryRow(ctx, sqlQuery, args...).Scan(&user.Id)
	if err != nil {
		return fmt.Errorf("TxRepo - CreateUserTx - Exec: %w", err)
	}

	return nil
}

func (r *TxRepo) UpdateUserTeamIDTx(ctx context.Context, tx pgx.Tx, userId uuid.UUID, teamId *uuid.UUID) error {
	updateBuilder := squirrel.Update("users").
		Where(squirrel.Eq{"id": userId}).
		PlaceholderFormat(squirrel.Dollar)

	if teamId != nil {
		updateBuilder = updateBuilder.Set("team_id", *teamId)
	} else {
		updateBuilder = updateBuilder.Set("team_id", nil)
	}

	sqlQuery, args, err := updateBuilder.ToSql()
	if err != nil {
		return fmt.Errorf("TxRepo - UpdateUserTeamIDTx - BuildQuery: %w", err)
	}

	cmdTag, err := tx.Exec(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("TxRepo - UpdateUserTeamIDTx - Exec: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return entityError.ErrUserNotFound
	}

	return nil
}

// TeamRepo Tx Methods

func (r *TxRepo) CreateTeamTx(ctx context.Context, tx pgx.Tx, team *entity.Team) error {
	team.CreatedAt = time.Now()

	query := squirrel.Insert("teams").
		Columns("name", "invite_token", "captain_id", "created_at").
		Values(team.Name, team.InviteToken, team.CaptainId, team.CreatedAt).
		Suffix("RETURNING id").
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("TxRepo - CreateTeamTx - BuildQuery: %w", err)
	}

	err = tx.QueryRow(ctx, sqlQuery, args...).Scan(&team.Id)
	if err != nil {
		return fmt.Errorf("TxRepo - CreateTeamTx - Exec: %w", err)
	}

	return nil
}

// ChallengeRepo Tx Methods

func (r *TxRepo) GetChallengeByIDTx(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*entity.Challenge, error) {
	query := squirrel.Select("id", "title", "description", "category", "points", "initial_value", "min_value", "decay", "solve_count", "flag_hash", "is_hidden").
		From("challenges").
		Where(squirrel.Eq{"id": id}).
		Suffix("FOR UPDATE").
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("TxRepo - GetChallengeByIDTx - BuildQuery: %w", err)
	}

	var challenge entity.Challenge
	err = tx.QueryRow(ctx, sqlQuery, args...).Scan(
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
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entityError.ErrChallengeNotFound
		}
		return nil, fmt.Errorf("TxRepo - GetChallengeByIDTx - Scan: %w", err)
	}

	return &challenge, nil
}

func (r *TxRepo) IncrementChallengeSolveCountTx(ctx context.Context, tx pgx.Tx, id uuid.UUID) (int, error) {
	query := squirrel.Update("challenges").
		Set("solve_count", squirrel.Expr("solve_count + 1")).
		Where(squirrel.Eq{"id": id}).
		Suffix("RETURNING solve_count").
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return 0, fmt.Errorf("TxRepo - IncrementChallengeSolveCountTx - BuildQuery: %w", err)
	}

	var solveCount int
	err = tx.QueryRow(ctx, sqlQuery, args...).Scan(&solveCount)
	if err != nil {
		return 0, fmt.Errorf("TxRepo - IncrementChallengeSolveCountTx - Query: %w", err)
	}

	return solveCount, nil
}

func (r *TxRepo) UpdateChallengePointsTx(ctx context.Context, tx pgx.Tx, id uuid.UUID, points int) error {
	query := squirrel.Update("challenges").
		Set("points", points).
		Where(squirrel.Eq{"id": id}).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("TxRepo - UpdateChallengePointsTx - BuildQuery: %w", err)
	}

	cmdTag, err := tx.Exec(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("TxRepo - UpdateChallengePointsTx - Exec: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return entityError.ErrChallengeNotFound
	}

	return nil
}

// SolveRepo Tx Methods

func (r *TxRepo) CreateSolveTx(ctx context.Context, tx pgx.Tx, s *entity.Solve) error {
	s.Id = uuid.New()
	s.SolvedAt = time.Now()

	query := squirrel.Insert("solves").
		Columns("id", "user_id", "team_id", "challenge_id", "solved_at").
		Values(s.Id, s.UserId, s.TeamId, s.ChallengeId, s.SolvedAt).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("TxRepo - CreateSolveTx - BuildQuery: %w", err)
	}

	_, err = tx.Exec(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("TxRepo - CreateSolveTx - Exec: %w", err)
	}

	return nil
}

func (r *TxRepo) GetSolveByTeamAndChallengeTx(ctx context.Context, tx pgx.Tx, teamId, challengeId uuid.UUID) (*entity.Solve, error) {
	query := squirrel.Select("id", "user_id", "team_id", "challenge_id", "solved_at").
		From("solves").
		Where(squirrel.Eq{"team_id": teamId, "challenge_id": challengeId}).
		Suffix("FOR UPDATE").
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("TxRepo - GetSolveByTeamAndChallengeTx - BuildQuery: %w", err)
	}

	var solve entity.Solve
	err = tx.QueryRow(ctx, sqlQuery, args...).Scan(
		&solve.Id,
		&solve.UserId,
		&solve.TeamId,
		&solve.ChallengeId,
		&solve.SolvedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entityError.ErrSolveNotFound
		}
		return nil, fmt.Errorf("TxRepo - GetSolveByTeamAndChallengeTx - Scan: %w", err)
	}

	return &solve, nil
}

func (r *TxRepo) GetTeamScoreTx(ctx context.Context, tx pgx.Tx, teamId uuid.UUID) (int, error) {
	query := squirrel.Select(
		"COALESCE(SUM(c.points), 0) + COALESCE((SELECT SUM(value) FROM awards WHERE team_id = $1), 0) as total_points",
	).
		From("solves s").
		RightJoin("(SELECT 1) dummy ON true").
		LeftJoin("challenges c ON c.id = s.challenge_id AND s.team_id = $1").
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, _, err := query.ToSql()
	if err != nil {
		return 0, fmt.Errorf("TxRepo - GetTeamScoreTx - BuildQuery: %w", err)
	}

	var points int
	err = tx.QueryRow(ctx, sqlQuery, teamId).Scan(&points)
	if err != nil {
		return 0, fmt.Errorf("TxRepo - GetTeamScoreTx - Scan: %w", err)
	}

	return points, nil
}

// HintUnlockRepo Tx Methods

func (r *TxRepo) CreateHintUnlockTx(ctx context.Context, tx pgx.Tx, teamId, hintId uuid.UUID) error {
	query := squirrel.Insert("hint_unlocks").
		Columns("id", "hint_id", "team_id").
		Values(uuid.New(), hintId, teamId).
		Suffix("ON CONFLICT (hint_id, team_id) DO NOTHING").
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("TxRepo - CreateHintUnlockTx - BuildQuery: %w", err)
	}

	_, err = tx.Exec(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("TxRepo - CreateHintUnlockTx - Exec: %w", err)
	}

	return nil
}

func (r *TxRepo) GetHintUnlockByTeamAndHintTx(ctx context.Context, tx pgx.Tx, teamId, hintId uuid.UUID) (*entity.HintUnlock, error) {
	query := squirrel.Select("id", "hint_id", "team_id", "unlocked_at").
		From("hint_unlocks").
		Where(squirrel.Eq{"team_id": teamId, "hint_id": hintId}).
		Suffix("FOR UPDATE").
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("TxRepo - GetHintUnlockByTeamAndHintTx - BuildQuery: %w", err)
	}

	var unlock entity.HintUnlock
	err = tx.QueryRow(ctx, sqlQuery, args...).Scan(
		&unlock.Id,
		&unlock.HintId,
		&unlock.TeamId,
		&unlock.UnlockedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entityError.ErrHintNotFound
		}
		return nil, fmt.Errorf("TxRepo - GetHintUnlockByTeamAndHintTx - Scan: %w", err)
	}

	return &unlock, nil
}

// AwardRepo Tx Methods

func (r *TxRepo) CreateAwardTx(ctx context.Context, tx pgx.Tx, a *entity.Award) error {
	a.Id = uuid.New()
	a.CreatedAt = time.Now()

	query := squirrel.Insert("awards").
		Columns("id", "team_id", "value", "description", "created_at").
		Values(a.Id, a.TeamId, a.Value, a.Description, a.CreatedAt).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("TxRepo - CreateAwardTx - BuildQuery: %w", err)
	}

	_, err = tx.Exec(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("TxRepo - CreateAwardTx - Exec: %w", err)
	}

	return nil
}

func (r *TxRepo) LockTeamTx(ctx context.Context, tx pgx.Tx, teamId uuid.UUID) error {
	query := squirrel.Select("id").
		From("teams").
		Where(squirrel.Eq{"id": teamId}).
		Suffix("FOR UPDATE").
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("TxRepo - LockTeamTx - BuildQuery: %w", err)
	}

	var id uuid.UUID
	err = tx.QueryRow(ctx, sqlQuery, args...).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entityError.ErrTeamNotFound
		}
		return fmt.Errorf("TxRepo - LockTeamTx - Scan: %w", err)
	}

	return nil
}

var _ repo.TxRepository = (*TxRepo)(nil)
