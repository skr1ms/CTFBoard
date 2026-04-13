package persistent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wahrwelt-kit/go-pgkit/pgutil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
)

type TeamRepo struct {
	BaseRepo
}

var _ repo.TeamRepository = (*TeamRepo)(nil)

func NewTeamRepo(pool *pgxpool.Pool) *TeamRepo {
	return &TeamRepo{BaseRepo: BaseRepo{pool: pool}}
}

type teamRow struct {
	ID                   uuid.UUID
	Name                 string
	InviteToken          uuid.UUID
	InviteTokenExpiresAt *time.Time
	CaptainID            uuid.UUID
	BracketID            *uuid.UUID
	IsSolo               bool
	IsAutoCreated        bool
	IsBanned             bool
	IsHidden             bool
	BannedAt             *time.Time
	BannedReason         *string
	AvatarURL            *string
	CreatedAt            *time.Time
}

func toDomainTeam(r teamRow) *domain.Team {
	return &domain.Team{
		ID:                   r.ID,
		Name:                 r.Name,
		InviteToken:          r.InviteToken,
		InviteTokenExpiresAt: r.InviteTokenExpiresAt,
		CaptainID:            r.CaptainID,
		BracketID:            r.BracketID,
		IsSolo:               r.IsSolo,
		IsAutoCreated:        r.IsAutoCreated,
		IsBanned:             r.IsBanned,
		BannedAt:             r.BannedAt,
		BannedReason:         r.BannedReason,
		IsHidden:             r.IsHidden,
		AvatarURL:            r.AvatarURL,
		CreatedAt:            pgutil.PtrTimeToTime(r.CreatedAt),
	}
}

