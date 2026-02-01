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
)

var (
	teamInsertColumns = []string{"id", "name", "invite_token", "captain_id", "is_solo", "is_auto_created", "created_at"}
	teamSelectColumns = []string{"id", "name", "invite_token", "captain_id", "is_solo", "is_auto_created", "is_banned", "banned_at", "banned_reason", "is_hidden", "created_at"}
)

func scanTeam(row rowScanner) (*entity.Team, error) {
	var t entity.Team
	err := row.Scan(
		&t.ID, &t.Name, &t.InviteToken, &t.CaptainID, &t.IsSolo, &t.IsAutoCreated,
		&t.IsBanned, &t.BannedAt, &t.BannedReason, &t.IsHidden, &t.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func teamSelectColumnsT() []string {
	out := make([]string, len(teamSelectColumns))
	for i, col := range teamSelectColumns {
		out[i] = "t." + col
	}
	return out
}

type TeamRepo struct {
	pool *pgxpool.Pool
}

func NewTeamRepo(pool *pgxpool.Pool) *TeamRepo {
	return &TeamRepo{pool: pool}
}

func (r *TeamRepo) Create(ctx context.Context, t *entity.Team) error {
	t.ID = uuid.New()
	t.CreatedAt = time.Now()

	query := squirrel.Insert("teams").
		Columns(teamInsertColumns...).
		Values(t.ID, t.Name, t.InviteToken, t.CaptainID, t.IsSolo, t.IsAutoCreated, t.CreatedAt).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("TeamRepo - Create - BuildQuery: %w", err)
	}

	_, err = r.pool.Exec(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("TeamRepo - Create - ExecQuery: %w", err)
	}

	return nil
}

func (r *TeamRepo) GetByID(ctx context.Context, ID uuid.UUID) (*entity.Team, error) {
	query := squirrel.Select(teamSelectColumns...).
		From("teams").
		Where(squirrel.Eq{"id": ID}).
		Where(squirrel.Eq{"deleted_at": nil}).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("TeamRepo - GetByID - BuildQuery: %w", err)
	}

	team, err := scanTeam(r.pool.QueryRow(ctx, sqlQuery, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entityError.ErrTeamNotFound
		}
		return nil, fmt.Errorf("TeamRepo - GetByID - Scan: %w", err)
	}
	return team, nil
}

func (r *TeamRepo) GetByInviteToken(ctx context.Context, inviteToken uuid.UUID) (*entity.Team, error) {
	if inviteToken == uuid.Nil {
		return nil, entityError.ErrTeamNotFound
	}

	query := squirrel.Select(teamSelectColumns...).
		From("teams").
		Where(squirrel.Eq{"invite_token": inviteToken}).
		Where(squirrel.Eq{"deleted_at": nil}).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("TeamRepo - GetByInviteToken - BuildQuery: %w", err)
	}

	team, err := scanTeam(r.pool.QueryRow(ctx, sqlQuery, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entityError.ErrTeamNotFound
		}
		return nil, fmt.Errorf("TeamRepo - GetByInviteToken - Scan: %w", err)
	}
	return team, nil
}

func (r *TeamRepo) GetByName(ctx context.Context, name string) (*entity.Team, error) {
	query := squirrel.Select(teamSelectColumns...).
		From("teams").
		Where(squirrel.Eq{"name": name}).
		Where(squirrel.Eq{"deleted_at": nil}).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("TeamRepo - GetByName - BuildQuery: %w", err)
	}

	team, err := scanTeam(r.pool.QueryRow(ctx, sqlQuery, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entityError.ErrTeamNotFound
		}
		return nil, fmt.Errorf("TeamRepo - GetByName - Scan: %w", err)
	}
	return team, nil
}

func (r *TeamRepo) Delete(ctx context.Context, ID uuid.UUID) error {
	query := squirrel.Update("teams").
		Set("deleted_at", time.Now()).
		Where(squirrel.Eq{"id": ID}).
		Where(squirrel.Eq{"deleted_at": nil}).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("TeamRepo - Delete - BuildQuery: %w", err)
	}

	_, err = r.pool.Exec(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("TeamRepo - Delete - ExecQuery: %w", err)
	}

	return nil
}

