package persistent

import (
	"context"
	"encoding/json"
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
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				_ = fmt.Sprintf("rollback: %v", rbErr)
			}
			panic(p)
		}
	}()

	if err := fn(ctx, tx); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("RunTransaction - FnError: %w, RollbackError: %w", err, rbErr)
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

	err = tx.QueryRow(ctx, sqlQuery, args...).Scan(&user.ID)
	if err != nil {
		return fmt.Errorf("TxRepo - CreateUserTx - Exec: %w", err)
	}

	return nil
}

func (r *TxRepo) UpdateUserTeamIDTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID, teamID *uuid.UUID) error {
	updateBuilder := squirrel.Update("users").
		Where(squirrel.Eq{"id": userID}).
		PlaceholderFormat(squirrel.Dollar)

	if teamID != nil {
		updateBuilder = updateBuilder.Set("team_id", *teamID)
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

func (r *TxRepo) LockUserTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID) error {
	query := squirrel.Select("id").
		From("users").
		Where(squirrel.Eq{"id": userID}).
		Suffix("FOR UPDATE").
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("TxRepo - LockUserTx - BuildQuery: %w", err)
	}

	var ID uuid.UUID
	err = tx.QueryRow(ctx, sqlQuery, args...).Scan(&ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entityError.ErrUserNotFound
		}
		return fmt.Errorf("TxRepo - LockUserTx - Scan: %w", err)
	}

	return nil
}

// TeamRepo Tx Methods

func (r *TxRepo) CreateTeamTx(ctx context.Context, tx pgx.Tx, team *entity.Team) error {
	team.CreatedAt = time.Now()

	query := squirrel.Insert("teams").
		Columns("name", "invite_token", "captain_id", "is_solo", "is_auto_created", "created_at").
		Values(team.Name, team.InviteToken, team.CaptainID, team.IsSolo, team.IsAutoCreated, team.CreatedAt).
		Suffix("RETURNING id").
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("TxRepo - CreateTeamTx - BuildQuery: %w", err)
	}

	err = tx.QueryRow(ctx, sqlQuery, args...).Scan(&team.ID)
	if err != nil {
		return fmt.Errorf("TxRepo - CreateTeamTx - Exec: %w", err)
	}

	return nil
}

func (r *TxRepo) GetTeamByNameTx(ctx context.Context, tx pgx.Tx, name string) (*entity.Team, error) {
	query := squirrel.Select("id", "name", "invite_token", "captain_id", "is_solo", "is_auto_created", "created_at").
		From("teams").
		Where(squirrel.Eq{"name": name}).
		Where(squirrel.Eq{"deleted_at": nil}).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("TxRepo - GetTeamByNameTx - BuildQuery: %w", err)
	}

	var team entity.Team
	err = tx.QueryRow(ctx, sqlQuery, args...).Scan(
		&team.ID,
		&team.Name,
		&team.InviteToken,
		&team.CaptainID,
		&team.IsSolo,
		&team.IsAutoCreated,
		&team.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entityError.ErrTeamNotFound
		}
		return nil, fmt.Errorf("TxRepo - GetTeamByNameTx - Scan: %w", err)
	}

	return &team, nil
}

func (r *TxRepo) GetTeamByInviteTokenTx(ctx context.Context, tx pgx.Tx, inviteToken uuid.UUID) (*entity.Team, error) {
	query := squirrel.Select("id", "name", "invite_token", "captain_id", "is_solo", "is_auto_created", "created_at").
		From("teams").
		Where(squirrel.Eq{"invite_token": inviteToken}).
		Where(squirrel.Eq{"deleted_at": nil}).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("TxRepo - GetTeamByInviteTokenTx - BuildQuery: %w", err)
	}

	var team entity.Team
	err = tx.QueryRow(ctx, sqlQuery, args...).Scan(
		&team.ID,
		&team.Name,
		&team.InviteToken,
		&team.CaptainID,
		&team.IsSolo,
		&team.IsAutoCreated,
		&team.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entityError.ErrTeamNotFound
		}
		return nil, fmt.Errorf("TxRepo - GetTeamByInviteTokenTx - Scan: %w", err)
	}

	return &team, nil
}

