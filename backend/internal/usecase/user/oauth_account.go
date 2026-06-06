package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

// loginExistingOAuthUser refreshes the stored OAuth tokens for an already-linked
// account, then loads the user, checks ban status (IsBanned and WasInBannedTeam),
// and issues a new JWT token pair.
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

	if user.IsBanned {
		return nil, apperr.ErrInvalidCredentials
	}

	if user.WasInBannedTeam && user.Role != domain.RoleAdmin {
		return nil, apperr.ErrInvalidCredentials
	}

	if err := uc.deps.OAuthRepo.Upsert(ctx, oauthAcc); err != nil {
		return nil, fmt.Errorf("OAuthUseCase - loginExistingOAuthUser - Upsert: %w", err)
	}

	pair, err := uc.deps.JWTService.GenerateTokenPair(ctx, user.ID, string(user.Role))
	if err != nil {
		return nil, fmt.Errorf("OAuthUseCase - loginExistingOAuthUser - GenerateTokenPair: %w", err)
	}

	return tokenPairFromJWT(pair), nil
}

// linkOAuthToExistingUser attaches an OAuth provider to an existing account (same email). Returns JWT for that user.
func (uc *OAuthUseCase) linkOAuthToExistingUser(
	ctx context.Context,
	existingUser *domain.User,
	profile *domain.OAuthUserProfile,
	token *OAuthProviderToken,
	provider string,
) (*usecase.TokenPair, error) {
	if existingUser.IsBanned {
		return nil, apperr.ErrInvalidCredentials
	}

	if existingUser.WasInBannedTeam && existingUser.Role != domain.RoleAdmin {
		return nil, apperr.ErrInvalidCredentials
	}

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

	pair, err := uc.deps.JWTService.GenerateTokenPair(ctx, existingUser.ID, string(existingUser.Role))
	if err != nil {
		return nil, fmt.Errorf("OAuthUseCase - linkOAuthToExistingUser - GenerateTokenPair: %w", err)
	}

	return tokenPairFromJWT(pair), nil
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

	err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		settings, err := uc.deps.SettingsRepo.Get(ctx)
		if err != nil {
			return fmt.Errorf("OAuthUseCase - registerNewOAuthUser - SettingsRepo.Get: %w", err)
		}

		if !settings.RegistrationOpen {
			return apperr.ErrRegistrationClosed
		}

		desiredUsername, fallbackUsername := oauthUsernameCandidates(profile.Username, provider, profile.ID)

		locks := []registrationAdvisoryLock{
			{label: "email", key: registrationAdvisoryKey("reg:email:", profile.Email)},
			{label: "username", key: registrationAdvisoryKey("reg:username:", desiredUsername)},
		}
		if fallbackUsername != desiredUsername {
			locks = append(locks, registrationAdvisoryLock{label: "username_fallback", key: registrationAdvisoryKey("reg:username:", fallbackUsername)})
		}

		if err := acquireRegistrationAdvisoryLocks(ctx, uc.deps.UserRepo, "OAuthUseCase - registerNewOAuthUser", locks...); err != nil {
			return err
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