func (r *TeamRepo) HardDeleteTeams(ctx context.Context, cutoffDate time.Time) error {
	query := squirrel.Delete("teams").
		Where(squirrel.NotEq{"deleted_at": nil}).
		Where(squirrel.Lt{"deleted_at": cutoffDate}).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("TeamRepo - HardDeleteTeams - BuildQuery: %w", err)
	}

	_, err = r.pool.Exec(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("TeamRepo - HardDeleteTeams - ExecQuery: %w", err)
	}

	return nil
}

func (r *TeamRepo) GetSoloTeamByUserID(ctx context.Context, userID uuid.UUID) (*entity.Team, error) {
	query := squirrel.Select(teamSelectColumnsT()...).
		From("teams t").
		Join("users u ON u.team_id = t.id").
		Where(squirrel.Eq{"u.id": userID}).
		Where(squirrel.Eq{"t.is_solo": true}).
		Where(squirrel.Eq{"t.deleted_at": nil}).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("TeamRepo - GetSoloTeamByUserID - BuildQuery: %w", err)
	}

	team, err := scanTeam(r.pool.QueryRow(ctx, sqlQuery, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entityError.ErrTeamNotFound
		}
		return nil, fmt.Errorf("TeamRepo - GetSoloTeamByUserID - Scan: %w", err)
	}
	return team, nil
}

func (r *TeamRepo) CountTeamMembers(ctx context.Context, teamID uuid.UUID) (int, error) {
	query := squirrel.Select("COUNT(*)").
		From("users").
		Where(squirrel.Eq{"team_id": teamID}).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return 0, fmt.Errorf("TeamRepo - CountTeamMembers - BuildQuery: %w", err)
	}

	var count int
	err = r.pool.QueryRow(ctx, sqlQuery, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("TeamRepo - CountTeamMembers - Scan: %w", err)
	}

	return count, nil
}

func (r *TeamRepo) Ban(ctx context.Context, teamID uuid.UUID, reason string) error {
	now := time.Now()
	query := squirrel.Update("teams").
		Set("is_banned", true).
		Set("banned_at", now).
		Set("banned_reason", reason).
		Where(squirrel.Eq{"id": teamID}).
		Where(squirrel.Eq{"deleted_at": nil}).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("TeamRepo - Ban - BuildQuery: %w", err)
	}

	cmdTag, err := r.pool.Exec(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("TeamRepo - Ban - ExecQuery: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return entityError.ErrTeamNotFound
	}
	return nil
}

func (r *TeamRepo) Unban(ctx context.Context, teamID uuid.UUID) error {
	query := squirrel.Update("teams").
		Set("is_banned", false).
		Set("banned_at", nil).
		Set("banned_reason", nil).
		Where(squirrel.Eq{"id": teamID}).
		Where(squirrel.Eq{"deleted_at": nil}).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("TeamRepo - Unban - BuildQuery: %w", err)
	}

	cmdTag, err := r.pool.Exec(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("TeamRepo - Unban - ExecQuery: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return entityError.ErrTeamNotFound
	}
	return nil
}

func (r *TeamRepo) SetHidden(ctx context.Context, teamID uuid.UUID, hidden bool) error {
	query := squirrel.Update("teams").
		Set("is_hidden", hidden).
		Where(squirrel.Eq{"id": teamID}).
		Where(squirrel.Eq{"deleted_at": nil}).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("TeamRepo - SetHidden - BuildQuery: %w", err)
	}

	cmdTag, err := r.pool.Exec(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("TeamRepo - SetHidden - ExecQuery: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return entityError.ErrTeamNotFound
	}
	return nil
}

func (r *TeamRepo) GetAll(ctx context.Context) ([]*entity.Team, error) {
	query := squirrel.Select(teamSelectColumns...).
		From("teams").
		Where(squirrel.Eq{"deleted_at": nil}).
		OrderBy("created_at ASC").
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("TeamRepo - GetAll - BuildQuery: %w", err)
	}

	rows, err := r.pool.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("TeamRepo - GetAll - Query: %w", err)
	}
	defer rows.Close()

	var teams []*entity.Team
	for rows.Next() {
		team, err := scanTeam(rows)
		if err != nil {
			return nil, fmt.Errorf("TeamRepo - GetAll - Scan: %w", err)
		}
		teams = append(teams, team)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("TeamRepo - GetAll - Rows: %w", err)
	}

	return teams, nil
}
