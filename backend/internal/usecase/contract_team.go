package usecase

import (
	"context"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

// =============================================================================
// Team
// =============================================================================

type ConfirmReason string

const (
	ConfirmReasonSoloTeamReset ConfirmReason = "solo_team_reset"
)

type (
	TeamCreateAffectedData struct {
		SolveCount      int
		Points          int
		HintUnlockCount int
		AwardsTotal     int
	}

	// TeamCreateResult is the result of a team creation attempt that may require explicit confirmation before proceeding.
	TeamCreateResult struct {
		Team               *domain.Team
		RequiresConfirm    bool
		ConfirmationReason ConfirmReason
		AffectedData       *TeamCreateAffectedData
	}

	// TeamUseCase handles team lifecycle, membership, bans, and admin operations.
	TeamUseCase interface {
		Create(ctx context.Context, name string, captainID uuid.UUID, isSolo, confirmReset bool) (*domain.Team, error)
		TryCreate(ctx context.Context, name string, captainID uuid.UUID, isSolo bool) (*TeamCreateResult, error)
		ConfirmCreate(ctx context.Context, name string, captainID uuid.UUID, isSolo bool) (*domain.Team, error)
		Join(ctx context.Context, inviteToken, userID uuid.UUID, confirmReset bool) (*domain.Team, error)
		Leave(ctx context.Context, userID uuid.UUID) error
		TransferCaptain(ctx context.Context, captainID, newCaptainID uuid.UUID) error
		GetByID(ctx context.Context, ID uuid.UUID) (*domain.Team, error)
		GetMyTeam(ctx context.Context, userID uuid.UUID) (*domain.Team, []*domain.User, int, bool, error)
		GetTeamMembers(ctx context.Context, teamID uuid.UUID) ([]*domain.User, error)
		CreateSoloTeam(ctx context.Context, userID uuid.UUID, confirmReset bool) (*domain.Team, error)
		DisbandTeam(ctx context.Context, captainID uuid.UUID) error
		KickMember(ctx context.Context, captainID, targetUserID uuid.UUID) error
		BanTeam(ctx context.Context, teamID uuid.UUID, reason string, banMembers bool, actorID uuid.UUID) error
		UnbanTeam(ctx context.Context, teamID, actorID uuid.UUID) error
		SetHidden(ctx context.Context, teamID uuid.UUID, hidden bool) error
		SetBracket(ctx context.Context, teamID uuid.UUID, bracketID *uuid.UUID) error
		ListTeams(ctx context.Context, search *string, page, perPage int) (*Paginated[*domain.Team], error)
		AdminListTeams(ctx context.Context, search *string, page, perPage int) (*Paginated[*domain.Team], error)
		GetTeamSolves(ctx context.Context, teamID uuid.UUID) ([]*domain.SolveWithDetails, error)
		GetTeamFails(ctx context.Context, teamID uuid.UUID, page, perPage int) (*Paginated[*domain.SubmissionWithDetails], error)
		GetTeamAwards(ctx context.Context, teamID uuid.UUID) ([]*domain.Award, error)
		AdminUpdate(ctx context.Context, teamID uuid.UUID, name *string, captainID, bracketID *uuid.UUID, isHidden *bool) (*domain.Team, error)
		AdminDelete(ctx context.Context, teamID uuid.UUID) error
		AdminGetMembers(ctx context.Context, teamID uuid.UUID) ([]*domain.User, error)
		AdminAddMember(ctx context.Context, teamID, userID uuid.UUID) error
		AdminRemoveMember(ctx context.Context, teamID, userID uuid.UUID) error
		UpdateMyTeam(ctx context.Context, captainID uuid.UUID, name string) (*domain.Team, error)
		GetInviteToken(ctx context.Context, captainID uuid.UUID) (*domain.Team, error)
		RegenerateInviteToken(ctx context.Context, captainID uuid.UUID) (*domain.Team, error)
	}
)
