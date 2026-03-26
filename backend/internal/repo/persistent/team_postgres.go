package persistent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/samber/lo"
	"github.com/wahrwelt-kit/go-pgkit/pgutil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

type TeamRepo struct {
	BaseRepo
}

var _ repo.TeamRepository = (*TeamRepo)(nil)

func NewTeamRepo(pool *pgxpool.Pool) *TeamRepo {
	return &TeamRepo{BaseRepo: BaseRepo{pool: pool}}
}

func toDomainTeam(ID uuid.UUID, name string, inviteToken uuid.UUID, inviteTokenExpiresAt *time.Time, captainID uuid.UUID, bracketID *uuid.UUID, isSolo, isAutoCreated, isBanned, isHidden *bool, bannedAt *time.Time, bannedReason *string, createdAt *time.Time) *domain.Team {
	return &domain.Team{
		ID:                   ID,
		Name:                 name,
		InviteToken:          inviteToken,
		InviteTokenExpiresAt: inviteTokenExpiresAt,
		CaptainID:            captainID,
		BracketID:            bracketID,
		IsSolo:               lo.FromPtr(isSolo),
		IsAutoCreated:        lo.FromPtr(isAutoCreated),
		IsBanned:             lo.FromPtr(isBanned),
		BannedAt:             bannedAt,
		BannedReason:         bannedReason,
		IsHidden:             lo.FromPtr(isHidden),
		CreatedAt:            pgutil.PtrTimeToTime(createdAt),
	}
}

func (r *TeamRepo) GetByID(ctx context.Context, ID uuid.UUID) (*domain.Team, error) {
	row, err := r.Q(ctx).GetTeamByID(ctx, ID)
	if err != nil {
		if pgutil.IsNoRows(err) {
			return nil, httperr.ErrTeamNotFound
		}

		return nil, fmt.Errorf("TeamRepo - GetByID: %w", err)
	}

	return toDomainTeam(row.ID, row.Name, row.InviteToken, pgutil.TimestamptzToTime(row.InviteTokenExpiresAt), row.CaptainID, row.BracketID, row.IsSolo, row.IsAutoCreated, row.IsBanned, row.IsHidden, pgutil.TimestamptzToTime(row.BannedAt), row.BannedReason, pgutil.TimestamptzToTime(row.CreatedAt)), nil
}

func (r *TeamRepo) GetByInviteToken(ctx context.Context, inviteToken uuid.UUID) (*domain.Team, error) {
	if inviteToken == uuid.Nil {
		return nil, httperr.ErrTeamNotFound
	}

	row, err := r.Q(ctx).GetTeamByInviteToken(ctx, inviteToken)
	if err != nil {
		if pgutil.IsNoRows(err) {
			return nil, httperr.ErrTeamNotFound
		}

		return nil, fmt.Errorf("TeamRepo - GetByInviteToken: %w", err)
	}

	return toDomainTeam(row.ID, row.Name, row.InviteToken, pgutil.TimestamptzToTime(row.InviteTokenExpiresAt), row.CaptainID, row.BracketID, row.IsSolo, row.IsAutoCreated, row.IsBanned, row.IsHidden, pgutil.TimestamptzToTime(row.BannedAt), row.BannedReason, pgutil.TimestamptzToTime(row.CreatedAt)), nil
}

func (r *TeamRepo) GetByName(ctx context.Context, name string) (*domain.Team, error) {
	row, err := r.Q(ctx).GetTeamByName(ctx, name)
	if err != nil {
		if pgutil.IsNoRows(err) {
			return nil, httperr.ErrTeamNotFound
		}

		return nil, fmt.Errorf("TeamRepo - GetByName: %w", err)
	}

	return toDomainTeam(row.ID, row.Name, row.InviteToken, pgutil.TimestamptzToTime(row.InviteTokenExpiresAt), row.CaptainID, row.BracketID, row.IsSolo, row.IsAutoCreated, row.IsBanned, row.IsHidden, pgutil.TimestamptzToTime(row.BannedAt), row.BannedReason, pgutil.TimestamptzToTime(row.CreatedAt)), nil
}

func (r *TeamRepo) Delete(ctx context.Context, ID uuid.UUID) error {
	now := time.Now()

	_, err := r.Q(ctx).SoftDeleteTeam(ctx, sqlc.SoftDeleteTeamParams{
		ID:        ID,
		DeletedAt: pgutil.TimeToTimestamptz(&now),
	})
	if err != nil && !pgutil.IsNoRows(err) {
		return fmt.Errorf("TeamRepo - Delete: %w", err)
	}

	return nil
}

func (r *TeamRepo) HardDeleteTeams(ctx context.Context, cutoffDate time.Time) error {
	err := r.Q(ctx).HardDeleteTeamsBefore(ctx, pgutil.TimeToTimestamptz(&cutoffDate))
	if err != nil {
		return fmt.Errorf("TeamRepo - HardDeleteTeams: %w", err)
	}

	return nil
}

