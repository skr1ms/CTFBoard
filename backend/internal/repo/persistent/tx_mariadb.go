package persistent

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/repo"
)

// TxRepo

type TxRepo struct {
	db *sql.DB
}

func NewTxRepo(db *sql.DB) *TxRepo {
	return &TxRepo{db: db}
}

func (r *TxRepo) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return r.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
	})
}

func (r *TxRepo) RunTransaction(ctx context.Context, fn func(context.Context, *sql.Tx) error) error {
	tx, err := r.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("RunTransaction - BeginTx: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(ctx, tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("RunTransaction - FnError: %w, RollbackError: %v", err, rbErr)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("RunTransaction - Commit: %w", err)
	}

	return nil
}

// UserRepo Tx Methods

func (r *TxRepo) CreateUserTx(ctx context.Context, tx *sql.Tx, user *entity.User) error {
	user.Id = uuid.New().String()

	query := squirrel.Insert("users").
		Columns("id", "username", "email", "password_hash", "role", "created_at").
		Values(user.Id, user.Username, user.Email, user.PasswordHash, user.Role, time.Now())

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("TxRepo - CreateUserTx - BuildQuery: %w", err)
	}

	_, err = tx.ExecContext(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("TxRepo - CreateUserTx - ExecQuery: %w", err)
	}

	return nil
}

func (r *TxRepo) UpdateUserTeamIDTx(ctx context.Context, tx *sql.Tx, userId string, teamId *string) error {
	uuidID, err := uuid.Parse(userId)
	if err != nil {
		return fmt.Errorf("TxRepo - UpdateUserTeamIDTx - Parse UserID: %w", err)
	}

	updateBuilder := squirrel.Update("users").Where(squirrel.Eq{"id": uuidID.String()})

	if teamId != nil {
		teamUUID, err := uuid.Parse(*teamId)
		if err != nil {
			return fmt.Errorf("TxRepo - UpdateUserTeamIDTx - Parse TeamID: %w", err)
		}
		updateBuilder = updateBuilder.Set("team_id", teamUUID.String())
	} else {
		updateBuilder = updateBuilder.Set("team_id", nil)
	}

	sqlQuery, args, err := updateBuilder.ToSql()
	if err != nil {
		return fmt.Errorf("TxRepo - UpdateUserTeamIDTx - BuildQuery: %w", err)
	}

	_, err = tx.ExecContext(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("TxRepo - UpdateUserTeamIDTx - ExecQuery: %w", err)
	}

	return nil
}

// TeamRepo Tx Methods

func (r *TxRepo) CreateTeamTx(ctx context.Context, tx *sql.Tx, team *entity.Team) error {
	team.Id = uuid.New().String()

	captainUUID, err := uuid.Parse(team.CaptainId)
	if err != nil {
		return fmt.Errorf("TxRepo - CreateTeamTx - Parse CaptainID: %w", err)
	}

	query := squirrel.Insert("teams").
		Columns("id", "name", "invite_token", "captain_id", "created_at").
		Values(team.Id, team.Name, team.InviteToken, captainUUID.String(), time.Now())

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("TxRepo - CreateTeamTx - BuildQuery: %w", err)
	}

	_, err = tx.ExecContext(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("TxRepo - CreateTeamTx - ExecQuery: %w", err)
	}

	return nil
}

// ChallengeRepo Tx Methods

func (r *TxRepo) GetChallengeByIDTx(ctx context.Context, tx *sql.Tx, id string) (*entity.Challenge, error) {
	uuidID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("TxRepo - GetChallengeByIDTx - ParseID: %w", err)
	}

	query := squirrel.Select("id", "title", "description", "category", "points", "initial_value", "min_value", "decay", "solve_count", "flag_hash", "is_hidden").
		From("challenges").
		Where(squirrel.Eq{"id": uuidID}).
		Suffix("FOR UPDATE")

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("TxRepo - GetChallengeByIDTx - BuildQuery: %w", err)
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
		return nil, fmt.Errorf("TxRepo - GetChallengeByIDTx - Scan: %w", err)
	}

	return &challenge, nil
}