func (r *TxRepo) GetUsersByTeamIDTx(ctx context.Context, tx pgx.Tx, teamID uuid.UUID) ([]*entity.User, error) {
	query := squirrel.Select("id", "team_id", "username", "email", "password_hash", "role", "is_verified", "verified_at", "created_at").
		From("users").
		Where(squirrel.Eq{"team_id": teamID}).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("TxRepo - GetUsersByTeamIDTx - BuildQuery: %w", err)
	}

	rows, err := tx.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("TxRepo - GetUsersByTeamIDTx - Query: %w", err)
	}
	defer rows.Close()

	var users []*entity.User
	for rows.Next() {
		var user entity.User
		err := rows.Scan(
			&user.ID,
			&user.TeamID,
			&user.Username,
			&user.Email,
			&user.PasswordHash,
			&user.Role,
			&user.IsVerified,
			&user.VerifiedAt,
			&user.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("TxRepo - GetUsersByTeamIDTx - Scan: %w", err)
		}
		users = append(users, &user)
	}

	return users, nil
}

func (r *TxRepo) DeleteTeamTx(ctx context.Context, tx pgx.Tx, teamID uuid.UUID) error {
	query := squirrel.Update("teams").
		Set("deleted_at", time.Now()).
		Where(squirrel.Eq{"id": teamID}).
		Where(squirrel.Eq{"deleted_at": nil}).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("TxRepo - DeleteTeamTx - BuildQuery: %w", err)
	}

	cmdTag, err := tx.Exec(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("TxRepo - DeleteTeamTx - Exec: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return entityError.ErrTeamNotFound
	}

	return nil
}

func (r *TxRepo) UpdateTeamCaptainTx(ctx context.Context, tx pgx.Tx, teamID, newCaptainID uuid.UUID) error {
	query := squirrel.Update("teams").
		Set("captain_id", newCaptainID).
		Where(squirrel.Eq{"id": teamID}).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("TxRepo - UpdateTeamCaptainTx - BuildQuery: %w", err)
	}

	cmdTag, err := tx.Exec(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("TxRepo - UpdateTeamCaptainTx - Exec: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return entityError.ErrTeamNotFound
	}

	return nil
}

func (r *TxRepo) SoftDeleteTeamTx(ctx context.Context, tx pgx.Tx, teamID uuid.UUID) error {
	query := squirrel.Update("teams").
		Set("deleted_at", time.Now()).
		Where(squirrel.Eq{"id": teamID}).
		Where(squirrel.Eq{"deleted_at": nil}).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("TxRepo - SoftDeleteTeamTx - BuildQuery: %w", err)
	}

	cmdTag, err := tx.Exec(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("TxRepo - SoftDeleteTeamTx - Exec: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return entityError.ErrTeamNotFound
	}

	return nil
}

func (r *TxRepo) CreateTeamAuditLogTx(ctx context.Context, tx pgx.Tx, log *entity.TeamAuditLog) error {
	log.ID = uuid.New()
	log.CreatedAt = time.Now()

	detailsJSON := []byte("{}")
	if log.Details != nil {
		var err error
		detailsJSON, err = json.Marshal(log.Details)
		if err != nil {
			return fmt.Errorf("TxRepo - CreateTeamAuditLogTx - MarshalDetails: %w", err)
		}
	}

	query := squirrel.Insert("team_audit_log").
		Columns("id", "team_id", "user_id", "action", "details", "created_at").
		Values(log.ID, log.TeamID, log.UserID, log.Action, detailsJSON, log.CreatedAt).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("TxRepo - CreateTeamAuditLogTx - BuildQuery: %w", err)
	}

	_, err = tx.Exec(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("TxRepo - CreateTeamAuditLogTx - Exec: %w", err)
	}

	return nil
}

// ChallengeRepo Tx Methods

func (r *TxRepo) GetChallengeByIDTx(ctx context.Context, tx pgx.Tx, ID uuid.UUID) (*entity.Challenge, error) {
	query := squirrel.Select("id", "title", "description", "category", "points", "initial_value", "min_value", "decay", "solve_count", "flag_hash", "is_hidden", "is_regex", "is_case_insensitive", "flag_regex").
		From("challenges").
		Where(squirrel.Eq{"id": ID}).
		Suffix("FOR UPDATE").
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("TxRepo - GetChallengeByIDTx - BuildQuery: %w", err)
	}

	var challenge entity.Challenge
	err = tx.QueryRow(ctx, sqlQuery, args...).Scan(
		&challenge.ID,
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
		&challenge.IsRegex,
		&challenge.IsCaseInsensitive,
		&challenge.FlagRegex,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entityError.ErrChallengeNotFound
		}
		return nil, fmt.Errorf("TxRepo - GetChallengeByIDTx - Scan: %w", err)
	}

	return &challenge, nil
}

