package team

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/cacheutil"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/guard"
	validation "github.com/TakuyaYagam1/AstroCTFb/pkg/validator"
)

// UpdateMyTeam validates the competition-level team-switch guard, runs updateMyTeamTx
// in a transaction, then evicts both the team-specific cache entry and the full
// scoreboard cache (name change affects public display).
func (uc *TeamUseCase) UpdateMyTeam(ctx context.Context, captainID uuid.UUID, params usecase.TeamUpdateParams) (*usecase.TeamProfile, error) {
	if params.Name == nil && params.CustomFields == nil {
		return nil, apperr.NewValidationErrorf("at least one team field must be provided")
	}

	if params.Name != nil && *params.Name == "" {
		return nil, apperr.NewValidationErrorf("name cannot be empty")
	}

	var (
		team        *domain.Team
		nameChanged bool
	)

	err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		var err error

		team, nameChanged, err = uc.updateMyTeamTx(ctx, captainID, params)
		if err != nil {
			return fmt.Errorf("TeamUseCase - UpdateMyTeam - updateMyTeamTx: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - UpdateMyTeam - TM.Run: %w", err)
	}

	if team == nil {
		return nil, nil
	}

	cacheutil.InvalidateTeam(ctx, uc.deps.TeamCache, uc.deps.Logger, team.ID)

	if nameChanged {
		cacheutil.InvalidateScoreboardForTeam(ctx, uc.deps.ScoreboardCache, team.ID)
	}

	customFields, err := uc.getSelfTeamCustomFields(ctx, team.ID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - UpdateMyTeam - getSelfTeamCustomFields: %w", err)
	}

	return &usecase.TeamProfile{Team: team, CustomFields: customFields}, nil
}

// updateMyTeamTx performs the team update inside a transaction: locks the caller
// and team rows, validates captainship, and applies name/custom-field changes.
func (uc *TeamUseCase) updateMyTeamTx(
	ctx context.Context,
	captainID uuid.UUID,
	params usecase.TeamUpdateParams,
) (*domain.Team, bool, error) {
	if err := uc.deps.UserRepo.Lock(ctx, captainID); err != nil {
		return nil, false, fmt.Errorf("TeamUseCase - updateMyTeamTx - UserRepo.Lock: %w", err)
	}

	user, err := uc.deps.UserRepo.GetByID(ctx, captainID)
	if err != nil {
		return nil, false, fmt.Errorf("TeamUseCase - updateMyTeamTx - UserRepo.GetByID: %w", err)
	}

	if user.TeamID == nil {
		return nil, false, apperr.ErrTeamNotFound
	}

	if user.IsBanned {
		return nil, false, apperr.ErrUserBanned
	}

	if err := uc.deps.TeamRepo.Lock(ctx, *user.TeamID); err != nil {
		return nil, false, fmt.Errorf("TeamUseCase - updateMyTeamTx - TeamRepo.Lock: %w", err)
	}

	team, err := uc.deps.TeamRepo.GetByID(ctx, *user.TeamID)
	if err != nil {
		return nil, false, fmt.Errorf("TeamUseCase - updateMyTeamTx - TeamRepo.GetByID: %w", err)
	}

	if team.IsBanned {
		return nil, false, apperr.ErrTeamBanned
	}

	if team.CaptainID != captainID {
		return nil, false, apperr.ErrNotCaptain
	}

	nameChanged := false

	if params.Name != nil && team.Name != *params.Name {
		comp, err := uc.deps.CompRepo.Get(ctx)
		if err != nil {
			return nil, false, fmt.Errorf("TeamUseCase - updateMyTeamTx - CompetitionRepo.Get: %w", err)
		}

		if err := guard.ValidateTeamSwitchState(comp); err != nil {
			return nil, false, fmt.Errorf("TeamUseCase - updateMyTeamTx - ValidateTeamSwitchState: %w", err)
		}

		if err := uc.validateTeamNameAvailable(ctx, *params.Name); err != nil {
			return nil, false, fmt.Errorf("TeamUseCase - updateMyTeamTx - validateTeamNameAvailable: %w", err)
		}

		if err := uc.deps.TeamRepo.UpdateName(ctx, team.ID, *params.Name); err != nil {
			return nil, false, fmt.Errorf("TeamUseCase - updateMyTeamTx - TeamRepo.UpdateName: %w", err)
		}

		team.Name = *params.Name
		nameChanged = true
	}

	if params.CustomFields != nil {
		customFields, err := uc.validateTeamCustomFields(ctx, *params.CustomFields)
		if err != nil {
			return nil, false, fmt.Errorf("TeamUseCase - updateMyTeamTx - validateTeamCustomFields: %w", err)
		}

		if err := uc.deps.FieldValueRepo.UpsertValues(ctx, team.ID, customFields); err != nil {
			return nil, false, fmt.Errorf("TeamUseCase - updateMyTeamTx - FieldValueRepo.UpsertValues: %w", err)
		}
	}

	return team, nameChanged, nil
}

