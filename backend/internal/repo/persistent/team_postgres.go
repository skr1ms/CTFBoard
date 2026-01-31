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
		Columns("id", "name", "invite_token", "captain_id", "is_solo", "is_auto_created", "created_at").
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
	query := squirrel.Select("id", "name", "invite_token", "captain_id", "is_solo", "is_auto_created", "created_at").
		From("teams").
		Where(squirrel.Eq{"id": ID}).
		Where(squirrel.Eq{"deleted_at": nil}).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("TeamRepo - GetByID - BuildQuery: %w", err)
	}

	var team entity.Team
	err = r.pool.QueryRow(ctx, sqlQuery, args...).Scan(
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
		return nil, fmt.Errorf("TeamRepo - GetByID - Scan: %w", err)
	}

	return &team, nil
}

func (r *TeamRepo) GetByInviteToken(ctx context.Context, inviteToken uuid.UUID) (*entity.Team, error) {
	if inviteToken == uuid.Nil {
		return nil, entityError.ErrTeamNotFound
	}

	query := squirrel.Select("id", "name", "invite_token", "captain_id", "is_solo", "is_auto_created", "created_at").
		From("teams").
		Where(squirrel.Eq{"invite_token": inviteToken}).
		Where(squirrel.Eq{"deleted_at": nil}).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("TeamRepo - GetByInviteToken - BuildQuery: %w", err)
	}

	var team entity.Team
	err = r.pool.QueryRow(ctx, sqlQuery, args...).Scan(
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
		return nil, fmt.Errorf("TeamRepo - GetByInviteToken - Scan: %w", err)
	}

	return &team, nil
}

func (r *TeamRepo) GetByName(ctx context.Context, name string) (*entity.Team, error) {
	query := squirrel.Select("id", "name", "invite_token", "captain_id", "is_solo", "is_auto_created", "created_at").
		From("teams").
		Where(squirrel.Eq{"name": name}).
		Where(squirrel.Eq{"deleted_at": nil}).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("TeamRepo - GetByName - BuildQuery: %w", err)
	}

	var team entity.Team
	err = r.pool.QueryRow(ctx, sqlQuery, args...).Scan(
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
		return nil, fmt.Errorf("TeamRepo - GetByName - Scan: %w", err)
	}

	return &team, nil
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
	query := squirrel.Select("t.id", "t.name", "t.invite_token", "t.captain_id", "t.is_solo", "t.is_auto_created", "t.created_at").
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

	var team entity.Team
	err = r.pool.QueryRow(ctx, sqlQuery, args...).Scan(
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
		return nil, fmt.Errorf("TeamRepo - GetSoloTeamByUserID - Scan: %w", err)
	}

	return &team, nil
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