func (r *TxRepo) IncrementChallengeSolveCountTx(ctx context.Context, tx *sql.Tx, id string) (int, error) {
	uuidID, err := uuid.Parse(id)
	if err != nil {
		return 0, fmt.Errorf("TxRepo - IncrementChallengeSolveCountTx - ParseID: %w", err)
	}

	updateQuery := squirrel.Update("challenges").
		Set("solve_count", squirrel.Expr("solve_count + 1")).
		Where(squirrel.Eq{"id": uuidID.String()})

	updateSQL, updateArgs, err := updateQuery.ToSql()
	if err != nil {
		return 0, fmt.Errorf("TxRepo - IncrementChallengeSolveCountTx - BuildUpdateQuery: %w", err)
	}

	_, err = tx.ExecContext(ctx, updateSQL, updateArgs...)
	if err != nil {
		return 0, fmt.Errorf("TxRepo - IncrementChallengeSolveCountTx - Exec: %w", err)
	}

	selectQuery := squirrel.Select("solve_count").
		From("challenges").
		Where(squirrel.Eq{"id": uuidID.String()})

	selectSQL, selectArgs, err := selectQuery.ToSql()
	if err != nil {
		return 0, fmt.Errorf("TxRepo - IncrementChallengeSolveCountTx - BuildSelectQuery: %w", err)
	}

	var solveCount int
	err = tx.QueryRowContext(ctx, selectSQL, selectArgs...).Scan(&solveCount)
	if err != nil {
		return 0, fmt.Errorf("TxRepo - IncrementChallengeSolveCountTx - Query: %w", err)
	}

	return solveCount, nil
}

func (r *TxRepo) UpdateChallengePointsTx(ctx context.Context, tx *sql.Tx, id string, points int) error {
	uuidID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("TxRepo - UpdateChallengePointsTx - ParseID: %w", err)
	}

	query := squirrel.Update("challenges").
		Set("points", points).
		Where(squirrel.Eq{"id": uuidID.String()})

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("TxRepo - UpdateChallengePointsTx - BuildQuery: %w", err)
	}

	_, err = tx.ExecContext(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("TxRepo - UpdateChallengePointsTx - Exec: %w", err)
	}

	return nil
}

// SolveRepo Tx Methods

func (r *TxRepo) CreateSolveTx(ctx context.Context, tx *sql.Tx, s *entity.Solve) error {
	userID, err := uuid.Parse(s.UserId)
	if err != nil {
		return fmt.Errorf("TxRepo - CreateSolveTx - Parse UserID: %w", err)
	}
	teamID, err := uuid.Parse(s.TeamId)
	if err != nil {
		return fmt.Errorf("TxRepo - CreateSolveTx - Parse TeamID: %w", err)
	}
	challengeID, err := uuid.Parse(s.ChallengeId)
	if err != nil {
		return fmt.Errorf("TxRepo - CreateSolveTx - Parse ChallengeID: %w", err)
	}

	query := squirrel.Insert("solves").
		Columns("id", "user_id", "team_id", "challenge_id", "solved_at").
		Values(uuid.New().String(), userID, teamID, challengeID, time.Now())

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("TxRepo - CreateSolveTx - BuildQuery: %w", err)
	}

	_, err = tx.ExecContext(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("TxRepo - CreateSolveTx - ExecQuery: %w", err)
	}

	return nil
}

func (r *TxRepo) GetSolveByTeamAndChallengeTx(ctx context.Context, tx *sql.Tx, teamId, challengeId string) (*entity.Solve, error) {
	teamUUID, err := uuid.Parse(teamId)
	if err != nil {
		return nil, fmt.Errorf("TxRepo - GetSolveByTeamAndChallengeTx - Parse TeamID: %w", err)
	}
	challengeUUID, err := uuid.Parse(challengeId)
	if err != nil {
		return nil, fmt.Errorf("TxRepo - GetSolveByTeamAndChallengeTx - Parse ChallengeID: %w", err)
	}

	query := squirrel.Select("id", "user_id", "team_id", "challenge_id", "solved_at").
		From("solves").
		Where(squirrel.Eq{"team_id": teamUUID, "challenge_id": challengeUUID}).
		Suffix("FOR UPDATE")

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("TxRepo - GetSolveByTeamAndChallengeTx - BuildQuery: %w", err)
	}

	var solve entity.Solve
	err = tx.QueryRowContext(ctx, sqlQuery, args...).Scan(
		&solve.Id,
		&solve.UserId,
		&solve.TeamId,
		&solve.ChallengeId,
		&solve.SolvedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, entityError.ErrSolveNotFound
		}
		return nil, fmt.Errorf("TxRepo - GetSolveByTeamAndChallengeTx - Scan: %w", err)
	}

	return &solve, nil
}

