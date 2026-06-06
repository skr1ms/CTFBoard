package persistent

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
)

func (r *UserRepo) GetByID(ctx context.Context, ID uuid.UUID) (*domain.User, error) {
	u, err := GetOrNotFound(func() (sqlc.User, error) { return r.Q(ctx).GetUserByID(ctx, ID) }, apperr.ErrUserNotFound, "UserRepo - GetByID")
	if err != nil {
		return nil, err
	}

	return toDomainUser(userRowFrom(u)), nil
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	u, err := GetOrNotFound(func() (sqlc.User, error) { return r.Q(ctx).GetUserByEmail(ctx, email) }, apperr.ErrUserNotFound, "UserRepo - GetByEmail")
	if err != nil {
		return nil, err
	}

	return toDomainUser(userRowFrom(u)), nil
}

func (r *UserRepo) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	u, err := GetOrNotFound(func() (sqlc.User, error) { return r.Q(ctx).GetUserByUsername(ctx, username) }, apperr.ErrUserNotFound, "UserRepo - GetByUsername")
	if err != nil {
		return nil, err
	}

	return toDomainUser(userRowFrom(u)), nil
}

func (r *UserRepo) GetByTeamID(ctx context.Context, teamID uuid.UUID) ([]*domain.User, error) {
	rows, err := r.Q(ctx).ListUsersByTeamID(ctx, &teamID)
	if err != nil {
		return nil, fmt.Errorf("UserRepo - GetByTeamID: %w", err)
	}

	out := make([]*domain.User, 0, len(rows))
	for _, u := range rows {
		out = append(out, toDomainUser(userRowFrom(u)))
	}

	return out, nil
}

func (r *UserRepo) GetByTeamIDs(ctx context.Context, teamIDs []uuid.UUID) (map[uuid.UUID][]*domain.User, error) {
	if len(teamIDs) == 0 {
		return map[uuid.UUID][]*domain.User{}, nil
	}

	rows, err := r.Q(ctx).ListUsersByTeamIDs(ctx, teamIDs)
	if err != nil {
		return nil, fmt.Errorf("UserRepo - GetByTeamIDs: %w", err)
	}

	out := make(map[uuid.UUID][]*domain.User)

	for _, u := range rows {
		if u.TeamID != nil {
			out[*u.TeamID] = append(out[*u.TeamID], toDomainUser(userRowFrom(u)))
		}
	}

	return out, nil
}

func (r *UserRepo) GetAll(ctx context.Context) ([]*domain.User, error) {
	rows, err := r.Q(ctx).GetAllUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("UserRepo - GetAll: %w", err)
	}

	out := make([]*domain.User, 0, len(rows))
	for _, u := range rows {
		out = append(out, toDomainUser(userRowFrom(u)))
	}

	return out, nil
}