func (r *TxRepo) IncrementChallengeSolveCountTx(ctx context.Context, tx pgx.Tx, ID uuid.UUID) (int, error) {
	query := squirrel.Update("challenges").
		Set("solve_count", squirrel.Expr("solve_count + 1")).
		Where(squirrel.Eq{"id": ID}).
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

func (r *TxRepo) UpdateChallengePointsTx(ctx context.Context, tx pgx.Tx, ID uuid.UUID, points int) error {
	query := squirrel.Update("challenges").
		Set("points", points).
		Where(squirrel.Eq{"id": ID}).
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
	s.ID = uuid.New()
	s.SolvedAt = time.Now()

	query := squirrel.Insert("solves").
		Columns("id", "user_id", "team_id", "challenge_id", "solved_at").
		Values(s.ID, s.UserID, s.TeamID, s.ChallengeID, s.SolvedAt).
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

func (r *TxRepo) DeleteSolvesByTeamIDTx(ctx context.Context, tx pgx.Tx, teamID uuid.UUID) error {
	query := squirrel.Delete("solves").
		Where(squirrel.Eq{"team_id": teamID}).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("TxRepo - DeleteSolvesByTeamIDTx - BuildQuery: %w", err)
	}

	if _, err := tx.Exec(ctx, sqlQuery, args...); err != nil {
		return fmt.Errorf("TxRepo - DeleteSolvesByTeamIDTx - Exec: %w", err)
	}

	return nil
}

func (r *TxRepo) GetSolveByTeamAndChallengeTx(ctx context.Context, tx pgx.Tx, teamID, challengeID uuid.UUID) (*entity.Solve, error) {
	query := squirrel.Select("id", "user_id", "team_id", "challenge_id", "solved_at").
		From("solves").
		Where(squirrel.Eq{"team_id": teamID, "challenge_id": challengeID}).
		Suffix("FOR UPDATE").
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("TxRepo - GetSolveByTeamAndChallengeTx - BuildQuery: %w", err)
	}

	var solve entity.Solve
	err = tx.QueryRow(ctx, sqlQuery, args...).Scan(
		&solve.ID,
		&solve.UserID,
		&solve.TeamID,
		&solve.ChallengeID,
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

func (r *TxRepo) GetTeamScoreTx(ctx context.Context, tx pgx.Tx, teamID uuid.UUID) (int, error) {
	query := squirrel.Select(
		"COALESCE(SUM(c.points), 0) + COALESCE((SELECT SUM(value) FROM awards WHERE team_ID = $1), 0) as total_points",
	).
		From("solves s").
		RightJoin("(SELECT 1) dummy ON true").
		LeftJoin("challenges c ON c.ID = s.challenge_ID AND s.team_ID = $1").
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, _, err := query.ToSql()
	if err != nil {
		return 0, fmt.Errorf("TxRepo - GetTeamScoreTx - BuildQuery: %w", err)
	}

	var points int
	err = tx.QueryRow(ctx, sqlQuery, teamID).Scan(&points)
	if err != nil {
		return 0, fmt.Errorf("TxRepo - GetTeamScoreTx - Scan: %w", err)
	}

	return points, nil
}

// HintUnlockRepo Tx Methods

func (r *TxRepo) CreateHintUnlockTx(ctx context.Context, tx pgx.Tx, teamID, hintID uuid.UUID) error {
	query := squirrel.Insert("hint_unlocks").
		Columns("id", "hint_id", "team_id").
		Values(uuid.New(), hintID, teamID).
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

func (r *TxRepo) GetHintUnlockByTeamAndHintTx(ctx context.Context, tx pgx.Tx, teamID, hintID uuid.UUID) (*entity.HintUnlock, error) {
	query := squirrel.Select("id", "hint_id", "team_id", "unlocked_at").
		From("hint_unlocks").
		Where(squirrel.Eq{"team_id": teamID, "hint_id": hintID}).
		Suffix("FOR UPDATE").
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("TxRepo - GetHintUnlockByTeamAndHintTx - BuildQuery: %w", err)
	}

	var unlock entity.HintUnlock
	err = tx.QueryRow(ctx, sqlQuery, args...).Scan(
		&unlock.ID,
		&unlock.HintID,
		&unlock.TeamID,
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
	a.ID = uuid.New()
	a.CreatedAt = time.Now()

	query := squirrel.Insert("awards").
		Columns("id", "team_id", "value", "description", "created_by", "created_at").
		Values(a.ID, a.TeamID, a.Value, a.Description, a.CreatedBy, a.CreatedAt).
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

func (r *TxRepo) CreateAuditLogTx(ctx context.Context, tx pgx.Tx, log *entity.AuditLog) error {
	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now()
	}
	detailsJSON, err := json.Marshal(log.Details)
	if err != nil {
		return fmt.Errorf("TxRepo - CreateAuditLogTx - MarshalDetails: %w", err)
	}

	var details string
	if log.Details == nil {
		details = "{}"
	} else {
		details = string(detailsJSON)
	}

	query := squirrel.Insert("audit_logs").
		Columns("id", "user_id", "action", "entity_type", "entity_id", "ip", "details", "created_at").
		Values(log.ID, log.UserID, log.Action, log.EntityType, log.EntityID, log.IP, details, log.CreatedAt).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("TxRepo - CreateAuditLogTx - BuildQuery: %w", err)
	}

	if _, err := tx.Exec(ctx, sqlQuery, args...); err != nil {
		return fmt.Errorf("TxRepo - CreateAuditLogTx - Exec: %w", err)
	}

	return nil
}

func (r *TxRepo) LockTeamTx(ctx context.Context, tx pgx.Tx, teamID uuid.UUID) error {
	query := squirrel.Select("id").
		From("teams").
		Where(squirrel.Eq{"id": teamID}).
		Suffix("FOR UPDATE").
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("TxRepo - LockTeamTx - BuildQuery: %w", err)
	}

	var ID uuid.UUID
	err = tx.QueryRow(ctx, sqlQuery, args...).Scan(&ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entityError.ErrTeamNotFound
		}
		return fmt.Errorf("TxRepo - LockTeamTx - Scan: %w", err)
	}

	return nil
}

