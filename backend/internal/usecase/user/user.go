package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/repo"
	"github.com/skr1ms/CTFBoard/internal/usecase/settings"
	"github.com/skr1ms/CTFBoard/pkg/jwt"
	"github.com/skr1ms/CTFBoard/pkg/usecaseutil"
	"golang.org/x/crypto/bcrypt"
)

type UserDeps struct {
	UserRepo       repo.UserRepository
	TeamRepo       repo.TeamRepository
	SolveRepo      repo.SolveRepository
	TxRepo         repo.TxRepository
	JWTService     jwt.Service
	FieldValidator *settings.FieldValidator
	FieldValueRepo repo.FieldValueRepository
}

type UserUseCase struct {
	deps UserDeps
}

func NewUserUseCase(deps UserDeps) *UserUseCase {
	return &UserUseCase{deps: deps}
}

func (uc *UserUseCase) Register(ctx context.Context, username, email, password string, customFields map[string]string) (*entity.User, error) {
	if err := uc.registerValidateCustomFields(ctx, customFields); err != nil {
		return nil, err
	}
	if err := uc.registerCheckUsernameAvailable(ctx, username); err != nil {
		return nil, err
	}
	if err := uc.registerCheckEmailAvailable(ctx, email); err != nil {
		return nil, err
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, usecaseutil.Wrap(err, "UserUseCase - Register - GenerateFromPassword")
	}
	user := &entity.User{
		Username:     username,
		Email:        email,
		PasswordHash: string(passwordHash),
		Role:         entity.RoleUser,
	}
	if err := uc.deps.TxRepo.RunTransaction(ctx, func(ctx context.Context, tx repo.Transaction) error {
		return uc.deps.TxRepo.CreateUserTx(ctx, tx, user)
	}); err != nil {
		return nil, usecaseutil.Wrap(err, "UserUseCase - Register - Transaction")
	}
	if err := uc.registerSetCustomFields(ctx, user.ID, customFields); err != nil {
		return nil, err
	}
	return user, nil
}

func (uc *UserUseCase) registerValidateCustomFields(ctx context.Context, customFields map[string]string) error {
	if len(customFields) == 0 || uc.deps.FieldValidator == nil {
		return nil
	}
	fieldValues := make(map[uuid.UUID]string)
	for k, v := range customFields {
		id, err := uuid.Parse(k)
		if err != nil {
			return fmt.Errorf("invalid field ID %s: %w", k, err)
		}
		fieldValues[id] = v
	}
	if err := uc.deps.FieldValidator.ValidateValues(ctx, entity.EntityTypeUser, fieldValues); err != nil {
		return fmt.Errorf("custom fields validation")
	}
	return nil
}

func (uc *UserUseCase) registerCheckUsernameAvailable(ctx context.Context, username string) error {
	_, err := uc.deps.UserRepo.GetByUsername(ctx, username)
	if err == nil {
		return fmt.Errorf("%w: username", entityError.ErrUserAlreadyExists)
	}
	if !errors.Is(err, entityError.ErrUserNotFound) {
		return usecaseutil.Wrap(err, "UserUseCase - Register - GetByUsername")
	}
	return nil
}

func (uc *UserUseCase) registerCheckEmailAvailable(ctx context.Context, email string) error {
	_, err := uc.deps.UserRepo.GetByEmail(ctx, email)
	if err == nil {
		return fmt.Errorf("%w: email", entityError.ErrUserAlreadyExists)
	}
	if !errors.Is(err, entityError.ErrUserNotFound) {
		return usecaseutil.Wrap(err, "UserUseCase - Register - GetByEmail")
	}
	return nil
}

func (uc *UserUseCase) registerSetCustomFields(ctx context.Context, userID uuid.UUID, customFields map[string]string) error {
	if len(customFields) == 0 || uc.deps.FieldValueRepo == nil {
		return nil
	}
	if err := uc.deps.FieldValueRepo.SetValues(ctx, userID, customFields); err != nil {
		return fmt.Errorf("save custom fields")
	}
	return nil
}

func (uc *UserUseCase) Login(ctx context.Context, email, password string) (*jwt.TokenPair, error) {
	user, err := uc.deps.UserRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, entityError.ErrUserNotFound) {
			return nil, entityError.ErrInvalidCredentials
		}
		return nil, usecaseutil.Wrap(err, "UserUseCase - Login - GetByEmail")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return nil, entityError.ErrInvalidCredentials
	}

	tokenPair, err := uc.deps.JWTService.GenerateTokenPair(user.ID, user.Email, user.Username, user.Role)
	if err != nil {
		return nil, usecaseutil.Wrap(err, "UserUseCase - Login - GenerateTokenPair")
	}

	return tokenPair, nil
}

func (uc *UserUseCase) GetByID(ctx context.Context, ID uuid.UUID) (*entity.User, error) {
	user, err := uc.deps.UserRepo.GetByID(ctx, ID)
	if err != nil {
		return nil, usecaseutil.Wrap(err, "UserUseCase - GetByID")
	}
	return user, nil
}

type UserProfile struct {
	User   *entity.User
	Solves []*entity.Solve
}

func (uc *UserUseCase) GetProfile(ctx context.Context, userID uuid.UUID) (*UserProfile, error) {
	user, err := uc.deps.UserRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, usecaseutil.Wrap(err, "UserUseCase - GetProfile - GetByID")
	}

	user.PasswordHash = ""

	solves, err := uc.deps.SolveRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, usecaseutil.Wrap(err, "UserUseCase - GetProfile - GetByUserID")
	}

	return &UserProfile{
		User:   user,
		Solves: solves,
	}, nil
}
