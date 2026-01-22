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
)

const DefaultMaxTeamSize = 10

type TeamUseCase struct {
	teamRepo    repo.TeamRepository
	userRepo    repo.UserRepository
	txRepo      repo.TxRepository
	maxTeamSize int
}

func NewTeamUseCase(teamRepo repo.TeamRepository, userRepo repo.UserRepository, txRepo repo.TxRepository) *TeamUseCase {
	return &TeamUseCase{
		teamRepo:    teamRepo,
		userRepo:    userRepo,
		txRepo:      txRepo,
		maxTeamSize: DefaultMaxTeamSize,
	}
}

func NewTeamUseCaseWithSize(teamRepo repo.TeamRepository, userRepo repo.UserRepository, txRepo repo.TxRepository, maxTeamSize int) *TeamUseCase {
	return &TeamUseCase{
		teamRepo:    teamRepo,
		userRepo:    userRepo,
		txRepo:      txRepo,
		maxTeamSize: maxTeamSize,
	}
}

func (uc *TeamUseCase) Create(ctx context.Context, name string, captainId uuid.UUID) (*entity.Team, error) {
	var team *entity.Team

	err := uc.txRepo.RunTransaction(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if err := uc.txRepo.LockUserTx(ctx, tx, captainId); err != nil {
			return fmt.Errorf("LockUserTx: %w", err)
		}

		_, err := uc.txRepo.GetTeamByNameTx(ctx, tx, name)
		if err == nil {
			return fmt.Errorf("%w: name", entityError.ErrTeamAlreadyExists)
		}
		if !errors.Is(err, entityError.ErrTeamNotFound) {
			return fmt.Errorf("GetTeamByNameTx: %w", err)
		}

		user, err := uc.userRepo.GetByID(ctx, captainId)
		if err != nil {
			return fmt.Errorf("GetByID: %w", err)
		}

		if user.TeamId != nil {
			members, err := uc.txRepo.GetUsersByTeamIDTx(ctx, tx, *user.TeamId)
			if err != nil {
				return fmt.Errorf("GetUsersByTeamIDTx: %w", err)
			}

			if len(members) == 1 && members[0].Id == user.Id {
				oldTeamId := *user.TeamId
				if err := uc.txRepo.SoftDeleteTeamTx(ctx, tx, oldTeamId); err != nil {
					return fmt.Errorf("SoftDeleteTeamTx: %w", err)
				}
				auditLog := &entity.TeamAuditLog{
					TeamId: oldTeamId,
					UserId: captainId,
					Action: entity.TeamActionDeleted,
					Details: map[string]interface{}{
						"reason": "solo_team_replaced",
					},
				}
				if err := uc.txRepo.CreateTeamAuditLogTx(ctx, tx, auditLog); err != nil {
					return fmt.Errorf("CreateTeamAuditLogTx: %w", err)
				}
			} else {
				return entityError.ErrUserAlreadyInTeam
			}
		}

		inviteToken := uuid.New()

		team = &entity.Team{
			Name:        name,
			InviteToken: inviteToken,
			CaptainId:   captainId,
		}

		if err := uc.txRepo.CreateTeamTx(ctx, tx, team); err != nil {
			return fmt.Errorf("CreateTeamTx: %w", err)
		}

		if err := uc.txRepo.UpdateUserTeamIDTx(ctx, tx, captainId, &team.Id); err != nil {
			return fmt.Errorf("UpdateUserTeamIDTx: %w", err)
		}

		auditLog := &entity.TeamAuditLog{
			TeamId: team.Id,
			UserId: captainId,
			Action: entity.TeamActionCreated,
		}
		if err := uc.txRepo.CreateTeamAuditLogTx(ctx, tx, auditLog); err != nil {
			return fmt.Errorf("CreateTeamAuditLogTx: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - Create - Transaction: %w", err)
	}

	return team, nil
}

func (uc *TeamUseCase) Join(ctx context.Context, inviteToken uuid.UUID, userId uuid.UUID) (*entity.Team, error) {
	var team *entity.Team

	err := uc.txRepo.RunTransaction(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if err := uc.txRepo.LockUserTx(ctx, tx, userId); err != nil {
			return fmt.Errorf("LockUserTx: %w", err)
		}

		var err error
		team, err = uc.txRepo.GetTeamByInviteTokenTx(ctx, tx, inviteToken)
		if err != nil {
			return fmt.Errorf("GetTeamByInviteTokenTx: %w", err)
		}

		if err := uc.txRepo.LockTeamTx(ctx, tx, team.Id); err != nil {
			return fmt.Errorf("LockTeamTx: %w", err)
		}

		targetMembers, err := uc.txRepo.GetUsersByTeamIDTx(ctx, tx, team.Id)
		if err != nil {
			return fmt.Errorf("GetUsersByTeamIDTx target: %w", err)
		}

		if len(targetMembers) >= uc.maxTeamSize {
			return entityError.ErrTeamFull
		}

		user, err := uc.userRepo.GetByID(ctx, userId)
		if err != nil {
			return fmt.Errorf("GetByID: %w", err)
		}

		if user.TeamId != nil {
			members, err := uc.txRepo.GetUsersByTeamIDTx(ctx, tx, *user.TeamId)
			if err != nil {
				return fmt.Errorf("GetUsersByTeamIDTx: %w", err)
			}

			if len(members) == 1 && members[0].Id == user.Id {
				oldTeamId := *user.TeamId
				if err := uc.txRepo.SoftDeleteTeamTx(ctx, tx, oldTeamId); err != nil {
					return fmt.Errorf("SoftDeleteTeamTx: %w", err)
				}
				auditLog := &entity.TeamAuditLog{
					TeamId: oldTeamId,
					UserId: userId,
					Action: entity.TeamActionDeleted,
					Details: map[string]interface{}{
						"reason": "solo_team_replaced",
					},
				}
				if err := uc.txRepo.CreateTeamAuditLogTx(ctx, tx, auditLog); err != nil {
					return fmt.Errorf("CreateTeamAuditLogTx: %w", err)
				}
			} else {
				return entityError.ErrUserAlreadyInTeam
			}
		}

		if err := uc.txRepo.UpdateUserTeamIDTx(ctx, tx, userId, &team.Id); err != nil {
			return fmt.Errorf("UpdateUserTeamIDTx: %w", err)
		}

		auditLog := &entity.TeamAuditLog{
			TeamId: team.Id,
			UserId: userId,
			Action: entity.TeamActionJoined,
		}
		if err := uc.txRepo.CreateTeamAuditLogTx(ctx, tx, auditLog); err != nil {
			return fmt.Errorf("CreateTeamAuditLogTx: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - Join - Transaction: %w", err)
	}

	return team, nil
}

func (uc *TeamUseCase) Leave(ctx context.Context, userId uuid.UUID) error {
	return uc.txRepo.RunTransaction(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if err := uc.txRepo.LockUserTx(ctx, tx, userId); err != nil {
			return fmt.Errorf("LockUserTx: %w", err)
		}

		user, err := uc.userRepo.GetByID(ctx, userId)
		if err != nil {
			return fmt.Errorf("GetByID: %w", err)
		}

		if user.TeamId == nil {
			return entityError.ErrTeamNotFound
		}

		if err := uc.txRepo.LockTeamTx(ctx, tx, *user.TeamId); err != nil {
			return fmt.Errorf("LockTeamTx: %w", err)
		}

		team, err := uc.teamRepo.GetByID(ctx, *user.TeamId)
		if err != nil {
			return fmt.Errorf("GetByID team: %w", err)
		}

		members, err := uc.txRepo.GetUsersByTeamIDTx(ctx, tx, *user.TeamId)
		if err != nil {
			return fmt.Errorf("GetUsersByTeamIDTx: %w", err)
		}

		if len(members) == 1 {
			return entityError.ErrCannotLeaveAsOnlyMember
		}

		if team.CaptainId == userId {
			return entityError.ErrNotCaptain
		}

		if err := uc.txRepo.UpdateUserTeamIDTx(ctx, tx, userId, nil); err != nil {
			return fmt.Errorf("UpdateUserTeamIDTx: %w", err)
		}

		auditLog := &entity.TeamAuditLog{
			TeamId: team.Id,
			UserId: userId,
			Action: entity.TeamActionLeft,
		}
		if err := uc.txRepo.CreateTeamAuditLogTx(ctx, tx, auditLog); err != nil {
			return fmt.Errorf("CreateTeamAuditLogTx: %w", err)
		}

		return nil
	})
}

func (uc *TeamUseCase) TransferCaptain(ctx context.Context, captainId uuid.UUID, newCaptainId uuid.UUID) error {
	if captainId == newCaptainId {
		return entityError.ErrCannotTransferToSelf
	}

	return uc.txRepo.RunTransaction(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if err := uc.txRepo.LockUserTx(ctx, tx, captainId); err != nil {
			return fmt.Errorf("LockUserTx captain: %w", err)
		}

		captain, err := uc.userRepo.GetByID(ctx, captainId)
		if err != nil {
			return fmt.Errorf("GetByID captain: %w", err)
		}

		if captain.TeamId == nil {
			return entityError.ErrTeamNotFound
		}

		if err := uc.txRepo.LockTeamTx(ctx, tx, *captain.TeamId); err != nil {
			return fmt.Errorf("LockTeamTx: %w", err)
		}

		team, err := uc.teamRepo.GetByID(ctx, *captain.TeamId)
		if err != nil {
			return fmt.Errorf("GetByID team: %w", err)
		}

		if team.CaptainId != captainId {
			return entityError.ErrNotCaptain
		}

		newCaptain, err := uc.userRepo.GetByID(ctx, newCaptainId)
		if err != nil {
			return fmt.Errorf("GetByID newCaptain: %w", err)
		}

		if newCaptain.TeamId == nil || *newCaptain.TeamId != team.Id {
			return entityError.ErrNewCaptainNotInTeam
		}

		if err := uc.txRepo.UpdateTeamCaptainTx(ctx, tx, team.Id, newCaptainId); err != nil {
			return fmt.Errorf("UpdateTeamCaptainTx: %w", err)
		}

		auditLog := &entity.TeamAuditLog{
			TeamId: team.Id,
			UserId: captainId,
			Action: entity.TeamActionCaptainTransfer,
			Details: map[string]interface{}{
				"from": captainId.String(),
				"to":   newCaptainId.String(),
			},
		}
		if err := uc.txRepo.CreateTeamAuditLogTx(ctx, tx, auditLog); err != nil {
			return fmt.Errorf("CreateTeamAuditLogTx: %w", err)
		}

		return nil
	})
}

func (uc *TeamUseCase) GetByID(ctx context.Context, id uuid.UUID) (*entity.Team, error) {
	team, err := uc.teamRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - GetByID: %w", err)
	}
	return team, nil
}

func (uc *TeamUseCase) GetMyTeam(ctx context.Context, userId uuid.UUID) (*entity.Team, []*entity.User, error) {
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

func (uc *TeamUseCase) GetTeamMembers(ctx context.Context, teamId uuid.UUID) ([]*entity.User, error) {
	users, err := uc.userRepo.GetByTeamId(ctx, teamId)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - GetTeamMembers: %w", err)
	}
	return users, nil
}
