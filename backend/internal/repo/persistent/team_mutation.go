package persistent

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-pgkit/pgutil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
)

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

func (r *TeamRepo) SetBracket(ctx context.Context, teamID uuid.UUID, bracketID *uuid.UUID) error {
	_, err := GetOrNotFound(func() (uuid.UUID, error) {
		return r.Q(ctx).SetTeamBracket(ctx, sqlc.SetTeamBracketParams{ID: teamID, BracketID: bracketID})
	}, apperr.ErrTeamNotFound, "TeamRepo - SetBracket")

	return err
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
