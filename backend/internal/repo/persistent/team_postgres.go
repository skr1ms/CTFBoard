package persistent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TeamRepo struct {
	pool *pgxpool.Pool
}

var _ repo.TeamRepository = (*TeamRepo)(nil)

func NewTeamRepo(pool *pgxpool.Pool) *TeamRepo {
	return &TeamRepo{pool: pool}
}

func (r *TeamRepo) q(ctx context.Context) *sqlc.Queries {
	return sqlc.New(ExtractDB(ctx, r.pool))
}

func toEntityTeam(ID uuid.UUID, name string, inviteToken, captainID uuid.UUID, bracketID *uuid.UUID, isSolo, isAutoCreated, isBanned, isHidden *bool, bannedAt *time.Time, bannedReason *string, createdAt *time.Time) *entity.Team {
	return &entity.Team{
		ID:            ID,
		Name:          name,
		InviteToken:   inviteToken,
		CaptainID:     captainID,
		BracketID:     bracketID,
		IsSolo:        boolPtrToBool(isSolo),
		IsAutoCreated: boolPtrToBool(isAutoCreated),
		IsBanned:      boolPtrToBool(isBanned),
		BannedAt:      bannedAt,
		BannedReason:  bannedReason,
		IsHidden:      boolPtrToBool(isHidden),
		CreatedAt:     ptrTimeToTime(createdAt),
	}
}

func (r *TeamRepo) GetByID(ctx context.Context, ID uuid.UUID) (*entity.Team, error) {
	row, err := r.q(ctx).GetTeamByID(ctx, ID)
	if err != nil {
		if isNoRows(err) {
			return nil, httperr.ErrTeamNotFound
		}
		return nil, fmt.Errorf("TeamRepo - GetByID: %w", err)
	}
	return toEntityTeam(row.ID, row.Name, row.InviteToken, row.CaptainID, row.BracketID, row.IsSolo, row.IsAutoCreated, row.IsBanned, row.IsHidden, row.BannedAt, row.BannedReason, row.CreatedAt), nil
}

func (r *TeamRepo) GetByInviteToken(ctx context.Context, inviteToken uuid.UUID) (*entity.Team, error) {
	if inviteToken == uuid.Nil {
		return nil, httperr.ErrTeamNotFound
	}
	row, err := r.q(ctx).GetTeamByInviteToken(ctx, inviteToken)
	if err != nil {
		if isNoRows(err) {
			return nil, httperr.ErrTeamNotFound
		}
		return nil, fmt.Errorf("TeamRepo - GetByInviteToken: %w", err)
	}
	return toEntityTeam(row.ID, row.Name, row.InviteToken, row.CaptainID, row.BracketID, row.IsSolo, row.IsAutoCreated, row.IsBanned, row.IsHidden, row.BannedAt, row.BannedReason, row.CreatedAt), nil
}

func (r *TeamRepo) GetByName(ctx context.Context, name string) (*entity.Team, error) {
	row, err := r.q(ctx).GetTeamByName(ctx, name)
	if err != nil {
		if isNoRows(err) {
			return nil, httperr.ErrTeamNotFound
		}
		return nil, fmt.Errorf("TeamRepo - GetByName: %w", err)
	}
	return toEntityTeam(row.ID, row.Name, row.InviteToken, row.CaptainID, row.BracketID, row.IsSolo, row.IsAutoCreated, row.IsBanned, row.IsHidden, row.BannedAt, row.BannedReason, row.CreatedAt), nil
}

func (r *TeamRepo) Delete(ctx context.Context, ID uuid.UUID) error {
	_, err := r.q(ctx).SoftDeleteTeam(ctx, ID)
	if err != nil && !isNoRows(err) {
		return fmt.Errorf("TeamRepo - Delete: %w", err)
	}
	return nil
}

func (r *TeamRepo) HardDeleteTeams(ctx context.Context, cutoffDate time.Time) error {
	if err := r.q(ctx).HardDeleteTeamsBefore(ctx, &cutoffDate); err != nil {
		return fmt.Errorf("TeamRepo - HardDeleteTeams: %w", err)
	}
	return nil
}

func (r *TeamRepo) GetSoloTeamByUserID(ctx context.Context, userID uuid.UUID) (*entity.Team, error) {
	row, err := r.q(ctx).GetSoloTeamByUserID(ctx, userID)
	if err != nil {
		if isNoRows(err) {
			return nil, httperr.ErrTeamNotFound
		}
		return nil, fmt.Errorf("TeamRepo - GetSoloTeamByUserID: %w", err)
	}
	return toEntityTeam(row.ID, row.Name, row.InviteToken, row.CaptainID, row.BracketID, row.IsSolo, row.IsAutoCreated, row.IsBanned, row.IsHidden, row.BannedAt, row.BannedReason, row.CreatedAt), nil
}