func (uc *TeamUseCase) validateTeamCustomFields(ctx context.Context, raw usecase.CustomFieldValues) (map[string]string, error) {
	if uc.deps.FieldValidator == nil {
		return nil, apperr.NewValidationErrorf("custom fields are not configured")
	}

	if uc.deps.FieldValueRepo == nil {
		return nil, apperr.NewValidationErrorf("custom field storage is not configured")
	}

	if err := validation.ValidateCustomFieldEnvelope(raw); err != nil {
		return nil, apperr.NewValidationErrorf("%v", err)
	}

	fieldValues := make(map[uuid.UUID]any, len(raw))

	for key, value := range raw {
		id, err := uuid.Parse(key)
		if err != nil {
			return nil, apperr.NewValidationErrorf("invalid custom field key")
		}

		fieldValues[id] = value
	}

	normalized, err := uc.deps.FieldValidator.ValidateEditableValues(ctx, domain.EntityTypeTeam, fieldValues)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - validateTeamCustomFields - FieldValidator.ValidateEditableValues: %w", err)
	}

	return usecase.CustomFieldStorageValuesToStringKeyMap(normalized), nil
}

func (uc *TeamUseCase) getSelfTeamCustomFields(ctx context.Context, teamID uuid.UUID) (usecase.CustomFieldValues, error) {
	if uc.deps.FieldRepo == nil || uc.deps.FieldValueRepo == nil {
		return nil, nil
	}

	fields, err := uc.deps.FieldRepo.GetByEntityType(ctx, domain.EntityTypeTeam)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - getSelfTeamCustomFields - FieldRepo.GetByEntityType: %w", err)
	}

	values, err := uc.deps.FieldValueRepo.GetByEntityID(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - getSelfTeamCustomFields - FieldValueRepo.GetByEntityID: %w", err)
	}

	return visibleTeamFieldValuesToMap(fields, values, selfTeamField), nil
}

func (uc *TeamUseCase) GetInviteToken(ctx context.Context, captainID uuid.UUID) (*domain.Team, error) {
	if _, err := uc.deps.Guard.RequireTeamSwitch(ctx); err != nil {
		return nil, fmt.Errorf("TeamUseCase - GetInviteToken - Guard.RequireTeamSwitch: %w", err)
	}

	user, err := uc.deps.UserRepo.GetByID(ctx, captainID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - GetInviteToken - UserRepo.GetByID: %w", err)
	}

	if user.IsBanned {
		return nil, apperr.ErrUserBanned
	}

	if user.TeamID == nil {
		return nil, apperr.ErrTeamNotFound
	}

	team, err := uc.deps.TeamRepo.GetByID(ctx, *user.TeamID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - GetInviteToken - TeamRepo.GetByID: %w", err)
	}

	if team.IsSolo {
		return nil, apperr.ErrTeamNotFound
	}

	if team.CaptainID != captainID {
		return nil, apperr.ErrNotCaptain
	}

	if team.IsBanned {
		return nil, apperr.ErrTeamBanned
	}

	return team, nil
}

const defaultInviteTokenTTL = 7 * 24 * time.Hour

// RegenerateInviteToken rotates the team invite token inside a transaction and returns the updated team.
func (uc *TeamUseCase) RegenerateInviteToken(ctx context.Context, captainID uuid.UUID) (*domain.Team, error) {
	var team *domain.Team

	err := uc.deps.TM.Run(ctx, func(txCtx context.Context) error {
		comp, err := uc.deps.CompRepo.Get(txCtx)
		if err != nil {
			return fmt.Errorf("TeamUseCase - RegenerateInviteToken - CompetitionRepo.Get: %w", err)
		}

		if err := guard.ValidateTeamSwitchState(comp); err != nil {
			return fmt.Errorf("TeamUseCase - RegenerateInviteToken - ValidateTeamSwitchState: %w", err)
		}

		if err := uc.deps.UserRepo.Lock(txCtx, captainID); err != nil {
			return fmt.Errorf("TeamUseCase - RegenerateInviteToken - UserRepo.Lock: %w", err)
		}

		user, err := uc.deps.UserRepo.GetByID(txCtx, captainID)
		if err != nil {
			return fmt.Errorf("TeamUseCase - RegenerateInviteToken - UserRepo.GetByID: %w", err)
		}

		if user.IsBanned {
			return apperr.ErrUserBanned
		}

		if user.TeamID == nil {
			return apperr.ErrTeamNotFound
		}

		if err := uc.deps.TeamRepo.Lock(txCtx, *user.TeamID); err != nil {
			return fmt.Errorf("TeamUseCase - RegenerateInviteToken - TeamRepo.Lock: %w", err)
		}

		var errTeam error

		team, errTeam = uc.deps.TeamRepo.GetByID(txCtx, *user.TeamID)
		if errTeam != nil {
			return fmt.Errorf("TeamUseCase - RegenerateInviteToken - TeamRepo.GetByID: %w", errTeam)
		}

		if team.CaptainID != captainID {
			return apperr.ErrNotCaptain
		}

		if team.IsSolo {
			return apperr.ErrTeamNotFound
		}

		if team.IsBanned {
			return apperr.ErrTeamBanned
		}

		newToken := uuid.New()

		expiresAt := time.Now().Add(defaultInviteTokenTTL)
		if err := uc.deps.TeamRepo.UpdateInviteToken(txCtx, team.ID, newToken, &expiresAt); err != nil {
			return fmt.Errorf("TeamUseCase - RegenerateInviteToken - TeamRepo.UpdateInviteToken: %w", err)
		}

		team.InviteToken = newToken
		team.InviteTokenExpiresAt = &expiresAt

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - RegenerateInviteToken - TM.Run: %w", err)
	}

	return team, nil
}
