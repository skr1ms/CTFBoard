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

	TeamProfile struct {
		Team         *domain.Team
		CustomFields CustomFieldValues
	}

	TeamMe struct {
		Team         *domain.Team
		Members      []*domain.User
		MinTeamSize  int
		MeetsMinSize bool
		CustomFields CustomFieldValues
	}

	TeamUpdateParams struct {
		Name         *string
		CustomFields *CustomFieldValues
	}

	// TeamReadUseCase exposes public/team-read operations used by the transport layer.
	TeamReadUseCase interface {
		GetByID(ctx context.Context, ID uuid.UUID) (*domain.Team, error)
		GetProfile(ctx context.Context, ID uuid.UUID) (*TeamProfile, error)
		ListTeams(ctx context.Context, search *string, page, perPage int) (*Paginated[*domain.Team], error)
		GetTeamSolves(ctx context.Context, teamID uuid.UUID) ([]*domain.SolveWithDetails, error)
		GetTeamFails(ctx context.Context, teamID uuid.UUID, page, perPage int) (*Paginated[*domain.SubmissionWithDetails], error)
		GetTeamAwards(ctx context.Context, teamID uuid.UUID) ([]*domain.Award, error)
	}

	// TeamSelfUseCase exposes participant-owned team lifecycle and membership operations.
	TeamSelfUseCase interface {
		TryCreate(ctx context.Context, name string, captainID uuid.UUID, isSolo bool) (*TeamCreateResult, error)
		ConfirmCreate(ctx context.Context, name string, captainID uuid.UUID, isSolo bool) (*domain.Team, error)
		Join(ctx context.Context, inviteToken, userID uuid.UUID, confirmReset bool) (*domain.Team, error)
		Leave(ctx context.Context, userID uuid.UUID) error
		TransferCaptain(ctx context.Context, captainID, newCaptainID uuid.UUID) error
		GetMyTeam(ctx context.Context, userID uuid.UUID) (*TeamMe, error)
		CreateSoloTeam(ctx context.Context, userID uuid.UUID, confirmReset bool) (*domain.Team, error)
		DisbandTeam(ctx context.Context, captainID uuid.UUID) error
		KickMember(ctx context.Context, captainID, targetUserID uuid.UUID) error
		UpdateMyTeam(ctx context.Context, captainID uuid.UUID, params TeamUpdateParams) (*TeamProfile, error)
		GetInviteToken(ctx context.Context, captainID uuid.UUID) (*domain.Team, error)
		RegenerateInviteToken(ctx context.Context, captainID uuid.UUID) (*domain.Team, error)
	}

	// TeamAdminUseCase exposes administrative team operations.
	TeamAdminUseCase interface {
		BanTeam(ctx context.Context, teamID uuid.UUID, reason string, banMembers bool, actorID uuid.UUID) error
		UnbanTeam(ctx context.Context, teamID, actorID uuid.UUID) error
		SetHidden(ctx context.Context, teamID uuid.UUID, hidden bool) error
		SetBracket(ctx context.Context, teamID uuid.UUID, bracketID *uuid.UUID) error
		AdminListTeams(ctx context.Context, search *string, page, perPage int) (*Paginated[*domain.Team], error)
		AdminUpdate(ctx context.Context, teamID uuid.UUID, name *string, captainID, bracketID *uuid.UUID, isHidden *bool) (*domain.Team, error)
		AdminDelete(ctx context.Context, teamID uuid.UUID) error
		AdminGetMembers(ctx context.Context, teamID uuid.UUID) ([]*domain.User, error)
		AdminAddMember(ctx context.Context, teamID, userID uuid.UUID) error
		AdminRemoveMember(ctx context.Context, teamID, userID uuid.UUID) error
	}

	// TeamUseCase keeps the legacy aggregate contract for internal implementations.
	TeamUseCase interface {
		TeamReadUseCase
		TeamSelfUseCase
		TeamAdminUseCase

		Create(ctx context.Context, name string, captainID uuid.UUID, isSolo, confirmReset bool) (*domain.Team, error)
		GetTeamMembers(ctx context.Context, teamID uuid.UUID) ([]*domain.User, error)
	}
)
