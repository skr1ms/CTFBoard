package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/repo"
	"github.com/skr1ms/CTFBoard/internal/usecase/settings"
	"github.com/skr1ms/CTFBoard/pkg/jwt"
	"golang.org/x/crypto/bcrypt"
)

type UserUseCase struct {
	userRepo       repo.UserRepository
	teamRepo       repo.TeamRepository
	solveRepo      repo.SolveRepository
	txRepo         repo.TxRepository
	jwtService     jwt.Service
	fieldValidator *settings.FieldValidator
	fieldValueRepo repo.FieldValueRepository
}

func NewUserUseCase(
	userRepo repo.UserRepository,
	teamRepo repo.TeamRepository,
	solveRepo repo.SolveRepository,
	txRepo repo.TxRepository,
	jwtService jwt.Service,
	fieldValidator *settings.FieldValidator,
	fieldValueRepo repo.FieldValueRepository,
) *UserUseCase {
	return &UserUseCase{
		userRepo:       userRepo,
		teamRepo:       teamRepo,
		solveRepo:      solveRepo,
		txRepo:         txRepo,
		jwtService:     jwtService,
		fieldValidator: fieldValidator,
		fieldValueRepo: fieldValueRepo,
	}
}

//nolint:gocognit,gocyclo // validation, tx, custom fields
func (uc *UserUseCase) Register(ctx context.Context, username, email, password string, customFields map[string]string) (*entity.User, error) {
	if len(customFields) > 0 && uc.fieldValidator != nil {
		fieldValues := make(map[uuid.UUID]string)
		for k, v := range customFields {
			id, err := uuid.Parse(k)
			if err != nil {
				return nil, fmt.Errorf("invalid field ID %s: %w", k, err)
			}
			fieldValues[id] = v
		}
		if err := uc.fieldValidator.ValidateValues(ctx, entity.EntityTypeUser, fieldValues); err != nil {
			return nil, fmt.Errorf("custom fields validation: %w", err)
		}
	}

	_, err := uc.userRepo.GetByUsername(ctx, username)
	if err == nil {
		return nil, fmt.Errorf("%w: username", entityError.ErrUserAlreadyExists)
	}
	if !errors.Is(err, entityError.ErrUserNotFound) {
		return nil, fmt.Errorf("UserUseCase - Register - GetByUsername: %w", err)
	}

	_, err = uc.userRepo.GetByEmail(ctx, email)
	if err == nil {
		return nil, fmt.Errorf("%w: email", entityError.ErrUserAlreadyExists)
	}
	if !errors.Is(err, entityError.ErrUserNotFound) {
		return nil, fmt.Errorf("UserUseCase - Register - GetByEmail: %w", err)
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("UserUseCase - Register - GenerateFromPassword: %w", err)
	}

	user := &entity.User{
		Username:     username,
		Email:        email,
		PasswordHash: string(passwordHash),
		Role:         entity.RoleUser,
	}

	err = uc.txRepo.RunTransaction(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if err := uc.txRepo.CreateUserTx(ctx, tx, user); err != nil {
			return fmt.Errorf("CreateUserTx: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("UserUseCase - Register - Transaction: %w", err)
	}

	if len(customFields) > 0 && uc.fieldValueRepo != nil {
		if err := uc.fieldValueRepo.SetValues(ctx, user.ID, customFields); err != nil {
			return nil, fmt.Errorf("save custom fields: %w", err)
		}
	}

	return user, nil
}

func (uc *UserUseCase) Login(ctx context.Context, email, password string) (*jwt.TokenPair, error) {
	user, err := uc.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, entityError.ErrUserNotFound) {
			return nil, entityError.ErrInvalidCredentials
		}
		return nil, fmt.Errorf("UserUseCase - Login - GetByEmail: %w", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return nil, entityError.ErrInvalidCredentials
	}

	tokenPair, err := uc.jwtService.GenerateTokenPair(user.ID, user.Email, user.Username, user.Role)
	if err != nil {
		return nil, fmt.Errorf("UserUseCase - Login - GenerateTokenPair: %w", err)
	}

	return tokenPair, nil
}

func (uc *UserUseCase) GetByID(ctx context.Context, ID uuid.UUID) (*entity.User, error) {
	user, err := uc.userRepo.GetByID(ctx, ID)
	if err != nil {
		return nil, fmt.Errorf("UserUseCase - GetByID: %w", err)
	}
	return user, nil
}

type UserProfile struct {
	User   *entity.User
	Solves []*entity.Solve
}

func (uc *UserUseCase) GetProfile(ctx context.Context, userID uuid.UUID) (*UserProfile, error) {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("UserUseCase - GetProfile - GetByID: %w", err)
	}

	user.PasswordHash = ""

	solves, err := uc.solveRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("UserUseCase - GetProfile - GetByUserID: %w", err)
	}

	return &UserProfile{
		User:   user,
		Solves: solves,
	}, nil
}