func (r *TeamRepo) GetSoloTeamByUserID(ctx context.Context, userID uuid.UUID) (*domain.Team, error) {
	row, err := r.Q(ctx).GetSoloTeamByUserID(ctx, userID)
	if err != nil {
		if pgutil.IsNoRows(err) {
			return nil, httperr.ErrTeamNotFound
		}

		return nil, fmt.Errorf("TeamRepo - GetSoloTeamByUserID: %w", err)
	}

	return toDomainTeam(row.ID, row.Name, row.InviteToken, pgutil.TimestamptzToTime(row.InviteTokenExpiresAt), row.CaptainID, row.BracketID, row.IsSolo, row.IsAutoCreated, row.IsBanned, row.IsHidden, pgutil.TimestamptzToTime(row.BannedAt), row.BannedReason, pgutil.TimestamptzToTime(row.CreatedAt)), nil
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

func (r *TeamRepo) Ban(ctx context.Context, teamID uuid.UUID, reason string) error {
	bannedAt := time.Now()

	_, err := r.Q(ctx).BanTeam(ctx, sqlc.BanTeamParams{
		ID:           teamID,
		BannedAt:     pgutil.TimeToTimestamptz(&bannedAt),
		BannedReason: &reason,
	})
	if err != nil {
		if pgutil.IsNoRows(err) {
			return httperr.ErrTeamNotFound
		}

		return fmt.Errorf("TeamRepo - Ban: %w", err)
	}

	return nil
}

func (r *TeamRepo) Unban(ctx context.Context, teamID uuid.UUID) error {
	_, err := r.Q(ctx).UnbanTeam(ctx, teamID)
	if err != nil {
		if pgutil.IsNoRows(err) {
			return httperr.ErrTeamNotFound
		}

		return fmt.Errorf("TeamRepo - Unban: %w", err)
	}

	return nil
}

func (r *TeamRepo) SetHidden(ctx context.Context, teamID uuid.UUID, hidden bool) error {
	_, err := r.Q(ctx).SetTeamHidden(ctx, sqlc.SetTeamHiddenParams{ID: teamID, IsHidden: &hidden})
	if err != nil {
		if pgutil.IsNoRows(err) {
			return httperr.ErrTeamNotFound
		}

		return fmt.Errorf("TeamRepo - SetHidden: %w", err)
	}

	return nil
}

func (r *TeamRepo) GetAll(ctx context.Context) ([]*domain.Team, error) {
	rows, err := r.Q(ctx).GetAllTeams(ctx)
	if err != nil {
		return nil, fmt.Errorf("TeamRepo - GetAll: %w", err)
	}

	out := make([]*domain.Team, 0, len(rows))
	for _, row := range rows {
		out = append(out, toDomainTeam(row.ID, row.Name, row.InviteToken, pgutil.TimestamptzToTime(row.InviteTokenExpiresAt), row.CaptainID, row.BracketID, row.IsSolo, row.IsAutoCreated, row.IsBanned, row.IsHidden, pgutil.TimestamptzToTime(row.BannedAt), row.BannedReason, pgutil.TimestamptzToTime(row.CreatedAt)))
	}

	return out, nil
}

func (r *TeamRepo) SetBracket(ctx context.Context, teamID uuid.UUID, bracketID *uuid.UUID) error {
	_, err := r.Q(ctx).SetTeamBracket(ctx, sqlc.SetTeamBracketParams{ID: teamID, BracketID: bracketID})
	if err != nil {
		if pgutil.IsNoRows(err) {
			return httperr.ErrTeamNotFound
		}

		return fmt.Errorf("TeamRepo - SetBracket: %w", err)
	}

	return nil
}

func (r *TeamRepo) Search(ctx context.Context, search *string, limit, offset int) ([]*domain.Team, error) {
	limit32, offset32, err := toLimitOffset(limit, offset)
	if err != nil {
		return nil, fmt.Errorf("TeamRepo - Search: %w", err)
	}

	var escapedSearch *string

	if search != nil {
		escaped := EscapeLikePattern(*search)
		escapedSearch = &escaped
	}

	rows, err := r.Q(ctx).SearchTeams(ctx, sqlc.SearchTeamsParams{
		Limit:  limit32,
		Offset: offset32,
		Search: escapedSearch,
	})
	if err != nil {
		return nil, fmt.Errorf("TeamRepo - Search: %w", err)
	}

	out := make([]*domain.Team, 0, len(rows))
	for _, row := range rows {
		out = append(out, toDomainTeam(row.ID, row.Name, row.InviteToken, pgutil.TimestamptzToTime(row.InviteTokenExpiresAt), row.CaptainID, row.BracketID, row.IsSolo, row.IsAutoCreated, row.IsBanned, row.IsHidden, pgutil.TimestamptzToTime(row.BannedAt), row.BannedReason, pgutil.TimestamptzToTime(row.CreatedAt)))
	}

	return out, nil
}

func (r *TeamRepo) CountSearch(ctx context.Context, search *string) (int64, error) {
	var escapedSearch *string

	if search != nil {
		escaped := EscapeLikePattern(*search)
		escapedSearch = &escaped
	}

	count, err := r.Q(ctx).CountSearchTeams(ctx, escapedSearch)
	if err != nil {
		return 0, fmt.Errorf("TeamRepo - CountSearch: %w", err)
	}

	return count, nil
}

func (r *TeamRepo) SearchAdmin(ctx context.Context, search *string, limit, offset int) ([]*domain.Team, error) {
	limit32, offset32, err := toLimitOffset(limit, offset)
	if err != nil {
		return nil, fmt.Errorf("TeamRepo - SearchAdmin: %w", err)
	}

	var escapedSearch *string

	if search != nil {
		escaped := EscapeLikePattern(*search)
		escapedSearch = &escaped
	}

	rows, err := r.Q(ctx).SearchTeamsAdmin(ctx, sqlc.SearchTeamsAdminParams{
		Limit:  limit32,
		Offset: offset32,
		Search: escapedSearch,
	})
	if err != nil {
		return nil, fmt.Errorf("TeamRepo - SearchAdmin: %w", err)
	}

	out := make([]*domain.Team, 0, len(rows))
	for _, row := range rows {
		out = append(out, toDomainTeam(row.ID, row.Name, row.InviteToken, pgutil.TimestamptzToTime(row.InviteTokenExpiresAt), row.CaptainID, row.BracketID, row.IsSolo, row.IsAutoCreated, row.IsBanned, row.IsHidden, pgutil.TimestamptzToTime(row.BannedAt), row.BannedReason, pgutil.TimestamptzToTime(row.CreatedAt)))
	}

	return out, nil
}

func (r *TeamRepo) CountSearchAdmin(ctx context.Context, search *string) (int64, error) {
	var escapedSearch *string

	if search != nil {
		escaped := EscapeLikePattern(*search)
		escapedSearch = &escaped
	}

	count, err := r.Q(ctx).CountSearchTeamsAdmin(ctx, escapedSearch)
	if err != nil {
		return 0, fmt.Errorf("TeamRepo - CountSearchAdmin: %w", err)
	}

	return count, nil
}

func (r *TeamRepo) UpdateAdmin(ctx context.Context, teamID uuid.UUID, name *string, captainID, bracketID *uuid.UUID, isHidden *bool) error {
	if _, err := r.Q(ctx).UpdateTeamAdmin(ctx, sqlc.UpdateTeamAdminParams{
		ID:        teamID,
		Name:      name,
		CaptainID: captainID,
		BracketID: bracketID,
		IsHidden:  isHidden,
	}); err != nil {
		if pgutil.IsNoRows(err) {
			return httperr.ErrTeamNotFound
		}

		return fmt.Errorf("TeamRepo - UpdateAdmin: %w", err)
	}

	return nil
}

func (r *TeamRepo) UpdateName(ctx context.Context, teamID uuid.UUID, name string) error {
	if _, err := r.Q(ctx).UpdateTeamName(ctx, sqlc.UpdateTeamNameParams{
		ID:   teamID,
		Name: name,
	}); err != nil {
		if pgutil.IsNoRows(err) {
			return httperr.ErrTeamNotFound
		}

		if pgutil.IsPgUniqueViolation(err) {
			return httperr.ErrTeamAlreadyExists
		}

		return fmt.Errorf("TeamRepo - UpdateName: %w", err)
	}

	return nil
}

func (r *TeamRepo) Create(ctx context.Context, team *domain.Team) error {
	team.CreatedAt = time.Now()

	id, err := r.Q(ctx).CreateTeamReturningID(ctx, sqlc.CreateTeamReturningIDParams{
		Name:                 team.Name,
		InviteToken:          team.InviteToken,
		CaptainID:            team.CaptainID,
		IsSolo:               &team.IsSolo,
		IsAutoCreated:        &team.IsAutoCreated,
		CreatedAt:            pgutil.TimeToTimestamptz(&team.CreatedAt),
		InviteTokenExpiresAt: pgutil.TimeToTimestamptz(team.InviteTokenExpiresAt),
	})
	if err != nil {
		if pgutil.IsPgUniqueViolation(err) {
			return httperr.ErrTeamAlreadyExists
		}

		return fmt.Errorf("TeamRepo - Create: %w", err)
	}

	team.ID = id

	return nil
}

func (r *TeamRepo) Lock(ctx context.Context, teamID uuid.UUID) error {
	_, err := r.Q(ctx).LockTeam(ctx, teamID)
	if err != nil {
		if pgutil.IsNoRows(err) {
			return httperr.ErrTeamNotFound
		}

		return fmt.Errorf("TeamRepo - Lock: %w", err)
	}

	return nil
}

func (r *TeamRepo) AcquireAdvisoryLock(ctx context.Context, lockKey int64) error {
	db := ExtractDB(ctx, r.pool)
	if _, err := db.Exec(ctx, "SELECT pg_advisory_xact_lock($1::bigint)", lockKey); err != nil {
		return fmt.Errorf("TeamRepo - AcquireAdvisoryLock: %w", err)
	}

	return nil
}

func (r *TeamRepo) UpdateAvatarURL(ctx context.Context, teamID uuid.UUID, avatarURL string) error {
	err := r.Q(ctx).UpdateTeamAvatarURL(ctx, sqlc.UpdateTeamAvatarURLParams{
		ID:        teamID,
		AvatarUrl: &avatarURL,
	})
	if err != nil {
		return fmt.Errorf("TeamRepo - UpdateAvatarURL: %w", err)
	}

	return nil
}

func (r *TeamRepo) ClearAvatarURL(ctx context.Context, teamID uuid.UUID) (*string, error) {
	oldURL, err := r.Q(ctx).ClearTeamAvatarURL(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("TeamRepo - ClearAvatarURL: %w", err)
	}

	return oldURL, nil
}

func (r *TeamRepo) ListAllTeamAvatarURLs(ctx context.Context) ([]*string, error) {
	urls, err := r.Q(ctx).ListAllTeamAvatarURLs(ctx)
	if err != nil {
		return nil, fmt.Errorf("TeamRepo - ListAllTeamAvatarURLs: %w", err)
	}

	return urls, nil
}

func (r *TeamRepo) UpdateCaptain(ctx context.Context, teamID, newCaptainID uuid.UUID) error {
	_, err := r.Q(ctx).UpdateTeamCaptain(ctx, sqlc.UpdateTeamCaptainParams{
		ID:        teamID,
		CaptainID: newCaptainID,
	})
	if err != nil {
		if pgutil.IsNoRows(err) {
			return httperr.ErrTeamNotFound
		}

		return fmt.Errorf("TeamRepo - UpdateCaptain: %w", err)
	}

	return nil
}

func (r *TeamRepo) UpdateInviteToken(ctx context.Context, teamID, inviteToken uuid.UUID, expiresAt *time.Time) error {
	_, err := r.Q(ctx).UpdateInviteToken(ctx, sqlc.UpdateInviteTokenParams{
		ID:                   teamID,
		InviteToken:          inviteToken,
		InviteTokenExpiresAt: pgutil.TimeToTimestamptz(expiresAt),
	})
	if err != nil {
		if pgutil.IsNoRows(err) {
			return httperr.ErrTeamNotFound
		}

		return fmt.Errorf("TeamRepo - UpdateInviteToken: %w", err)
	}

	return nil
}

func (r *TeamRepo) CreateAuditLog(ctx context.Context, log *domain.TeamAuditLog) error {
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

	err := r.Q(ctx).CreateTeamAuditLog(ctx, sqlc.CreateTeamAuditLogParams{
		ID:        log.ID,
		TeamID:    log.TeamID,
		UserID:    log.UserID,
		Action:    string(log.Action),
		Details:   detailsJSON,
		CreatedAt: pgutil.TimeToTimestamptz(&log.CreatedAt),
	})
	if err != nil {
		return fmt.Errorf("TeamRepo - CreateAuditLog: %w", err)
	}

	return nil
}

func (r *TeamRepo) GetLatestAuditLogByTeamIDAndAction(ctx context.Context, teamID uuid.UUID, action string) (*domain.TeamAuditLog, error) {
	row, err := r.Q(ctx).GetLatestTeamAuditLogByTeamIDAndAction(ctx, sqlc.GetLatestTeamAuditLogByTeamIDAndActionParams{
		TeamID: teamID,
		Action: action,
	})
	if err != nil {
		if pgutil.IsNoRows(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("TeamRepo - GetLatestAuditLogByTeamIDAndAction: %w", err)
	}

	var details map[string]any

	if len(row.Details) > 0 {
		err := json.Unmarshal(row.Details, &details)
		if err != nil {
			return nil, fmt.Errorf("TeamRepo - GetLatestAuditLogByTeamIDAndAction - Unmarshal details: %w", err)
		}
	}

	createdAt := pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(row.CreatedAt))

	return &domain.TeamAuditLog{
		ID:        row.ID,
		TeamID:    row.TeamID,
		UserID:    row.UserID,
		Action:    domain.TeamAuditAction(row.Action),
		Details:   details,
		CreatedAt: createdAt,
	}, nil
}
