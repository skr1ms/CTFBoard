package persistent

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
)

func (r *TeamRepo) GetByID(ctx context.Context, ID uuid.UUID) (*domain.Team, error) {
	row, err := GetOrNotFound(func() (sqlc.GetTeamByIDRow, error) { return r.Q(ctx).GetTeamByID(ctx, ID) }, apperr.ErrTeamNotFound, "TeamRepo - GetByID")
	if err != nil {
		return nil, err
	}

	return toDomainTeam(teamRowFromSQLC(
		row.ID, row.Name, row.InviteToken,
		row.InviteTokenExpiresAt, row.BannedAt, row.CreatedAt,
		row.CaptainID, row.BracketID,
		row.IsSolo, row.IsAutoCreated, row.IsBanned, row.IsHidden,
		row.BannedReason, row.AvatarUrl,
	)), nil
}

func (r *TeamRepo) GetByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*domain.Team, error) {
	if len(ids) == 0 {
		return map[uuid.UUID]*domain.Team{}, nil
	}

	rows, err := r.Q(ctx).GetTeamsByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("TeamRepo - GetByIDs: %w", err)
	}

	out := make(map[uuid.UUID]*domain.Team, len(ids))

	for _, row := range rows {
		team := toDomainTeam(teamRowFromSQLC(
			row.ID, row.Name, row.InviteToken,
			row.InviteTokenExpiresAt, row.BannedAt, row.CreatedAt,
			row.CaptainID, row.BracketID,
			row.IsSolo, row.IsAutoCreated, row.IsBanned, row.IsHidden,
			row.BannedReason, row.AvatarUrl,
		))

		out[team.ID] = team
	}

	return out, nil
}

func (r *TeamRepo) GetByInviteToken(ctx context.Context, inviteToken uuid.UUID) (*domain.Team, error) {
	if inviteToken == uuid.Nil {
		return nil, apperr.ErrTeamNotFound
	}

	row, err := GetOrNotFound(func() (sqlc.GetTeamByInviteTokenRow, error) { return r.Q(ctx).GetTeamByInviteToken(ctx, inviteToken) }, apperr.ErrTeamNotFound, "TeamRepo - GetByInviteToken")
	if err != nil {
		return nil, err
	}

	return toDomainTeam(teamRowFromSQLC(
		row.ID, row.Name, row.InviteToken,
		row.InviteTokenExpiresAt, row.BannedAt, row.CreatedAt,
		row.CaptainID, row.BracketID,
		row.IsSolo, row.IsAutoCreated, row.IsBanned, row.IsHidden,
		row.BannedReason, row.AvatarUrl,
	)), nil
}

func (r *TeamRepo) GetByName(ctx context.Context, name string) (*domain.Team, error) {
	row, err := GetOrNotFound(func() (sqlc.GetTeamByNameRow, error) { return r.Q(ctx).GetTeamByName(ctx, name) }, apperr.ErrTeamNotFound, "TeamRepo - GetByName")
	if err != nil {
		return nil, err
	}

	return toDomainTeam(teamRowFromSQLC(
		row.ID, row.Name, row.InviteToken,
		row.InviteTokenExpiresAt, row.BannedAt, row.CreatedAt,
		row.CaptainID, row.BracketID,
		row.IsSolo, row.IsAutoCreated, row.IsBanned, row.IsHidden,
		row.BannedReason, row.AvatarUrl,
	)), nil
}

func (r *TeamRepo) GetSoloTeamByUserID(ctx context.Context, userID uuid.UUID) (*domain.Team, error) {
	row, err := GetOrNotFound(func() (sqlc.GetSoloTeamByUserIDRow, error) { return r.Q(ctx).GetSoloTeamByUserID(ctx, userID) }, apperr.ErrTeamNotFound, "TeamRepo - GetSoloTeamByUserID")
	if err != nil {
		return nil, err
	}

	return toDomainTeam(teamRowFromSQLC(
		row.ID, row.Name, row.InviteToken,
		row.InviteTokenExpiresAt, row.BannedAt, row.CreatedAt,
		row.CaptainID, row.BracketID,
		row.IsSolo, row.IsAutoCreated, row.IsBanned, row.IsHidden,
		row.BannedReason, row.AvatarUrl,
	)), nil
}

func (r *TeamRepo) CountTeamMembers(ctx context.Context, teamID uuid.UUID) (int, error) {
	n, err := r.Q(ctx).CountTeamMembers(ctx, &teamID)
	if err != nil {
		return 0, fmt.Errorf("TeamRepo - CountTeamMembers: %w", err)
	}

	return int(n), nil
}

func (r *TeamRepo) CountActiveTeams(ctx context.Context) (int, error) {
	n, err := r.Q(ctx).CountActiveTeams(ctx)
	if err != nil {
		return 0, fmt.Errorf("TeamRepo - CountActiveTeams: %w", err)
	}

	return int(n), nil
}

func (r *TeamRepo) GetAll(ctx context.Context) ([]*domain.Team, error) {
	rows, err := r.Q(ctx).GetAllTeams(ctx)
	if err != nil {
		return nil, fmt.Errorf("TeamRepo - GetAll: %w", err)
	}

	out := make([]*domain.Team, 0, len(rows))
	for _, row := range rows {
		out = append(out, toDomainTeam(teamRowFromSQLC(
			row.ID, row.Name, row.InviteToken,
			row.InviteTokenExpiresAt, row.BannedAt, row.CreatedAt,
			row.CaptainID, row.BracketID,
			row.IsSolo, row.IsAutoCreated, row.IsBanned, row.IsHidden,
			row.BannedReason, row.AvatarUrl,
		)))
	}

	return out, nil
}

func (r *TeamRepo) Lock(ctx context.Context, teamID uuid.UUID) error {
	_, err := GetOrNotFound(func() (uuid.UUID, error) { return r.Q(ctx).LockTeam(ctx, teamID) }, apperr.ErrTeamNotFound, "TeamRepo - Lock")

	return err
}