func (r *TeamRepo) CountTeamMembers(ctx context.Context, teamID uuid.UUID) (int, error) {
	n, err := r.q(ctx).CountTeamMembers(ctx, &teamID)
	if err != nil {
		return 0, fmt.Errorf("TeamRepo - CountTeamMembers: %w", err)
	}
	return int(n), nil
}

func (r *TeamRepo) CountActiveTeams(ctx context.Context) (int, error) {
	n, err := r.q(ctx).CountActiveTeams(ctx)
	if err != nil {
		return 0, fmt.Errorf("TeamRepo - CountActiveTeams: %w", err)
	}
	return int(n), nil
}

func (r *TeamRepo) Ban(ctx context.Context, teamID uuid.UUID, reason string) error {
	bannedAt := time.Now()
	_, err := r.q(ctx).BanTeam(ctx, sqlc.BanTeamParams{
		ID:           teamID,
		BannedAt:     &bannedAt,
		BannedReason: &reason,
	})
	if err != nil {
		if isNoRows(err) {
			return httperr.ErrTeamNotFound
		}
		return fmt.Errorf("TeamRepo - Ban: %w", err)
	}
	return nil
}

func (r *TeamRepo) Unban(ctx context.Context, teamID uuid.UUID) error {
	_, err := r.q(ctx).UnbanTeam(ctx, teamID)
	if err != nil {
		if isNoRows(err) {
			return httperr.ErrTeamNotFound
		}
		return fmt.Errorf("TeamRepo - Unban: %w", err)
	}
	return nil
}

func (r *TeamRepo) SetHidden(ctx context.Context, teamID uuid.UUID, hidden bool) error {
	_, err := r.q(ctx).SetTeamHidden(ctx, sqlc.SetTeamHiddenParams{ID: teamID, IsHidden: &hidden})
	if err != nil {
		if isNoRows(err) {
			return httperr.ErrTeamNotFound
		}
		return fmt.Errorf("TeamRepo - SetHidden: %w", err)
	}
	return nil
}

func (r *TeamRepo) GetAll(ctx context.Context) ([]*entity.Team, error) {
	rows, err := r.q(ctx).GetAllTeams(ctx)
	if err != nil {
		return nil, fmt.Errorf("TeamRepo - GetAll: %w", err)
	}
	out := make([]*entity.Team, 0, len(rows))
	for _, row := range rows {
		out = append(out, toEntityTeam(row.ID, row.Name, row.InviteToken, row.CaptainID, row.BracketID, row.IsSolo, row.IsAutoCreated, row.IsBanned, row.IsHidden, row.BannedAt, row.BannedReason, row.CreatedAt))
	}
	return out, nil
}

func (r *TeamRepo) SetBracket(ctx context.Context, teamID uuid.UUID, bracketID *uuid.UUID) error {
	_, err := r.q(ctx).SetTeamBracket(ctx, sqlc.SetTeamBracketParams{ID: teamID, BracketID: bracketID})
	if err != nil {
		if isNoRows(err) {
			return httperr.ErrTeamNotFound
		}
		return fmt.Errorf("TeamRepo - SetBracket: %w", err)
	}
	return nil
}

func (r *TeamRepo) Search(ctx context.Context, search *string, limit, offset int) ([]*entity.Team, error) {
	limit32, err := intToInt32Safe(limit)
	if err != nil {
		return nil, fmt.Errorf("TeamRepo - Search - limit: %w", err)
	}
	offset32, err := intToInt32Safe(offset)
	if err != nil {
		return nil, fmt.Errorf("TeamRepo - Search - offset: %w", err)
	}
	rows, err := r.q(ctx).SearchTeams(ctx, sqlc.SearchTeamsParams{
		Limit:  limit32,
		Offset: offset32,
		Search: search,
	})
	if err != nil {
		return nil, fmt.Errorf("TeamRepo - Search: %w", err)
	}
	out := make([]*entity.Team, 0, len(rows))
	for _, row := range rows {
		out = append(out, toEntityTeam(row.ID, row.Name, row.InviteToken, row.CaptainID, row.BracketID, row.IsSolo, row.IsAutoCreated, row.IsBanned, row.IsHidden, row.BannedAt, row.BannedReason, row.CreatedAt))
	}
	return out, nil
}

func (r *TeamRepo) CountSearch(ctx context.Context, search *string) (int64, error) {
	count, err := r.q(ctx).CountSearchTeams(ctx, search)
	if err != nil {
		return 0, fmt.Errorf("TeamRepo - CountSearch: %w", err)
	}
	return count, nil
}

