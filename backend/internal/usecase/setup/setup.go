package setup

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-jwtkit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

// SetupDeps are the external dependencies required by SetupUseCase.
type SetupDeps struct {
	UserUC      usecase.UserUseCase
	CompUC      usecase.CompetitionUseCase
	CompParamUC setupCompetitionParamUseCase
	SettingsUC  usecase.SettingsUseCase
	TM          setupTransactionManager
	JWTService  jwtkit.Service
}

type setupTransactionManager interface {
	Run(ctx context.Context, fn func(context.Context) error) error
}

type setupCompetitionParamUseCase interface {
	GetBool(ctx context.Context, key string, defaultVal bool) bool
	GetForUpdate(ctx context.Context, key string) (*domain.CompetitionParam, error)
	Set(ctx context.Context, params usecase.CompetitionParamSetParams) error
	SetBatch(ctx context.Context, params []*domain.CompetitionParam, actorID uuid.UUID, clientIP string) error
}

// SetupUseCase orchestrates the first-run setup wizard completion.
type SetupUseCase struct {
	deps SetupDeps
	mu   sync.Mutex
}

var _ usecase.SetupUseCase = (*SetupUseCase)(nil)

// NewSetupUseCase constructs a SetupUseCase.
func NewSetupUseCase(deps SetupDeps) *SetupUseCase {
	return &SetupUseCase{deps: deps}
}

// IsComplete reports whether the platform setup has been completed.
func (uc *SetupUseCase) IsComplete(ctx context.Context) (bool, error) {
	return uc.deps.CompParamUC.GetBool(ctx, "setup_complete", false), nil
}

// Complete runs the first-run setup wizard. It creates the initial admin user,
// applies competition and config settings, then marks setup as complete and
// returns a token pair for the newly created admin.
//
// Returns apperr.ErrSetupAlreadyComplete (409) if called on an already-configured
// platform. An application-level mutex serializes concurrent calls so that only
// one admin is created even under parallel requests.
func (uc *SetupUseCase) Complete(ctx context.Context, req *usecase.SetupRequest) (*usecase.SetupResult, error) {
	// Fast check without lock.
	if uc.deps.CompParamUC.GetBool(ctx, "setup_complete", false) {
		return nil, apperr.ErrSetupAlreadyComplete
	}

	uc.mu.Lock()
	defer uc.mu.Unlock()

	// Re-check under lock to prevent duplicate creates from concurrent requests.
	if uc.deps.CompParamUC.GetBool(ctx, "setup_complete", false) {
		return nil, apperr.ErrSetupAlreadyComplete
	}

	if _, err := setupCompetitionMode(req.Mode); err != nil {
		return nil, err
	}

	var result *usecase.SetupResult

	if err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		setupComplete, err := uc.deps.CompParamUC.GetForUpdate(ctx, "setup_complete")
		if err != nil {
			return fmt.Errorf("SetupUseCase - Complete - GetForUpdate setup_complete: %w", err)
		}

		if setupComplete.Value == "true" {
			return apperr.ErrSetupAlreadyComplete
		}

		adminUser, err := uc.deps.UserUC.AdminCreate(ctx, req.AdminUsername, req.AdminEmail, req.AdminPassword, "admin")
		if err != nil {
			return fmt.Errorf("SetupUseCase - Complete - AdminCreate: %w", err)
		}

		if err := uc.applyCompetition(ctx, req, adminUser); err != nil {
			return err
		}

		if err := uc.applyConfigs(ctx, req, adminUser); err != nil {
			return err
		}

		if err := uc.applySettings(ctx, req, adminUser); err != nil {
			return err
		}

		tokenPair, err := uc.deps.JWTService.GenerateTokenPair(ctx, adminUser.ID, string(adminUser.Role))
		if err != nil {
			return fmt.Errorf("SetupUseCase - Complete - GenerateTokenPair: %w", err)
		}

		if err := uc.deps.CompParamUC.Set(
			ctx,
			usecase.CompetitionParamSetParams{
				Key:         "setup_complete",
				Value:       "true",
				Description: "initial setup wizard completed",
				ValueType:   domain.CompetitionParamTypeBool,
				Category:    domain.ConfigCategoryGeneral,
				ActorID:     adminUser.ID,
				ClientIP:    req.ClientIP,
			},
		); err != nil {
			return fmt.Errorf("SetupUseCase - Complete - Set setup_complete: %w", err)
		}

		result = &usecase.SetupResult{
			TokenPair: &usecase.TokenPair{
				AccessToken:      tokenPair.AccessToken,
				RefreshToken:     tokenPair.RefreshToken,
				AccessExpiresAt:  tokenPair.AccessExpiresAt,
				RefreshExpiresAt: tokenPair.RefreshExpiresAt,
			},
			User: adminUser,
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf("SetupUseCase - Complete - TM.Run: %w", err)
	}

	return result, nil
}

