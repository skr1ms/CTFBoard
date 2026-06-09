package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

// loginExistingOAuthUser refreshes the stored OAuth tokens for an already-linked
// account, then loads the user and issues a JWT token pair. Directly banned
// users and former members of banned teams may still receive tokens for /auth/me
// and appeal state. Protected CTF actions remain blocked elsewhere.
func (uc *OAuthUseCase) loginExistingOAuthUser(
	ctx context.Context,
	oauthAcc *domain.OAuthAccount,
	token *OAuthProviderToken,
	_ string,
) (*usecase.TokenPair, error) {
	oauthAcc.AccessToken = token.AccessToken

	rt := token.RefreshToken
	if rt != "" {
		oauthAcc.RefreshToken = &rt
	}

	if !token.ExpiresAt.IsZero() {
		oauthAcc.ExpiresAt = &token.ExpiresAt
	}

	user, err := uc.deps.UserRepo.GetByID(ctx, oauthAcc.UserID)
	if err != nil {
		return nil, fmt.Errorf("OAuthUseCase - loginExistingOAuthUser - GetByID: %w", err)
	}

	if err := uc.deps.OAuthRepo.Upsert(ctx, oauthAcc); err != nil {
		return nil, fmt.Errorf("OAuthUseCase - loginExistingOAuthUser - Upsert: %w", err)
	}

	return issueUserTokenPair(ctx, uc.deps.JWTService, user, "OAuthUseCase - loginExistingOAuthUser")
}