// teamRowFromSQLC converts the common set of sqlc-generated team fields to teamRow,
// eliminating the repeated pgutil.TimestamptzToTime conversions across all repo methods.
func teamRowFromSQLC(
	id uuid.UUID, name string, inviteToken uuid.UUID,
	inviteTokenExpiresAt, bannedAt, createdAt pgtype.Timestamptz,
	captainID uuid.UUID, bracketID *uuid.UUID,
	isSolo, isAutoCreated, isBanned, isHidden bool,
	bannedReason, avatarURL *string,
) teamRow {
	return teamRow{
		ID: id, Name: name, InviteToken: inviteToken,
		InviteTokenExpiresAt: pgutil.TimestamptzToTime(inviteTokenExpiresAt),
		CaptainID:            captainID, BracketID: bracketID,
		IsSolo: isSolo, IsAutoCreated: isAutoCreated, IsBanned: isBanned, IsHidden: isHidden,
		BannedAt: pgutil.TimestamptzToTime(bannedAt), BannedReason: bannedReason,
		AvatarURL: avatarURL,
		CreatedAt: pgutil.TimestamptzToTime(createdAt),
	}
}

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

	const q = `
		SELECT id, name, invite_token, invite_token_expires_at, captain_id, bracket_id,
		       is_solo, is_auto_created, is_banned, banned_at, banned_reason, is_hidden,
		       avatar_url, created_at
		FROM teams
		WHERE id = ANY($1) AND deleted_at IS NULL`

	rows, err := r.DB(ctx).Query(ctx, q, ids)
	if err != nil {
		return nil, fmt.Errorf("TeamRepo - GetByIDs: %w", err)
	}

	defer rows.Close()

	out := make(map[uuid.UUID]*domain.Team, len(ids))

	for rows.Next() {
		var (
			row     teamRow
			pgDates [3]pgtype.Timestamptz
		)

		if err = rows.Scan(
			&row.ID, &row.Name, &row.InviteToken, &pgDates[0],
			&row.CaptainID, &row.BracketID,
			&row.IsSolo, &row.IsAutoCreated, &row.IsBanned, &pgDates[1],
			&row.BannedReason, &row.IsHidden,
			&row.AvatarURL, &pgDates[2],
		); err != nil {
			return nil, fmt.Errorf("TeamRepo - GetByIDs - scan: %w", err)
		}

		row.InviteTokenExpiresAt = pgutil.TimestamptzToTime(pgDates[0])
		row.BannedAt = pgutil.TimestamptzToTime(pgDates[1])
		row.CreatedAt = pgutil.TimestamptzToTime(pgDates[2])

		out[row.ID] = toDomainTeam(row)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("TeamRepo - GetByIDs - rows: %w", err)
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

func (r *TeamRepo) Ban(ctx context.Context, teamID uuid.UUID, reason string) error {
	bannedAt := time.Now()
	_, err := GetOrNotFound(func() (uuid.UUID, error) {
		return r.Q(ctx).BanTeam(ctx, sqlc.BanTeamParams{ID: teamID, BannedAt: pgutil.TimeToTimestamptz(&bannedAt), BannedReason: &reason})
	}, apperr.ErrTeamNotFound, "TeamRepo - Ban")

	return err
}

func (r *TeamRepo) Unban(ctx context.Context, teamID uuid.UUID) error {
	_, err := GetOrNotFound(func() (uuid.UUID, error) { return r.Q(ctx).UnbanTeam(ctx, teamID) }, apperr.ErrTeamNotFound, "TeamRepo - Unban")

	return err
}

func (r *TeamRepo) SetHidden(ctx context.Context, teamID uuid.UUID, hidden bool) error {
	_, err := GetOrNotFound(func() (uuid.UUID, error) {
		return r.Q(ctx).SetTeamHidden(ctx, sqlc.SetTeamHiddenParams{ID: teamID, IsHidden: hidden})
	}, apperr.ErrTeamNotFound, "TeamRepo - SetHidden")

	return err
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

func (r *TeamRepo) SetBracket(ctx context.Context, teamID uuid.UUID, bracketID *uuid.UUID) error {
	_, err := GetOrNotFound(func() (uuid.UUID, error) {
		return r.Q(ctx).SetTeamBracket(ctx, sqlc.SetTeamBracketParams{ID: teamID, BracketID: bracketID})
	}, apperr.ErrTeamNotFound, "TeamRepo - SetBracket")

	return err
}

func (r *TeamRepo) Search(ctx context.Context, search *string, limit, offset int) ([]*domain.Team, error) {
	limit32, offset32, err := toLimitOffset(limit, offset)
	if err != nil {
		return nil, fmt.Errorf("TeamRepo - Search: %w", err)
	}

	escapedSearch := EscapeSearchPtr(search)

	rows, err := r.Q(ctx).SearchTeams(ctx, sqlc.SearchTeamsParams{
		Limit:  limit32,
		Offset: offset32,
		Search: escapedSearch,
	})
	if err != nil {
		return nil, fmt.Errorf("TeamRepo - Search - scan: %w", err)
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

func (r *TeamRepo) CountSearch(ctx context.Context, search *string) (int64, error) {
	escapedSearch := EscapeSearchPtr(search)

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

	escapedSearch := EscapeSearchPtr(search)

	rows, err := r.Q(ctx).SearchTeamsAdmin(ctx, sqlc.SearchTeamsAdminParams{
		Limit:  limit32,
		Offset: offset32,
		Search: escapedSearch,
	})
	if err != nil {
		return nil, fmt.Errorf("TeamRepo - SearchAdmin - scan: %w", err)
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

func (r *TeamRepo) CountSearchAdmin(ctx context.Context, search *string) (int64, error) {
	escapedSearch := EscapeSearchPtr(search)

	count, err := r.Q(ctx).CountSearchTeamsAdmin(ctx, escapedSearch)
	if err != nil {
		return 0, fmt.Errorf("TeamRepo - CountSearchAdmin: %w", err)
	}

	return count, nil
}

func (r *TeamRepo) UpdateAdmin(ctx context.Context, teamID uuid.UUID, name *string, captainID, bracketID *uuid.UUID, isHidden *bool) error {
	_, err := GetOrNotFound(func() (uuid.UUID, error) {
		return r.Q(ctx).UpdateTeamAdmin(ctx, sqlc.UpdateTeamAdminParams{ID: teamID, Name: name, CaptainID: captainID, BracketID: bracketID, IsHidden: isHidden})
	}, apperr.ErrTeamNotFound, "TeamRepo - UpdateAdmin")

	return err
}

func (r *TeamRepo) UpdateName(ctx context.Context, teamID uuid.UUID, name string) error {
	if _, err := r.Q(ctx).UpdateTeamName(ctx, sqlc.UpdateTeamNameParams{
		ID:   teamID,
		Name: name,
	}); err != nil {
		if pgutil.IsNoRows(err) {
			return apperr.ErrTeamNotFound
		}

		if pgutil.IsPgUniqueViolation(err) {
			return apperr.ErrTeamAlreadyExists
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
		IsSolo:               team.IsSolo,
		IsAutoCreated:        team.IsAutoCreated,
		CreatedAt:            pgutil.TimeToTimestamptz(&team.CreatedAt),
		InviteTokenExpiresAt: pgutil.TimeToTimestamptz(team.InviteTokenExpiresAt),
	})
	if err != nil {
		if pgutil.IsPgUniqueViolation(err) {
			return apperr.ErrTeamAlreadyExists
		}

		return fmt.Errorf("TeamRepo - Create: %w", err)
	}

	team.ID = id

	return nil
}

func (r *TeamRepo) Lock(ctx context.Context, teamID uuid.UUID) error {
	_, err := GetOrNotFound(func() (uuid.UUID, error) { return r.Q(ctx).LockTeam(ctx, teamID) }, apperr.ErrTeamNotFound, "TeamRepo - Lock")

	return err
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

func (r *TeamRepo) ClearAvatarURL(ctx context.Context, teamID uuid.UUID) error {
	if err := r.Q(ctx).ClearTeamAvatarURL(ctx, teamID); err != nil {
		return fmt.Errorf("TeamRepo - ClearAvatarURL: %w", err)
	}

	return nil
}

func (r *TeamRepo) ListAllTeamAvatarURLs(ctx context.Context) ([]*string, error) {
	urls, err := r.Q(ctx).ListAllTeamAvatarURLs(ctx)
	if err != nil {
		return nil, fmt.Errorf("TeamRepo - ListAllTeamAvatarURLs: %w", err)
	}

	return urls, nil
}

func (r *TeamRepo) UpdateCaptain(ctx context.Context, teamID, newCaptainID uuid.UUID) error {
	_, err := GetOrNotFound(func() (uuid.UUID, error) {
		return r.Q(ctx).UpdateTeamCaptain(ctx, sqlc.UpdateTeamCaptainParams{ID: teamID, CaptainID: newCaptainID})
	}, apperr.ErrTeamNotFound, "TeamRepo - UpdateCaptain")

	return err
}

func (r *TeamRepo) UpdateInviteToken(ctx context.Context, teamID, inviteToken uuid.UUID, expiresAt *time.Time) error {
	_, err := GetOrNotFound(func() (uuid.UUID, error) {
		return r.Q(ctx).UpdateInviteToken(ctx, sqlc.UpdateInviteTokenParams{
			ID:                   teamID,
			InviteToken:          inviteToken,
			InviteTokenExpiresAt: pgutil.TimeToTimestamptz(expiresAt),
		})
	}, apperr.ErrTeamNotFound, "TeamRepo - UpdateInviteToken")

	return err
}

// CreateAuditLog inserts a team audit-log entry, generating a new UUID and
// setting CreatedAt to the current time. The Details field, when non-nil, is
// marshalled to JSON; an empty JSON object is stored otherwise.
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

// GetLatestAuditLogByTeamIDAndAction returns the most recent audit log entry for the
// given team and action, or (nil, nil) when no such entry exists (ErrNoRows is not an error here).
// The JSON details field is unmarshalled into map[string]any.
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