func (uc *SetupUseCase) applyCompetition(ctx context.Context, req *usecase.SetupRequest, adminUser *domain.User) error {
	comp, err := uc.deps.CompUC.Get(ctx)
	if err != nil {
		return fmt.Errorf("SetupUseCase - applyCompetition - CompUC.Get: %w", err)
	}

	comp.Name = req.CTFName
	comp.StartTime = req.StartTime
	comp.EndTime = req.EndTime
	comp.FreezeTime = req.FreezeTime

	mode, err := setupCompetitionMode(req.Mode)
	if err != nil {
		return err
	}

	comp.Mode = mode

	if req.MaxTeamSize > 0 {
		comp.MaxTeamSize = req.MaxTeamSize
	}

	if err := uc.deps.CompUC.Update(ctx, comp, nil, adminUser.ID, req.ClientIP); err != nil {
		return fmt.Errorf("SetupUseCase - applyCompetition - CompUC.Update: %w", err)
	}

	return nil
}

func setupCompetitionMode(raw string) (domain.CompetitionMode, error) {
	mode := domain.CompetitionMode(raw)
	if !mode.IsValid() {
		return "", apperr.NewValidationErrorf("invalid competition mode %q: must be solo_only or teams_only", raw)
	}

	return mode, nil
}

func (uc *SetupUseCase) applyConfigs(ctx context.Context, req *usecase.SetupRequest, adminUser *domain.User) error {
	params := []*domain.CompetitionParam{
		{Key: "ctf_name", Value: req.CTFName, ValueType: domain.CompetitionParamTypeString, Category: domain.ConfigCategoryGeneral, Description: "CTF competition name"},
		{Key: "ctf_description", Value: req.CTFDescription, ValueType: domain.CompetitionParamTypeString, Category: domain.ConfigCategoryGeneral, Description: "CTF competition description (Markdown)"},
		{Key: "user_mode", Value: req.Mode, ValueType: domain.CompetitionParamTypeString, Category: domain.ConfigCategoryGeneral, Description: "Participation mode: teams_only or solo_only"},
		{Key: "challenge_visibility", Value: req.ChallengeVisibility, ValueType: domain.CompetitionParamTypeString, Category: domain.ConfigCategoryVisibility, Description: "Challenge visibility: public, private, admins"},
		{Key: "score_visibility", Value: req.ScoreVisibility, ValueType: domain.CompetitionParamTypeString, Category: domain.ConfigCategoryVisibility, Description: "Scoreboard visibility: public, private, hidden, admins, admins_only"},
		{Key: "account_visibility", Value: req.AccountVisibility, ValueType: domain.CompetitionParamTypeString, Category: domain.ConfigCategoryVisibility, Description: "User/team account visibility: public, private, admins"},
		{Key: "registration_visibility", Value: req.RegistrationVisibility, ValueType: domain.CompetitionParamTypeString, Category: domain.ConfigCategoryVisibility, Description: "Registration visibility: public, private"},
		{Key: "email_verification_required", Value: strconv.FormatBool(req.EmailVerificationRequired), ValueType: domain.CompetitionParamTypeBool, Category: domain.ConfigCategoryGeneral, Description: "Require email verification on signup"},
		{Key: "timezone", Value: req.Timezone, ValueType: domain.CompetitionParamTypeString, Category: domain.ConfigCategoryGeneral, Description: "Display timezone for competition times"},
	}

	if err := uc.deps.CompParamUC.SetBatch(ctx, params, adminUser.ID, req.ClientIP); err != nil {
		return fmt.Errorf("SetupUseCase - applyConfigs - SetBatch: %w", err)
	}

	return nil
}

// applySettings mirrors wizard inputs into the admin-editable app_settings row
// so /admin/settings reflects what was chosen during setup. Other fields keep
// their migration defaults.
func (uc *SetupUseCase) applySettings(ctx context.Context, req *usecase.SetupRequest, adminUser *domain.User) error {
	s, err := uc.deps.SettingsUC.Get(ctx)
	if err != nil {
		return fmt.Errorf("SetupUseCase - applySettings - SettingsUC.Get: %w", err)
	}

	s.AppName = req.CTFName
	s.VerifyEmails = req.EmailVerificationRequired

	if err := uc.deps.SettingsUC.Update(ctx, s, adminUser.ID, req.ClientIP); err != nil {
		return fmt.Errorf("SetupUseCase - applySettings - SettingsUC.Update: %w", err)
	}

	return nil
}
