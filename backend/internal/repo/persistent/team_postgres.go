package persistent

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/repo/persistent/sqlc"
)

type TeamRepo struct {
	db *pgxpool.Pool
	q  *sqlc.Queries
}

func NewTeamRepo(db *pgxpool.Pool) *TeamRepo {
	return &TeamRepo{db: db, q: sqlc.New(db)}
}

func toEntityTeamFromRow(id uuid.UUID, name string, inviteToken, captainID uuid.UUID, isSolo, isAutoCreated, isBanned, isHidden *bool, bannedAt *time.Time, bannedReason *string, createdAt *time.Time) *entity.Team {
	return &entity.Team{
		ID:            id,
		Name:          name,
		InviteToken:   inviteToken,
		CaptainID:     captainID,
		IsSolo:        boolPtrToBool(isSolo),
		IsAutoCreated: boolPtrToBool(isAutoCreated),
		IsBanned:      boolPtrToBool(isBanned),
		BannedAt:      bannedAt,
		BannedReason:  bannedReason,
		IsHidden:      boolPtrToBool(isHidden),
		CreatedAt:     ptrTimeToTime(createdAt),
	}
}

func (r *TeamRepo) Create(ctx context.Context, t *entity.Team) error {
	t.ID = uuid.New()
	t.CreatedAt = time.Now()
	err := r.q.CreateTeam(ctx, sqlc.CreateTeamParams{
		ID:            t.ID,
		Name:          t.Name,
		InviteToken:   t.InviteToken,
		CaptainID:     t.CaptainID,
		IsSolo:        &t.IsSolo,
		IsAutoCreated: &t.IsAutoCreated,
		CreatedAt:     &t.CreatedAt,
	})
	if err != nil {
		return fmt.Errorf("TeamRepo - Create: %w", err)
	}
	return nil
}

func (r *TeamRepo) GetByID(ctx context.Context, id uuid.UUID) (*entity.Team, error) {
	row, err := r.q.GetTeamByID(ctx, id)
	if err != nil {
		if isNoRows(err) {
			return nil, entityError.ErrTeamNotFound
		}
		return nil, fmt.Errorf("TeamRepo - GetByID: %w", err)
	}
	return toEntityTeamFromRow(row.ID, row.Name, row.InviteToken, row.CaptainID, row.IsSolo, row.IsAutoCreated, row.IsBanned, row.IsHidden, row.BannedAt, row.BannedReason, row.CreatedAt), nil
}

func (r *TeamRepo) GetByInviteToken(ctx context.Context, inviteToken uuid.UUID) (*entity.Team, error) {
	if inviteToken == uuid.Nil {
		return nil, entityError.ErrTeamNotFound
	}
	row, err := r.q.GetTeamByInviteToken(ctx, inviteToken)
	if err != nil {
		if isNoRows(err) {
			return nil, entityError.ErrTeamNotFound
		}
		return nil, fmt.Errorf("TeamRepo - GetByInviteToken: %w", err)
	}
	return toEntityTeamFromRow(row.ID, row.Name, row.InviteToken, row.CaptainID, row.IsSolo, row.IsAutoCreated, row.IsBanned, row.IsHidden, row.BannedAt, row.BannedReason, row.CreatedAt), nil
}

func (r *TeamRepo) GetByName(ctx context.Context, name string) (*entity.Team, error) {
	row, err := r.q.GetTeamByName(ctx, name)
	if err != nil {
		if isNoRows(err) {
			return nil, entityError.ErrTeamNotFound
		}
		return nil, fmt.Errorf("TeamRepo - GetByName: %w", err)
	}
	return toEntityTeamFromRow(row.ID, row.Name, row.InviteToken, row.CaptainID, row.IsSolo, row.IsAutoCreated, row.IsBanned, row.IsHidden, row.BannedAt, row.BannedReason, row.CreatedAt), nil
}

func (r *TeamRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.q.SoftDeleteTeam(ctx, id)
	if err != nil {
		if isNoRows(err) {
			return entityError.ErrTeamNotFound
		}
		return fmt.Errorf("TeamRepo - Delete: %w", err)
	}
	return nil
}

func (r *TeamRepo) HardDeleteTeams(ctx context.Context, cutoffDate time.Time) error {
	if err := r.q.HardDeleteTeamsBefore(ctx, &cutoffDate); err != nil {
		return fmt.Errorf("TeamRepo - HardDeleteTeams: %w", err)
	}
	return nil
}

func (r *TeamRepo) GetSoloTeamByUserID(ctx context.Context, userID uuid.UUID) (*entity.Team, error) {
	row, err := r.q.GetSoloTeamByUserID(ctx, userID)
	if err != nil {
		if isNoRows(err) {
			return nil, entityError.ErrTeamNotFound
		}
		return nil, fmt.Errorf("TeamRepo - GetSoloTeamByUserID: %w", err)
	}
	return toEntityTeamFromRow(row.ID, row.Name, row.InviteToken, row.CaptainID, row.IsSolo, row.IsAutoCreated, row.IsBanned, row.IsHidden, row.BannedAt, row.BannedReason, row.CreatedAt), nil
}

func (r *TeamRepo) CountTeamMembers(ctx context.Context, teamID uuid.UUID) (int, error) {
	n, err := r.q.CountTeamMembers(ctx, &teamID)
	if err != nil {
		return 0, fmt.Errorf("TeamRepo - CountTeamMembers: %w", err)
	}
	return int(n), nil
}

func (r *TeamRepo) Ban(ctx context.Context, teamID uuid.UUID, reason string) error {
	bannedAt := time.Now()
	_, err := r.q.BanTeam(ctx, sqlc.BanTeamParams{
		ID:           teamID,
		BannedAt:     &bannedAt,
		BannedReason: &reason,
	})
	if err != nil {
		if isNoRows(err) {
			return entityError.ErrTeamNotFound
		}
		return fmt.Errorf("TeamRepo - Ban: %w", err)
	}
	return nil
}

func (r *TeamRepo) Unban(ctx context.Context, teamID uuid.UUID) error {
	_, err := r.q.UnbanTeam(ctx, teamID)
	if err != nil {
		if isNoRows(err) {
			return entityError.ErrTeamNotFound
		}
		return fmt.Errorf("TeamRepo - Unban: %w", err)
	}
	return nil
}

func (r *TeamRepo) SetHidden(ctx context.Context, teamID uuid.UUID, hidden bool) error {
	_, err := r.q.SetTeamHidden(ctx, sqlc.SetTeamHiddenParams{ID: teamID, IsHidden: &hidden})
	if err != nil {
		if isNoRows(err) {
			return entityError.ErrTeamNotFound
		}
		return fmt.Errorf("TeamRepo - SetHidden: %w", err)
	}
	return nil
}

func (r *TeamRepo) GetAll(ctx context.Context) ([]*entity.Team, error) {
	rows, err := r.q.GetAllTeams(ctx)
	if err != nil {
		return nil, fmt.Errorf("TeamRepo - GetAll: %w", err)
	}
	out := make([]*entity.Team, 0, len(rows))
	for _, row := range rows {
		out = append(out, toEntityTeamFromRow(row.ID, row.Name, row.InviteToken, row.CaptainID, row.IsSolo, row.IsAutoCreated, row.IsBanned, row.IsHidden, row.BannedAt, row.BannedReason, row.CreatedAt))
	}
	return out, nil
}