func (r *TxRepo) GetTeamScoreTx(ctx context.Context, tx *sql.Tx, teamId string) (int, error) {
	teamUUID, err := uuid.Parse(teamId)
	if err != nil {
		return 0, fmt.Errorf("TxRepo - GetTeamScoreTx - Parse TeamID: %w", err)
	}

	solvePointsSubquery := squirrel.Select("SUM(c.points) as points").
		From("solves s").
		Join("challenges c ON c.id = s.challenge_id").
		Where(squirrel.Eq{"s.team_id": teamUUID.String()})

	awardPointsSubquery := squirrel.Select("SUM(value) as total").
		From("awards").
		Where(squirrel.Eq{"team_id": teamUUID.String()})

	solveSQL, solveArgs, err := solvePointsSubquery.ToSql()
	if err != nil {
		return 0, fmt.Errorf("TxRepo - GetTeamScoreTx - BuildSolveSubquery: %w", err)
	}

	awardSQL, awardArgs, err := awardPointsSubquery.ToSql()
	if err != nil {
		return 0, fmt.Errorf("TxRepo - GetTeamScoreTx - BuildAwardSubquery: %w", err)
	}

	query := squirrel.Select("COALESCE(solve_points.points, 0) + COALESCE(award_points.total, 0) as total_points").
		From("(SELECT 1) dummy").
		LeftJoin(fmt.Sprintf("(%s) solve_points ON 1=1", solveSQL), solveArgs...).
		LeftJoin(fmt.Sprintf("(%s) award_points ON 1=1", awardSQL), awardArgs...)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return 0, fmt.Errorf("TxRepo - GetTeamScoreTx - BuildQuery: %w", err)
	}

	var points int
	err = tx.QueryRowContext(ctx, sqlQuery, args...).Scan(&points)
	if err != nil {
		return 0, fmt.Errorf("TxRepo - GetTeamScoreTx - Scan: %w", err)
	}

	return points, nil
}

// HintUnlockRepo Tx Methods

func (r *TxRepo) CreateHintUnlockTx(ctx context.Context, tx *sql.Tx, teamId, hintId string) error {
	teamUUID, err := uuid.Parse(teamId)
	if err != nil {
		return fmt.Errorf("TxRepo - CreateHintUnlockTx - Parse TeamID: %w", err)
	}
	hintUUID, err := uuid.Parse(hintId)
	if err != nil {
		return fmt.Errorf("TxRepo - CreateHintUnlockTx - Parse HintID: %w", err)
	}

	query := squirrel.Insert("hint_unlocks").
		Columns("id", "hint_id", "team_id").
		Values(uuid.New().String(), hintUUID.String(), teamUUID.String())

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("TxRepo - CreateHintUnlockTx - BuildQuery: %w", err)
	}

	_, err = tx.ExecContext(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("TxRepo - CreateHintUnlockTx - ExecQuery: %w", err)
	}

	return nil
}

func (r *TxRepo) GetHintUnlockByTeamAndHintTx(ctx context.Context, tx *sql.Tx, teamId, hintId string) (*entity.HintUnlock, error) {
	teamUUID, err := uuid.Parse(teamId)
	if err != nil {
		return nil, fmt.Errorf("TxRepo - GetHintUnlockByTeamAndHintTx - Parse TeamID: %w", err)
	}
	hintUUID, err := uuid.Parse(hintId)
	if err != nil {
		return nil, fmt.Errorf("TxRepo - GetHintUnlockByTeamAndHintTx - Parse HintID: %w", err)
	}

	query := squirrel.Select("id", "hint_id", "team_id", "unlocked_at").
		From("hint_unlocks").
		Where(squirrel.Eq{"team_id": teamUUID, "hint_id": hintUUID}).
		Suffix("FOR UPDATE")

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("TxRepo - GetHintUnlockByTeamAndHintTx - BuildQuery: %w", err)
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
		return nil, fmt.Errorf("TxRepo - GetHintUnlockByTeamAndHintTx - Scan: %w", err)
	}

	return &unlock, nil
}

// AwardRepo Tx Methods

func (r *TxRepo) CreateAwardTx(ctx context.Context, tx *sql.Tx, a *entity.Award) error {
	id := uuid.New().String()
	a.Id = id

	teamUUID, err := uuid.Parse(a.TeamId)
	if err != nil {
		return fmt.Errorf("TxRepo - CreateAwardTx - Parse TeamID: %w", err)
	}

	query := squirrel.Insert("awards").
		Columns("id", "team_id", "value", "description", "created_at").
		Values(id, teamUUID.String(), a.Value, a.Description, time.Now())

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("TxRepo - CreateAwardTx - BuildQuery: %w", err)
	}

	_, err = tx.ExecContext(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("TxRepo - CreateAwardTx - ExecQuery: %w", err)
	}

	return nil
}

func (r *TxRepo) LockTeamTx(ctx context.Context, tx *sql.Tx, teamId string) error {
	uuidID, err := uuid.Parse(teamId)
	if err != nil {
		return fmt.Errorf("TxRepo - LockTeamTx - Parse TeamID: %w", err)
	}

	query := squirrel.Select("id").
		From("teams").
		Where(squirrel.Eq{"id": uuidID}).
		Suffix("FOR UPDATE")

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("TxRepo - LockTeamTx - BuildQuery: %w", err)
	}

	var id string
	err = tx.QueryRowContext(ctx, sqlQuery, args...).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entityError.ErrTeamNotFound
		}
		return fmt.Errorf("TxRepo - LockTeamTx - Scan: %w", err)
	}

	return nil
}

var _ repo.TxRepository = (*TxRepo)(nil)
