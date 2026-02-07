package persistent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/repo"
	"github.com/skr1ms/CTFBoard/internal/repo/persistent/sqlc"
)

type TxTeamRepo struct {
	base *TxBase
}

func (r *TxTeamRepo) CreateTeamTx(ctx context.Context, tx repo.Transaction, team *entity.Team) error {
	pgxTx := mustPgxTx(tx)
	team.CreatedAt = time.Now()
	id, err := r.base.q.WithTx(pgxTx).CreateTeamReturningID(ctx, sqlc.CreateTeamReturningIDParams{
		Name:          team.Name,
		InviteToken:   team.InviteToken,
		CaptainID:     team.CaptainID,
		IsSolo:        &team.IsSolo,
		IsAutoCreated: &team.IsAutoCreated,
		CreatedAt:     &team.CreatedAt,
	})
	if err != nil {
		return fmt.Errorf("TxTeamRepo - CreateTeamTx: %w", err)
	}
	team.ID = id
	return nil
}

func (r *TxTeamRepo) GetTeamByNameTx(ctx context.Context, tx repo.Transaction, name string) (*entity.Team, error) {
	pgxTx := mustPgxTx(tx)
	row, err := r.base.q.WithTx(pgxTx).GetTeamByName(ctx, name)
	if err != nil {
		if isNoRows(err) {
			return nil, entityError.ErrTeamNotFound
		}
		return nil, fmt.Errorf("TxTeamRepo - GetTeamByNameTx: %w", err)
	}
	return toEntityTeamFromRow(row.ID, row.Name, row.InviteToken, row.CaptainID, row.BracketID, row.IsSolo, row.IsAutoCreated, row.IsBanned, row.IsHidden, row.BannedAt, row.BannedReason, row.CreatedAt), nil
}

func (r *TxTeamRepo) GetTeamByInviteTokenTx(ctx context.Context, tx repo.Transaction, inviteToken uuid.UUID) (*entity.Team, error) {
	pgxTx := mustPgxTx(tx)
	row, err := r.base.q.WithTx(pgxTx).GetTeamByInviteToken(ctx, inviteToken)
	if err != nil {
		if isNoRows(err) {
			return nil, entityError.ErrTeamNotFound
		}
		return nil, fmt.Errorf("TxTeamRepo - GetTeamByInviteTokenTx: %w", err)
	}
	return toEntityTeamFromRow(row.ID, row.Name, row.InviteToken, row.CaptainID, row.BracketID, row.IsSolo, row.IsAutoCreated, row.IsBanned, row.IsHidden, row.BannedAt, row.BannedReason, row.CreatedAt), nil
}

func (r *TxTeamRepo) GetUsersByTeamIDTx(ctx context.Context, tx repo.Transaction, teamID uuid.UUID) ([]*entity.User, error) {
	pgxTx := mustPgxTx(tx)
	rows, err := r.base.q.WithTx(pgxTx).ListUsersByTeamID(ctx, &teamID)
	if err != nil {
		return nil, fmt.Errorf("TxTeamRepo - GetUsersByTeamIDTx: %w", err)
	}
	out := make([]*entity.User, 0, len(rows))
	for _, u := range rows {
		out = append(out, toEntityUser(u))
	}
	return out, nil
}

func (r *TxTeamRepo) DeleteTeamTx(ctx context.Context, tx repo.Transaction, teamID uuid.UUID) error {
	pgxTx := mustPgxTx(tx)
	query := squirrel.Update("teams").
		Set("deleted_at", time.Now()).
		Where(squirrel.Eq{"id": teamID}).
		Where(squirrel.Eq{"deleted_at": nil}).
		PlaceholderFormat(squirrel.Dollar)
	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("TxTeamRepo - DeleteTeamTx - BuildQuery: %w", err)
	}
	cmdTag, err := pgxTx.Exec(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("TxTeamRepo - DeleteTeamTx - Exec: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return entityError.ErrTeamNotFound
	}
	return nil
}

func (r *TxTeamRepo) UpdateTeamCaptainTx(ctx context.Context, tx repo.Transaction, teamID, newCaptainID uuid.UUID) error {
	pgxTx := mustPgxTx(tx)
	query := squirrel.Update("teams").
		Set("captain_id", newCaptainID).
		Where(squirrel.Eq{"id": teamID}).
		PlaceholderFormat(squirrel.Dollar)
	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("TxTeamRepo - UpdateTeamCaptainTx - BuildQuery: %w", err)
	}
	cmdTag, err := pgxTx.Exec(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("TxTeamRepo - UpdateTeamCaptainTx - Exec: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return entityError.ErrTeamNotFound
	}
	return nil
}