func (r *TeamRepo) SearchAdmin(ctx context.Context, search *string, limit, offset int) ([]*entity.Team, error) {
	limit32, err := intToInt32Safe(limit)
	if err != nil {
		return nil, fmt.Errorf("TeamRepo - SearchAdmin - limit: %w", err)
	}
	offset32, err := intToInt32Safe(offset)
	if err != nil {
		return nil, fmt.Errorf("TeamRepo - SearchAdmin - offset: %w", err)
	}
	rows, err := r.q(ctx).SearchTeamsAdmin(ctx, sqlc.SearchTeamsAdminParams{
		Limit:  limit32,
		Offset: offset32,
		Search: search,
	})
	if err != nil {
		return nil, fmt.Errorf("TeamRepo - SearchAdmin: %w", err)
	}
	out := make([]*entity.Team, 0, len(rows))
	for _, row := range rows {
		out = append(out, toEntityTeam(row.ID, row.Name, row.InviteToken, row.CaptainID, row.BracketID, row.IsSolo, row.IsAutoCreated, row.IsBanned, row.IsHidden, row.BannedAt, row.BannedReason, row.CreatedAt))
	}
	return out, nil
}

func (r *TeamRepo) CountSearchAdmin(ctx context.Context, search *string) (int64, error) {
	count, err := r.q(ctx).CountSearchTeamsAdmin(ctx, search)
	if err != nil {
		return 0, fmt.Errorf("TeamRepo - CountSearchAdmin: %w", err)
	}
	return count, nil
}

func (r *TeamRepo) UpdateAdmin(ctx context.Context, teamID uuid.UUID, name *string, captainID, bracketID *uuid.UUID, isHidden *bool) error {
	if _, err := r.q(ctx).UpdateTeamAdmin(ctx, sqlc.UpdateTeamAdminParams{
		ID:        teamID,
		Name:      name,
		CaptainID: captainID,
		BracketID: bracketID,
		IsHidden:  isHidden,
	}); err != nil {
		if isNoRows(err) {
			return httperr.ErrTeamNotFound
		}
		return fmt.Errorf("TeamRepo - UpdateAdmin: %w", err)
	}
	return nil
}

func (r *TeamRepo) UpdateName(ctx context.Context, teamID uuid.UUID, name string) error {
	if _, err := r.q(ctx).UpdateTeamName(ctx, sqlc.UpdateTeamNameParams{
		ID:   teamID,
		Name: name,
	}); err != nil {
		if isNoRows(err) {
			return httperr.ErrTeamNotFound
		}
		if isPgUniqueViolation(err) {
			return httperr.ErrTeamAlreadyExists
		}
		return fmt.Errorf("TeamRepo - UpdateName: %w", err)
	}
	return nil
}

func (r *TeamRepo) Create(ctx context.Context, team *entity.Team) error {
	team.CreatedAt = time.Now()
	id, err := r.q(ctx).CreateTeamReturningID(ctx, sqlc.CreateTeamReturningIDParams{
		Name:          team.Name,
		InviteToken:   team.InviteToken,
		CaptainID:     team.CaptainID,
		IsSolo:        &team.IsSolo,
		IsAutoCreated: &team.IsAutoCreated,
		CreatedAt:     &team.CreatedAt,
	})
	if err != nil {
		return fmt.Errorf("TeamRepo - Create: %w", err)
	}
	team.ID = id
	return nil
}

func (r *TeamRepo) Lock(ctx context.Context, teamID uuid.UUID) error {
	_, err := r.q(ctx).LockTeam(ctx, teamID)
	if err != nil {
		if isNoRows(err) {
			return httperr.ErrTeamNotFound
		}
		return fmt.Errorf("TeamRepo - Lock: %w", err)
	}
	return nil
}

func (r *TeamRepo) AcquireAdvisoryLock(ctx context.Context, lockKey int64) error {
	db := ExtractDB(ctx, r.pool)
	if _, err := db.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", lockKey); err != nil {
		return fmt.Errorf("TeamRepo - AcquireAdvisoryLock: %w", err)
	}
	return nil
}

func (r *TeamRepo) UpdateCaptain(ctx context.Context, teamID, newCaptainID uuid.UUID) error {
	_, err := r.q(ctx).UpdateTeamCaptain(ctx, sqlc.UpdateTeamCaptainParams{
		ID:        teamID,
		CaptainID: newCaptainID,
	})
	if err != nil {
		if isNoRows(err) {
			return httperr.ErrTeamNotFound
		}
		return fmt.Errorf("TeamRepo - UpdateCaptain: %w", err)
	}
	return nil
}

func (r *TeamRepo) CreateAuditLog(ctx context.Context, log *entity.TeamAuditLog) error {
	log.ID = uuid.New()
	log.CreatedAt = time.Now()
	detailsJSON := []byte("{}")
	if log.Details != nil {
		var jsonErr error
		detailsJSON, jsonErr = json.Marshal(log.Details)
		if jsonErr != nil {
			return fmt.Errorf("TeamRepo - CreateAuditLog - MarshalDetails: %w", jsonErr)
		}
	}
	err := r.q(ctx).CreateTeamAuditLog(ctx, sqlc.CreateTeamAuditLogParams{
		ID:        log.ID,
		TeamID:    log.TeamID,
		UserID:    log.UserID,
		Action:    string(log.Action),
		Details:   detailsJSON,
		CreatedAt: &log.CreatedAt,
	})
	if err != nil {
		return fmt.Errorf("TeamRepo - CreateAuditLog: %w", err)
	}
	return nil
}