func (r *TxRepo) GetTeamByIDTx(ctx context.Context, tx pgx.Tx, ID uuid.UUID) (*entity.Team, error) {
	query := squirrel.Select("id", "name", "invite_token", "captain_id", "is_solo", "is_auto_created", "created_at").
		From("teams").
		Where(squirrel.Eq{"id": ID}).
		Where(squirrel.Eq{"deleted_at": nil}).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("TxRepo - GetTeamByIDTx - BuildQuery: %w", err)
	}

	var team entity.Team
	err = tx.QueryRow(ctx, sqlQuery, args...).Scan(
		&team.ID,
		&team.Name,
		&team.InviteToken,
		&team.CaptainID,
		&team.IsSolo,
		&team.IsAutoCreated,
		&team.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entityError.ErrTeamNotFound
		}
		return nil, fmt.Errorf("TxRepo - GetTeamByIDTx - Scan: %w", err)
	}

	return &team, nil
}

func (r *TxRepo) GetSoloTeamByUserIDTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID) (*entity.Team, error) {
	query := squirrel.Select("t.id", "t.name", "t.invite_token", "t.captain_id", "t.is_solo", "t.is_auto_created", "t.created_at").
		From("teams t").
		Join("users u ON u.team_id = t.id").
		Where(squirrel.Eq{"u.id": userID}).
		Where(squirrel.Eq{"t.is_solo": true}).
		Where(squirrel.Eq{"t.deleted_at": nil}).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("TxRepo - GetSoloTeamByUserIDTx - BuildQuery: %w", err)
	}

	var team entity.Team
	err = tx.QueryRow(ctx, sqlQuery, args...).Scan(
		&team.ID,
		&team.Name,
		&team.InviteToken,
		&team.CaptainID,
		&team.IsSolo,
		&team.IsAutoCreated,
		&team.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entityError.ErrTeamNotFound
		}
		return nil, fmt.Errorf("TxRepo - GetSoloTeamByUserIDTx - Scan: %w", err)
	}

	return &team, nil
}

var _ repo.TxRepository = (*TxRepo)(nil)