func (r *TxTeamRepo) SoftDeleteTeamTx(ctx context.Context, tx repo.Transaction, teamID uuid.UUID) error {
	pgxTx := mustPgxTx(tx)
	_, err := r.base.q.WithTx(pgxTx).SoftDeleteTeam(ctx, teamID)
	if err != nil {
		if isNoRows(err) {
			return entityError.ErrTeamNotFound
		}
		return fmt.Errorf("TxTeamRepo - SoftDeleteTeamTx: %w", err)
	}
	return nil
}

func (r *TxTeamRepo) CreateTeamAuditLogTx(ctx context.Context, tx repo.Transaction, log *entity.TeamAuditLog) error {
	pgxTx := mustPgxTx(tx)
	log.ID = uuid.New()
	log.CreatedAt = time.Now()
	detailsJSON := []byte("{}")
	if log.Details != nil {
		var err error
		detailsJSON, err = json.Marshal(log.Details)
		if err != nil {
			return fmt.Errorf("TxTeamRepo - CreateTeamAuditLogTx - MarshalDetails: %w", err)
		}
	}
	query := squirrel.Insert("team_audit_log").
		Columns("id", "team_id", "user_id", "action", "details", "created_at").
		Values(log.ID, log.TeamID, log.UserID, log.Action, detailsJSON, log.CreatedAt).
		PlaceholderFormat(squirrel.Dollar)
	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("TxTeamRepo - CreateTeamAuditLogTx - BuildQuery: %w", err)
	}
	_, err = pgxTx.Exec(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("TxTeamRepo - CreateTeamAuditLogTx - Exec: %w", err)
	}
	return nil
}

func (r *TxTeamRepo) LockTeamTx(ctx context.Context, tx repo.Transaction, teamID uuid.UUID) error {
	pgxTx := mustPgxTx(tx)
	query := squirrel.Select("id").
		From("teams").
		Where(squirrel.Eq{"id": teamID}).
		Suffix("FOR UPDATE").
		PlaceholderFormat(squirrel.Dollar)
	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("TxTeamRepo - LockTeamTx - BuildQuery: %w", err)
	}
	var id uuid.UUID
	err = pgxTx.QueryRow(ctx, sqlQuery, args...).Scan(&id)
	if err != nil {
		if isNoRows(err) {
			return entityError.ErrTeamNotFound
		}
		return fmt.Errorf("TxTeamRepo - LockTeamTx - Scan: %w", err)
	}
	return nil
}

func (r *TxTeamRepo) GetTeamByIDTx(ctx context.Context, tx repo.Transaction, id uuid.UUID) (*entity.Team, error) {
	pgxTx := mustPgxTx(tx)
	row, err := r.base.q.WithTx(pgxTx).GetTeamByID(ctx, id)
	if err != nil {
		if isNoRows(err) {
			return nil, entityError.ErrTeamNotFound
		}
		return nil, fmt.Errorf("TxTeamRepo - GetTeamByIDTx: %w", err)
	}
	return toEntityTeamFromRow(row.ID, row.Name, row.InviteToken, row.CaptainID, row.BracketID, row.IsSolo, row.IsAutoCreated, row.IsBanned, row.IsHidden, row.BannedAt, row.BannedReason, row.CreatedAt), nil
}

func (r *TxTeamRepo) GetSoloTeamByUserIDTx(ctx context.Context, tx repo.Transaction, userID uuid.UUID) (*entity.Team, error) {
	pgxTx := mustPgxTx(tx)
	row, err := r.base.q.WithTx(pgxTx).GetSoloTeamByUserID(ctx, userID)
	if err != nil {
		if isNoRows(err) {
			return nil, entityError.ErrTeamNotFound
		}
		return nil, fmt.Errorf("TxTeamRepo - GetSoloTeamByUserIDTx: %w", err)
	}
	return toEntityTeamFromRow(row.ID, row.Name, row.InviteToken, row.CaptainID, row.BracketID, row.IsSolo, row.IsAutoCreated, row.IsBanned, row.IsHidden, row.BannedAt, row.BannedReason, row.CreatedAt), nil
}
