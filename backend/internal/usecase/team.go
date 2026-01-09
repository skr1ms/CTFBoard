package usecase

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/repo"
)

type TeamUseCase struct {
	teamRepo repo.TeamRepository
	userRepo repo.UserRepository
}

func NewTeamUseCase(teamRepo repo.TeamRepository, userRepo repo.UserRepository) *TeamUseCase {
	return &TeamUseCase{
		teamRepo: teamRepo,
		userRepo: userRepo,
	}
}

func (uc *TeamUseCase) Create(ctx context.Context, name, captainId string) (*entity.Team, error) {
	_, err := uc.teamRepo.GetByName(ctx, name)
	if err == nil {
		return nil, fmt.Errorf("%w: name", entityError.ErrTeamAlreadyExists)
	}
	if !errors.Is(err, entityError.ErrTeamNotFound) {
		return nil, fmt.Errorf("TeamUseCase - Create - GetByName: %w", err)
	}

	user, err := uc.userRepo.GetByID(ctx, captainId)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - Create - GetByID: %w", err)
	}

	if user.TeamId != nil {
		return nil, entityError.ErrUserAlreadyInTeam
	}

	inviteTokenBytes := make([]byte, 16)
	if _, err := rand.Read(inviteTokenBytes); err != nil {
		return nil, fmt.Errorf("TeamUseCase - Create - GenerateToken: %w", err)
	}
	inviteToken := hex.EncodeToString(inviteTokenBytes)

	team := &entity.Team{
		Name:        name,
		InviteToken: inviteToken,
		CaptainId:   captainId,
	}

	err = uc.teamRepo.Create(ctx, team)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - Create - Create: %w", err)
	}

	err = uc.userRepo.UpdateTeamId(ctx, captainId, &team.Id)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - Create - UpdateTeamId: %w", err)
	}

	return team, nil
}

func (uc *TeamUseCase) Join(ctx context.Context, inviteToken, userId string) (*entity.Team, error) {
	team, err := uc.teamRepo.GetByInviteToken(ctx, inviteToken)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - Join - GetByInviteToken: %w", err)
	}

	user, err := uc.userRepo.GetByID(ctx, userId)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - Join - GetByID: %w", err)
	}

	if user.TeamId != nil {
		return nil, entityError.ErrUserAlreadyInTeam
	}

	err = uc.userRepo.UpdateTeamId(ctx, userId, &team.Id)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - Join - UpdateTeamId: %w", err)
	}

	return team, nil
}

func (uc *TeamUseCase) GetByID(ctx context.Context, id string) (*entity.Team, error) {
	team, err := uc.teamRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - GetByID: %w", err)
	}
	return team, nil
}

func (uc *TeamUseCase) GetMyTeam(ctx context.Context, userId string) (*entity.Team, []*entity.User, error) {
	user, err := uc.userRepo.GetByID(ctx, userId)
	if err != nil {
		return nil, nil, fmt.Errorf("TeamUseCase - GetMyTeam - GetByID: %w", err)
	}

	if user.TeamId == nil {
		return nil, nil, entityError.ErrTeamNotFound
	}

	team, err := uc.teamRepo.GetByID(ctx, *user.TeamId)
	if err != nil {
		return nil, nil, fmt.Errorf("TeamUseCase - GetMyTeam - GetByID: %w", err)
	}

	members, err := uc.userRepo.GetByTeamId(ctx, *user.TeamId)
	if err != nil {
		return nil, nil, fmt.Errorf("TeamUseCase - GetMyTeam - GetByTeamId: %w", err)
	}

	return team, members, nil
}

func (uc *TeamUseCase) GetTeamMembers(ctx context.Context, teamId string) ([]*entity.User, error) {
	users, err := uc.userRepo.GetByTeamId(ctx, teamId)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - GetTeamMembers: %w", err)
	}
	return users, nil
}
