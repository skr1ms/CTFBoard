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
	"github.com/skr1ms/CTFBoard/internal/repo/persistent/sqlc"
)

type TxRepo struct {
	pool *pgxpool.Pool
	q    *sqlc.Queries
}

func NewTxRepo(pool *pgxpool.Pool) *TxRepo {
	return &TxRepo{pool: pool, q: sqlc.New(pool)}
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

func (r *TxRepo) CreateUserTx(ctx context.Context, tx pgx.Tx, user *entity.User) error {
	user.CreatedAt = time.Now()
	isVerified := false
	id, err := r.q.WithTx(tx).CreateUserReturningID(ctx, sqlc.CreateUserReturningIDParams{
		Username:     user.Username,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		Role:         &user.Role,
		IsVerified:   &isVerified,
		CreatedAt:    &user.CreatedAt,
	})
	if err != nil {
		return fmt.Errorf("TxRepo - CreateUserTx: %w", err)
	}
	user.ID = id
	return nil
}

func (r *TxRepo) UpdateUserTeamIDTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID, teamID *uuid.UUID) error {
	_, err := r.q.WithTx(tx).UpdateUserTeamID(ctx, sqlc.UpdateUserTeamIDParams{
		ID:     userID,
		TeamID: teamID,
	})
	if err != nil {
		if isNoRows(err) {
			return entityError.ErrUserNotFound
		}
		return fmt.Errorf("TxRepo - UpdateUserTeamIDTx: %w", err)
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

func (r *TxRepo) CreateTeamTx(ctx context.Context, tx pgx.Tx, team *entity.Team) error {
	team.CreatedAt = time.Now()
	id, err := r.q.WithTx(tx).CreateTeamReturningID(ctx, sqlc.CreateTeamReturningIDParams{
		Name:          team.Name,
		InviteToken:   team.InviteToken,
		CaptainID:     team.CaptainID,
		IsSolo:        &team.IsSolo,
		IsAutoCreated: &team.IsAutoCreated,
		CreatedAt:     &team.CreatedAt,
	})
	if err != nil {
		return fmt.Errorf("TxRepo - CreateTeamTx: %w", err)
	}
	team.ID = id
	return nil
}

func (r *TxRepo) GetTeamByNameTx(ctx context.Context, tx pgx.Tx, name string) (*entity.Team, error) {
	row, err := r.q.WithTx(tx).GetTeamByName(ctx, name)
	if err != nil {
		if isNoRows(err) {
			return nil, entityError.ErrTeamNotFound
		}
		return nil, fmt.Errorf("TxRepo - GetTeamByNameTx: %w", err)
	}
	return toEntityTeamFromRow(row.ID, row.Name, row.InviteToken, row.CaptainID, row.IsSolo, row.IsAutoCreated, row.IsBanned, row.IsHidden, row.BannedAt, row.BannedReason, row.CreatedAt), nil
}

func (r *TxRepo) GetTeamByInviteTokenTx(ctx context.Context, tx pgx.Tx, inviteToken uuid.UUID) (*entity.Team, error) {
	row, err := r.q.WithTx(tx).GetTeamByInviteToken(ctx, inviteToken)
	if err != nil {
		if isNoRows(err) {
			return nil, entityError.ErrTeamNotFound
		}
		return nil, fmt.Errorf("TxRepo - GetTeamByInviteTokenTx: %w", err)
	}
	return toEntityTeamFromRow(row.ID, row.Name, row.InviteToken, row.CaptainID, row.IsSolo, row.IsAutoCreated, row.IsBanned, row.IsHidden, row.BannedAt, row.BannedReason, row.CreatedAt), nil
}

func (r *TxRepo) GetUsersByTeamIDTx(ctx context.Context, tx pgx.Tx, teamID uuid.UUID) ([]*entity.User, error) {
	rows, err := r.q.WithTx(tx).ListUsersByTeamID(ctx, &teamID)
	if err != nil {
		return nil, fmt.Errorf("TxRepo - GetUsersByTeamIDTx: %w", err)
	}
	out := make([]*entity.User, 0, len(rows))
	for _, u := range rows {
		out = append(out, toEntityUser(u))
	}
	return out, nil
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
	_, err := r.q.WithTx(tx).SoftDeleteTeam(ctx, teamID)
	if err != nil {
		if isNoRows(err) {
			return entityError.ErrTeamNotFound
		}
		return fmt.Errorf("TxRepo - SoftDeleteTeamTx: %w", err)
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

func (r *TxRepo) GetChallengeByIDTx(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*entity.Challenge, error) {
	row, err := r.q.WithTx(tx).GetChallengeByIDForUpdate(ctx, id)
	if err != nil {
		if isNoRows(err) {
			return nil, entityError.ErrChallengeNotFound
		}
		return nil, fmt.Errorf("TxRepo - GetChallengeByIDTx: %w", err)
	}
	return toEntityChallengeFromRow(row.ID, row.Title, row.Description, row.Category, row.Points, row.InitialValue, row.MinValue, row.Decay, row.SolveCount, row.FlagHash, row.IsHidden, row.IsRegex, row.IsCaseInsensitive, row.FlagRegex, row.FlagFormatRegex), nil
}

func (r *TxRepo) IncrementChallengeSolveCountTx(ctx context.Context, tx pgx.Tx, id uuid.UUID) (int, error) {
	n, err := r.q.WithTx(tx).IncrementChallengeSolveCount(ctx, id)
	if err != nil {
		return 0, fmt.Errorf("TxRepo - IncrementChallengeSolveCountTx: %w", err)
	}
	return int(n), nil
}

func (r *TxRepo) UpdateChallengePointsTx(ctx context.Context, tx pgx.Tx, id uuid.UUID, points int) error {
	pts, err := intToInt32Safe(points)
	if err != nil {
		return fmt.Errorf("TxRepo - UpdateChallengePointsTx: %w", err)
	}
	_, err = r.q.WithTx(tx).UpdateChallengePoints(ctx, sqlc.UpdateChallengePointsParams{ID: id, Points: &pts})
	if err != nil {
		if isNoRows(err) {
			return entityError.ErrChallengeNotFound
		}
		return fmt.Errorf("TxRepo - UpdateChallengePointsTx: %w", err)
	}
	return nil
}

func (r *TxRepo) DeleteChallengeTx(ctx context.Context, tx pgx.Tx, challengeID uuid.UUID) error {
	_, err := r.q.WithTx(tx).DeleteChallenge(ctx, challengeID)
	if err != nil {
		if isNoRows(err) {
			return entityError.ErrChallengeNotFound
		}
		return fmt.Errorf("TxRepo - DeleteChallengeTx: %w", err)
	}
	return nil
}

func (r *TxRepo) CreateSolveTx(ctx context.Context, tx pgx.Tx, s *entity.Solve) error {
	s.ID = uuid.New()
	s.SolvedAt = time.Now()
	err := r.q.WithTx(tx).CreateSolve(ctx, sqlc.CreateSolveParams{
		ID:          s.ID,
		UserID:      s.UserID,
		TeamID:      s.TeamID,
		ChallengeID: s.ChallengeID,
		SolvedAt:    &s.SolvedAt,
	})
	if err != nil {
		if isPgUniqueViolation(err) {
			return entityError.ErrAlreadySolved
		}
		return fmt.Errorf("TxRepo - CreateSolveTx: %w", err)
	}
	return nil
}

func (r *TxRepo) DeleteSolvesByTeamIDTx(ctx context.Context, tx pgx.Tx, teamID uuid.UUID) error {
	if err := r.q.WithTx(tx).DeleteSolvesByTeamID(ctx, teamID); err != nil {
		return fmt.Errorf("TxRepo - DeleteSolvesByTeamIDTx: %w", err)
	}
	return nil
}

func (r *TxRepo) GetSolveByTeamAndChallengeTx(ctx context.Context, tx pgx.Tx, teamID, challengeID uuid.UUID) (*entity.Solve, error) {
	s, err := r.q.WithTx(tx).GetSolveByTeamAndChallengeForUpdate(ctx, sqlc.GetSolveByTeamAndChallengeForUpdateParams{
		TeamID:      teamID,
		ChallengeID: challengeID,
	})
	if err != nil {
		if isNoRows(err) {
			return nil, entityError.ErrSolveNotFound
		}
		return nil, fmt.Errorf("TxRepo - GetSolveByTeamAndChallengeTx: %w", err)
	}
	return toEntitySolve(s), nil
}

func (r *TxRepo) GetTeamScoreTx(ctx context.Context, tx pgx.Tx, teamID uuid.UUID) (int, error) {
	total, err := r.q.WithTx(tx).GetTeamScore(ctx, teamID)
	if err != nil {
		return 0, fmt.Errorf("TxRepo - GetTeamScoreTx: %w", err)
	}
	return int(total), nil
}

// HintUnlockRepo Tx Methods

func (r *TxRepo) CreateHintUnlockTx(ctx context.Context, tx pgx.Tx, teamID, hintID uuid.UUID) error {
	err := r.q.WithTx(tx).CreateHintUnlock(ctx, sqlc.CreateHintUnlockParams{
		ID:     uuid.New(),
		HintID: hintID,
		TeamID: teamID,
	})
	if err != nil {
		return fmt.Errorf("TxRepo - CreateHintUnlockTx: %w", err)
	}
	return nil
}

func (r *TxRepo) GetHintUnlockByTeamAndHintTx(ctx context.Context, tx pgx.Tx, teamID, hintID uuid.UUID) (*entity.HintUnlock, error) {
	u, err := r.q.WithTx(tx).GetHintUnlockByTeamAndHintForUpdate(ctx, sqlc.GetHintUnlockByTeamAndHintForUpdateParams{
		TeamID: teamID,
		HintID: hintID,
	})
	if err != nil {
		if isNoRows(err) {
			return nil, entityError.ErrHintNotFound
		}
		return nil, fmt.Errorf("TxRepo - GetHintUnlockByTeamAndHintTx: %w", err)
	}
	return toEntityHintUnlock(u), nil
}

func (r *TxRepo) CreateAwardTx(ctx context.Context, tx pgx.Tx, a *entity.Award) error {
	a.ID = uuid.New()
	a.CreatedAt = time.Now()
	value, err := intToInt32Safe(a.Value)
	if err != nil {
		return fmt.Errorf("TxRepo - CreateAwardTx: %w", err)
	}
	err = r.q.WithTx(tx).CreateAward(ctx, sqlc.CreateAwardParams{
		ID:          a.ID,
		TeamID:      a.TeamID,
		Value:       value,
		Description: a.Description,
		CreatedBy:   a.CreatedBy,
		CreatedAt:   &a.CreatedAt,
	})
	if err != nil {
		return fmt.Errorf("TxRepo - CreateAwardTx: %w", err)
	}
	return nil
}

func (r *TxRepo) CreateAuditLogTx(ctx context.Context, tx pgx.Tx, log *entity.AuditLog) error {
	details, err := json.Marshal(log.Details)
	if err != nil {
		return fmt.Errorf("TxRepo - CreateAuditLogTx Marshal: %w", err)
	}
	row, err := r.q.WithTx(tx).CreateAuditLog(ctx, sqlc.CreateAuditLogParams{
		UserID:     log.UserID,
		Action:     string(log.Action),
		EntityType: string(log.EntityType),
		EntityID:   strPtrOrNil(log.EntityID),
		Ip:         strPtrOrNil(log.IP),
		Details:    details,
	})
	if err != nil {
		return fmt.Errorf("TxRepo - CreateAuditLogTx: %w", err)
	}
	log.ID = row.ID
	log.CreatedAt = ptrTimeToTime(row.CreatedAt)
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

func (r *TxRepo) GetTeamByIDTx(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*entity.Team, error) {
	row, err := r.q.WithTx(tx).GetTeamByID(ctx, id)
	if err != nil {
		if isNoRows(err) {
			return nil, entityError.ErrTeamNotFound
		}
		return nil, fmt.Errorf("TxRepo - GetTeamByIDTx: %w", err)
	}
	return toEntityTeamFromRow(row.ID, row.Name, row.InviteToken, row.CaptainID, row.IsSolo, row.IsAutoCreated, row.IsBanned, row.IsHidden, row.BannedAt, row.BannedReason, row.CreatedAt), nil
}

func (r *TxRepo) GetSoloTeamByUserIDTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID) (*entity.Team, error) {
	row, err := r.q.WithTx(tx).GetSoloTeamByUserID(ctx, userID)
	if err != nil {
		if isNoRows(err) {
			return nil, entityError.ErrTeamNotFound
		}
		return nil, fmt.Errorf("TxRepo - GetSoloTeamByUserIDTx: %w", err)
	}
	return toEntityTeamFromRow(row.ID, row.Name, row.InviteToken, row.CaptainID, row.IsSolo, row.IsAutoCreated, row.IsBanned, row.IsHidden, row.BannedAt, row.BannedReason, row.CreatedAt), nil
}

var _ repo.TxRepository = (*TxRepo)(nil)