// linkOAuthToExistingUser attaches an OAuth provider to an existing account (same email). Returns JWT for that user.
func (uc *OAuthUseCase) linkOAuthToExistingUser(
	ctx context.Context,
	existingUser *domain.User,
	profile *domain.OAuthUserProfile,
	token *OAuthProviderToken,
	provider string,
) (*usecase.TokenPair, error) {
	oauthAcc := &domain.OAuthAccount{
		UserID:         existingUser.ID,
		Provider:       provider,
		ProviderUserID: profile.ID,
		AccessToken:    token.AccessToken,
	}
	if token.RefreshToken != "" {
		oauthAcc.RefreshToken = &token.RefreshToken
	}

	if !token.ExpiresAt.IsZero() {
		oauthAcc.ExpiresAt = &token.ExpiresAt
	}

	err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		err := uc.deps.OAuthRepo.Upsert(ctx, oauthAcc)
		if err != nil {
			return fmt.Errorf("OAuthUseCase - linkOAuthToExistingUser - OAuthRepo.Upsert: %w", err)
		}
		// In solo_only mode an existing user without a team (e.g. registered before
		// the mode was set to solo_only) must receive a solo team on first OAuth login
		// so they can submit flags
		if existingUser.TeamID == nil {
			if err := ensureSoloTeamIfRequired(ctx, uc.deps.CompRepo, uc.deps.SoloTeamCreator, existingUser); err != nil {
				return fmt.Errorf("OAuthUseCase - linkOAuthToExistingUser - ensureSoloTeamIfRequired: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("OAuthUseCase - linkOAuthToExistingUser - Transaction: %w", err)
	}

	return issueUserTokenPair(ctx, uc.deps.JWTService, existingUser, "OAuthUseCase - linkOAuthToExistingUser")
}

// registerNewOAuthUser creates a brand-new user account from the OAuth provider
// profile data inside a single transaction. It checks that registration is open,
// verifies that no local account already exists for the provider email, and
// resolves a unique username by first trying the provider-supplied name, then
// a provider-scoped fallback that embeds the provider user ID to avoid collisions
// The user is created with the email pre-verified and an OAuth-only password
// sentinel. In solo_only competition mode a solo team is created within the same
// transaction. On success a JWT token pair is issued.
func (uc *OAuthUseCase) registerNewOAuthUser(
	ctx context.Context,
	profile *domain.OAuthUserProfile,
	token *OAuthProviderToken,
	provider string,
) (*usecase.TokenPair, error) {
	settings, err := uc.deps.SettingsRepo.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("OAuthUseCase - registerNewOAuthUser - SettingsRepo.Get: %w", err)
	}

	if err := preflightRegistrationPolicy(ctx, registrationPolicyDeps{
		UserRepo:    uc.deps.UserRepo,
		CompParamUC: uc.deps.CompParamUC,
	}, settings, registrationPolicyOAuth, ""); err != nil {
		return nil, fmt.Errorf("OAuthUseCase - registerNewOAuthUser - preflightRegistrationPolicy: %w", err)
	}

	user := &domain.User{
		Email:        profile.Email,
		PasswordHash: domain.OAuthOnlyPasswordSentinel,
		Role:         domain.RoleUser,
		IsVerified:   true,
	}

	oauthAcc := &domain.OAuthAccount{
		Provider:       provider,
		ProviderUserID: profile.ID,
		AccessToken:    token.AccessToken,
	}
	if token.RefreshToken != "" {
		oauthAcc.RefreshToken = &token.RefreshToken
	}

	if !token.ExpiresAt.IsZero() {
		oauthAcc.ExpiresAt = &token.ExpiresAt
	}

	err = uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		txSettings, err := uc.deps.SettingsRepo.Get(ctx)
		if err != nil {
			return fmt.Errorf("OAuthUseCase - registerNewOAuthUser - SettingsRepo.Get: %w", err)
		}

		if err := enforceRegistrationPolicy(ctx, registrationPolicyDeps{
			UserRepo:    uc.deps.UserRepo,
			CompParamUC: uc.deps.CompParamUC,
		}, txSettings, registrationPolicyOAuth, ""); err != nil {
			return fmt.Errorf("OAuthUseCase - registerNewOAuthUser - enforceRegistrationPolicy: %w", err)
		}

		desiredUsername, fallbackUsername := oauthUsernameCandidates(profile.Username, provider, profile.ID)

		locks := []repo.RegistrationAdvisoryLock{
			{Label: "email", Scope: repo.RegistrationLockEmail, Value: profile.Email},
			{Label: "username", Scope: repo.RegistrationLockUsername, Value: desiredUsername},
		}
		if fallbackUsername != desiredUsername {
			locks = append(locks, repo.RegistrationAdvisoryLock{Label: "username_fallback", Scope: repo.RegistrationLockUsername, Value: fallbackUsername})
		}

		if err := repo.AcquireRegistrationAdvisoryLocks(ctx, uc.deps.UserRepo, locks...); err != nil {
			return fmt.Errorf("OAuthUseCase - registerNewOAuthUser - %w", err)
		}

		existing, err := uc.deps.UserRepo.GetByEmail(ctx, profile.Email)
		if err != nil && !errors.Is(err, apperr.ErrUserNotFound) {
			return fmt.Errorf("OAuthUseCase - registerNewOAuthUser - UserRepo.GetByEmail: %w", err)
		}

		if existing != nil {
			return apperr.ErrUserAlreadyExists
		}

		username, err := uc.resolveUsername(ctx, profile.Username, provider, profile.ID)
		if err != nil {
			return fmt.Errorf("OAuthUseCase - registerNewOAuthUser - resolveUsername: %w", err)
		}

		user.Username = username
		if err := uc.deps.UserRepo.Create(ctx, user); err != nil {
			return fmt.Errorf("OAuthUseCase - registerNewOAuthUser - UserRepo.Create: %w", err)
		}

		oauthAcc.UserID = user.ID
		if err := uc.deps.OAuthRepo.Upsert(ctx, oauthAcc); err != nil {
			return fmt.Errorf("OAuthUseCase - registerNewOAuthUser - OAuthRepo.Upsert: %w", err)
		}
		// Solo team creation is inside the transaction so a failure rolls back
		// the entire registration. CreateSoloTeamForNewUser reuses this tx
		if err := ensureSoloTeamIfRequired(ctx, uc.deps.CompRepo, uc.deps.SoloTeamCreator, user); err != nil {
			return fmt.Errorf("OAuthUseCase - registerNewOAuthUser - ensureSoloTeamIfRequired: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("OAuthUseCase - registerNewOAuthUser - Transaction: %w", err)
	}

	pair, err := uc.deps.JWTService.GenerateTokenPair(ctx, user.ID, string(user.Role))
	if err != nil {
		return nil, fmt.Errorf("OAuthUseCase - registerNewOAuthUser - GenerateTokenPair: %w", err)
	}

	return tokenPairFromJWT(pair), nil
}
