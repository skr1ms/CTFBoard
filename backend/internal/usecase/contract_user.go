package usecase

import (
	"context"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

// =============================================================================
// User
// =============================================================================

type (
	// UserProfile is a user profile view combining the user record with their solve history.
	UserProfile struct {
		User         *domain.User
		Solves       []*domain.Solve
		CustomFields CustomFieldValues
	}

	UserMe struct {
		User         *domain.User
		CustomFields CustomFieldValues
	}

	UserRegisterParams struct {
		Username         string
		Email            string
		Password         string
		RegistrationCode string
		CustomFields     CustomFieldValues
	}

	UserProfileUpdateParams struct {
		UserID          uuid.UUID
		Username        *string
		Email           *string
		CurrentPassword *string
		NewPassword     *string
		CustomFields    *CustomFieldValues
	}

	// UserUseCase handles user registration, authentication, profile management, and admin operations.
	UserUseCase interface {
		Register(ctx context.Context, params UserRegisterParams) (*domain.User, error)
		Login(ctx context.Context, email, password string) (*TokenPair, error)
		RefreshTokenPair(ctx context.Context, refreshToken string) (*TokenPair, error)
		Logout(ctx context.Context, refreshToken string, accessToken *string) error
		GetMe(ctx context.Context, userID uuid.UUID) (*UserMe, error)
		GetByID(ctx context.Context, ID uuid.UUID) (*domain.User, error)
		GetProfile(ctx context.Context, userID uuid.UUID) (*UserProfile, error)
		ListUsers(ctx context.Context, search *string, field string, page, perPage int) (*Paginated[*domain.User], error)
		GetUserSolves(ctx context.Context, userID uuid.UUID) ([]*domain.SolveWithDetails, error)
		GetUserFails(ctx context.Context, userID uuid.UUID, page, perPage int) (*Paginated[*domain.SubmissionWithDetails], error)
		GetUserAwards(ctx context.Context, userID uuid.UUID) ([]*domain.Award, error)
		AdminCreate(ctx context.Context, username, email, password, role string) (*domain.User, error)
		AdminUpdate(ctx context.Context, userID uuid.UUID, username, email, role, password *string, isVerified *bool) (*domain.User, error)
		AdminDelete(ctx context.Context, userID, actorID uuid.UUID) error
		BanUser(ctx context.Context, userID uuid.UUID, reason string, actorID uuid.UUID) error
		UnbanUser(ctx context.Context, userID, actorID uuid.UUID) error
		UpdateProfile(ctx context.Context, params UserProfileUpdateParams) (*UserMe, error)
		GetMySubmissions(ctx context.Context, userID uuid.UUID, page, perPage int) (*Paginated[*domain.SubmissionWithDetails], error)
	}
)

// =============================================================================
// BanAppeal
// =============================================================================

type (
	// BanAppealUseCase handles creation and review of ban appeals.
	BanAppealUseCase interface {
		CreateAppeal(ctx context.Context, userID uuid.UUID, message string) (*domain.BanAppeal, error)
		GetAppealsByUser(ctx context.Context, userID uuid.UUID) ([]*domain.BanAppeal, error)
		ListAppeals(ctx context.Context, decision *domain.AppealDecision, page, perPage int) (*Paginated[*domain.BanAppeal], error)
		ReviewAppeal(ctx context.Context, appealID uuid.UUID, decision domain.AppealDecision, adminResponse *string, actorID uuid.UUID) (*domain.BanAppeal, error)
		// CanAppeal reports whether a banned user is eligible to submit a new appeal
		// (no pending appeal and outside the cooldown window).
		CanAppeal(ctx context.Context, userID uuid.UUID) (canAppeal bool, hasPending bool, err error)
	}
)

// =============================================================================
// OAuth
// =============================================================================

type (
	// OAuthUseCase handles OAuth2 authorization URL generation, state validation, and callback processing.
	OAuthUseCase interface {
		GetAuthURL(ctx context.Context, provider string) (authURL, state string, err error)
		ValidateState(cookieState, queryState string) bool
		HandleCallback(ctx context.Context, provider, code string) (*TokenPair, error)
		IssueExchangeCode(ctx context.Context, pair *TokenPair) (string, error)
		ConsumeExchangeCode(ctx context.Context, code string) (*TokenPair, error)
	}
)
