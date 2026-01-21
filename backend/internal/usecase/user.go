package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/repo"
	"github.com/skr1ms/CTFBoard/pkg/jwt"
	"github.com/skr1ms/CTFBoard/pkg/validator"
	"golang.org/x/crypto/bcrypt"
)

type UserUseCase struct {
	userRepo   repo.UserRepository
	teamRepo   repo.TeamRepository
	solveRepo  repo.SolveRepository
	txRepo     repo.TxRepository
	jwtService jwt.Service
	validator  validator.Validator
}

func NewUserUseCase(
	userRepo repo.UserRepository,
	teamRepo repo.TeamRepository,
	solveRepo repo.SolveRepository,
	txRepo repo.TxRepository,
	jwtService jwt.Service,
) *UserUseCase {
	return &UserUseCase{
		userRepo:   userRepo,
		teamRepo:   teamRepo,
		solveRepo:  solveRepo,
		txRepo:     txRepo,
		jwtService: jwtService,
	}
}

func (uc *UserUseCase) Register(ctx context.Context, username, email, password string) (*entity.User, error) {
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

	inviteToken := uuid.New()

	user := &entity.User{
		Username:     username,
		Email:        email,
		PasswordHash: string(passwordHash),
		Role:         "user",
	}

	var team *entity.Team

	err = uc.txRepo.RunTransaction(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if err := uc.txRepo.CreateUserTx(ctx, tx, user); err != nil {
			return fmt.Errorf("CreateUserTx: %w", err)
		}

		team = &entity.Team{
			Name:        username,
			InviteToken: inviteToken,
			CaptainId:   user.Id,
		}

		if err := uc.txRepo.CreateTeamTx(ctx, tx, team); err != nil {
			return fmt.Errorf("CreateTeamTx: %w", err)
		}

		if err := uc.txRepo.UpdateUserTeamIDTx(ctx, tx, user.Id, &team.Id); err != nil {
			return fmt.Errorf("UpdateUserTeamIDTx: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("UserUseCase - Register - Transaction: %w", err)
	}

	user.TeamId = &team.Id

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

	tokenPair, err := uc.jwtService.GenerateTokenPair(user.Id, user.Email, user.Username, user.Role)
	if err != nil {
		return nil, fmt.Errorf("UserUseCase - Login - GenerateTokenPair: %w", err)
	}

	return tokenPair, nil
}

func (uc *UserUseCase) GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	user, err := uc.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("UserUseCase - GetByID: %w", err)
	}
	return user, nil
}

type UserProfile struct {
	User   *entity.User
	Solves []*entity.Solve
}

func (uc *UserUseCase) GetProfile(ctx context.Context, userId uuid.UUID) (*UserProfile, error) {
	user, err := uc.userRepo.GetByID(ctx, userId)
	if err != nil {
		return nil, fmt.Errorf("UserUseCase - GetProfile - GetByID: %w", err)
	}

	user.PasswordHash = ""

	solves, err := uc.solveRepo.GetByUserId(ctx, userId)
	if err != nil {
		return nil, fmt.Errorf("UserUseCase - GetProfile - GetByUserId: %w", err)
	}

	return &UserProfile{
		User:   user,
		Solves: solves,
	}, nil
}
