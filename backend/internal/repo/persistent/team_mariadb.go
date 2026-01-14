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
)

type TeamRepo struct {
	db *sql.DB
}

func NewTeamRepo(db *sql.DB) *TeamRepo {
	return &TeamRepo{db: db}
}

func (r *TeamRepo) Create(ctx context.Context, t *entity.Team) error {
	query := squirrel.Insert("teams").
		Columns("id", "name", "invite_token", "captain_id", "created_at").
		Values(uuid.New().String(), t.Name, t.InviteToken, t.CaptainId, time.Now())

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("TeamRepo - Create - BuildQuery: %w", err)
	}

	_, err = r.db.ExecContext(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("TeamRepo - Create - ExecQuery: %w", err)
	}

	return nil
}

func (r *TeamRepo) GetByID(ctx context.Context, id string) (*entity.Team, error) {
	uuidID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("TeamRepo - GetByID - ParseID: %w", err)
	}

	query := squirrel.Select("id", "name", "invite_token", "captain_id", "created_at").
		From("teams").
		Where(squirrel.Eq{"id": uuidID})

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("TeamRepo - GetByID - BuildQuery: %w", err)
	}

	var team entity.Team
	err = r.db.QueryRowContext(ctx, sqlQuery, args...).Scan(
		&team.Id,
		&team.Name,
		&team.InviteToken,
		&team.CaptainId,
		&team.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, entityError.ErrTeamNotFound
		}
		return nil, fmt.Errorf("TeamRepo - GetByID - Scan: %w", err)
	}

	return &team, nil
}

func (r *TeamRepo) GetByInviteToken(ctx context.Context, inviteToken string) (*entity.Team, error) {
	query := squirrel.Select("id", "name", "invite_token", "captain_id", "created_at").
		From("teams").
		Where(squirrel.Eq{"invite_token": inviteToken})

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("TeamRepo - GetByInviteToken - BuildQuery: %w", err)
	}

	var team entity.Team
	err = r.db.QueryRowContext(ctx, sqlQuery, args...).Scan(
		&team.Id,
		&team.Name,
		&team.InviteToken,
		&team.CaptainId,
		&team.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, entityError.ErrTeamNotFound
		}
		return nil, fmt.Errorf("TeamRepo - GetByInviteToken - Scan: %w", err)
	}

	return &team, nil
}

func (r *TeamRepo) GetByName(ctx context.Context, name string) (*entity.Team, error) {
	query := squirrel.Select("id", "name", "invite_token", "captain_id", "created_at").
		From("teams").
		Where(squirrel.Eq{"name": name})

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("TeamRepo - GetByName - BuildQuery: %w", err)
	}

	var team entity.Team
	err = r.db.QueryRowContext(ctx, sqlQuery, args...).Scan(
		&team.Id,
		&team.Name,
		&team.InviteToken,
		&team.CaptainId,
		&team.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, entityError.ErrTeamNotFound
		}
		return nil, fmt.Errorf("TeamRepo - GetByName - Scan: %w", err)
	}

	return &team, nil
}
