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
	compRepo    repo.CompetitionRepository
	txRepo      repo.TxRepository
	maxTeamSize int
}

func NewTeamUseCase(teamRepo repo.TeamRepository, userRepo repo.UserRepository, compRepo repo.CompetitionRepository, txRepo repo.TxRepository) *TeamUseCase {
	return &TeamUseCase{
		teamRepo:    teamRepo,
		userRepo:    userRepo,
		compRepo:    compRepo,
		txRepo:      txRepo,
		maxTeamSize: DefaultMaxTeamSize,
	}
}

func NewTeamUseCaseWithSize(teamRepo repo.TeamRepository, userRepo repo.UserRepository, compRepo repo.CompetitionRepository, txRepo repo.TxRepository, maxTeamSize int) *TeamUseCase {
	return &TeamUseCase{
		teamRepo:    teamRepo,
		userRepo:    userRepo,
		compRepo:    compRepo,
		txRepo:      txRepo,
		maxTeamSize: maxTeamSize,
	}
}

func (uc *TeamUseCase) Create(ctx context.Context, name string, captainId uuid.UUID, isSolo bool, confirmReset bool) (*entity.Team, error) {
	comp, err := uc.compRepo.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("GetCompetition: %w", err)
	}

	mode := entity.CompetitionMode(comp.Mode)
	if !comp.AllowTeamSwitch {
		return nil, entityError.ErrRosterFrozen
	}
	if !mode.AllowsTeams() {
		return nil, entityError.ErrSoloModeNotAllowed
	}

	var team *entity.Team

	err = uc.txRepo.RunTransaction(ctx, func(ctx context.Context, tx pgx.Tx) error {
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
			if err := uc.handleSoloTeamCleanup(ctx, tx, user, captainId, confirmReset); err != nil {
				return err
			}
		}

		inviteToken := uuid.New()

		team = &entity.Team{
			Name:        name,
			InviteToken: inviteToken,
			CaptainId:   captainId,
			IsSolo:      isSolo,
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

func (uc *TeamUseCase) Join(ctx context.Context, inviteToken uuid.UUID, userId uuid.UUID, confirmReset bool) (*entity.Team, error) {
	comp, err := uc.compRepo.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("GetCompetition: %w", err)
	}
	if !comp.AllowTeamSwitch {
		return nil, entityError.ErrRosterFrozen
	}

	var team *entity.Team

	err = uc.txRepo.RunTransaction(ctx, func(ctx context.Context, tx pgx.Tx) error {
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
			if err := uc.handleSoloTeamCleanup(ctx, tx, user, userId, confirmReset); err != nil {
				return err
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
	comp, err := uc.compRepo.Get(ctx)
	if err != nil {
		return fmt.Errorf("GetCompetition: %w", err)
	}
	if !comp.AllowTeamSwitch {
		return entityError.ErrRosterFrozen
	}

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
	comp, err := uc.compRepo.Get(ctx)
	if err != nil {
		return fmt.Errorf("GetCompetition: %w", err)
	}
	if !comp.AllowTeamSwitch {
		return entityError.ErrRosterFrozen
	}

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
			Details: map[string]any{
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

func (uc *TeamUseCase) CreateSoloTeam(ctx context.Context, userId uuid.UUID, confirmReset bool) (*entity.Team, error) {
	comp, err := uc.compRepo.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("GetCompetition: %w", err)
	}

	mode := entity.CompetitionMode(comp.Mode)
	if !comp.AllowTeamSwitch {
		return nil, entityError.ErrRosterFrozen
	}
	if !mode.AllowsSolo() {
		return nil, entityError.ErrSoloModeNotAllowed
	}

	var team *entity.Team

	err = uc.txRepo.RunTransaction(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if err := uc.txRepo.LockUserTx(ctx, tx, userId); err != nil {
			return fmt.Errorf("LockUserTx: %w", err)
		}

		user, err := uc.userRepo.GetByID(ctx, userId)
		if err != nil {
			return fmt.Errorf("GetByID: %w", err)
		}

		if user.TeamId != nil {
			if err := uc.handleSoloTeamCleanup(ctx, tx, user, userId, confirmReset); err != nil {
				return err
			}
		}

		inviteToken := uuid.New()

		team = &entity.Team{
			Name:          user.Username,
			InviteToken:   inviteToken,
			CaptainId:     userId,
			IsSolo:        true,
			IsAutoCreated: false,
		}

		existingTeam, err := uc.txRepo.GetTeamByNameTx(ctx, tx, team.Name)
		if err == nil && existingTeam != nil {
			team.Name = fmt.Sprintf("%s (Solo)", user.Username)
		}

		if err := uc.txRepo.CreateTeamTx(ctx, tx, team); err != nil {
			return fmt.Errorf("CreateTeamTx: %w", err)
		}

		if err := uc.txRepo.UpdateUserTeamIDTx(ctx, tx, userId, &team.Id); err != nil {
			return fmt.Errorf("UpdateUserTeamIDTx: %w", err)
		}

		auditLog := &entity.TeamAuditLog{
			TeamId: team.Id,
			UserId: userId,
			Action: entity.TeamActionCreated,
			Details: map[string]any{
				"mode": "solo",
			},
		}
		if err := uc.txRepo.CreateTeamAuditLogTx(ctx, tx, auditLog); err != nil {
			return fmt.Errorf("CreateTeamAuditLogTx: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - CreateSoloTeam - Transaction: %w", err)
	}

	return team, nil
}

func (uc *TeamUseCase) handleSoloTeamCleanup(ctx context.Context, tx pgx.Tx, user *entity.User, actorId uuid.UUID, confirmReset bool) error {
	oldTeam, err := uc.txRepo.GetTeamByIDTx(ctx, tx, *user.TeamId)
	if err != nil {
		return fmt.Errorf("GetTeamByIDTx: %w", err)
	}

	members, err := uc.txRepo.GetUsersByTeamIDTx(ctx, tx, *user.TeamId)
	if err != nil {
		return fmt.Errorf("GetUsersByTeamIDTx: %w", err)
	}

	if len(members) == 1 && members[0].Id == user.Id && (oldTeam.IsSolo || oldTeam.IsAutoCreated) {
		if !confirmReset {
			return entityError.ErrConfirmationRequired
		}

		oldTeamId := *user.TeamId
		if err := uc.txRepo.SoftDeleteTeamTx(ctx, tx, oldTeamId); err != nil {
			return fmt.Errorf("SoftDeleteTeamTx: %w", err)
		}

		if err := uc.txRepo.DeleteSolvesByTeamIDTx(ctx, tx, oldTeamId); err != nil {
			return fmt.Errorf("DeleteSolvesByTeamIDTx: %w", err)
		}

		auditLog := &entity.TeamAuditLog{
			TeamId: oldTeamId,
			UserId: actorId,
			Action: entity.TeamActionDeleted,
			Details: map[string]any{
				"reason": "solo_team_cleanup",
			},
		}
		if err := uc.txRepo.CreateTeamAuditLogTx(ctx, tx, auditLog); err != nil {
			return fmt.Errorf("CreateTeamAuditLogTx: %w", err)
		}
		return nil
	}

	return entityError.ErrUserAlreadyInTeam
}

func (uc *TeamUseCase) DisbandTeam(ctx context.Context, captainId uuid.UUID) error {
	comp, err := uc.compRepo.Get(ctx)
	if err != nil {
		return fmt.Errorf("GetCompetition: %w", err)
	}
	if !comp.AllowTeamSwitch {
		return entityError.ErrRosterFrozen
	}

	return uc.txRepo.RunTransaction(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if err := uc.txRepo.LockUserTx(ctx, tx, captainId); err != nil {
			return fmt.Errorf("LockUserTx: %w", err)
		}

		user, err := uc.userRepo.GetByID(ctx, captainId)
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

		if team.CaptainId != captainId {
			return entityError.ErrNotCaptain
		}

		if err := uc.txRepo.SoftDeleteTeamTx(ctx, tx, team.Id); err != nil {
			return fmt.Errorf("SoftDeleteTeamTx: %w", err)
		}

		members, err := uc.txRepo.GetUsersByTeamIDTx(ctx, tx, team.Id)
		if err != nil {
			return fmt.Errorf("GetUsersByTeamIDTx: %w", err)
		}

		for _, member := range members {
			if err := uc.txRepo.UpdateUserTeamIDTx(ctx, tx, member.Id, nil); err != nil {
				return fmt.Errorf("UpdateUserTeamIDTx member %s: %w", member.Id, err)
			}
		}

		auditLog := &entity.TeamAuditLog{
			TeamId: team.Id,
			UserId: captainId,
			Action: entity.TeamActionDeleted,
			Details: map[string]any{
				"reason": "disbanded_by_captain",
			},
		}
		if err := uc.txRepo.CreateTeamAuditLogTx(ctx, tx, auditLog); err != nil {
			return fmt.Errorf("CreateTeamAuditLogTx: %w", err)
		}

		return nil
	})
}

func (uc *TeamUseCase) KickMember(ctx context.Context, captainId uuid.UUID, targetUserId uuid.UUID) error {
	comp, err := uc.compRepo.Get(ctx)
	if err != nil {
		return fmt.Errorf("GetCompetition: %w", err)
	}
	if !comp.AllowTeamSwitch {
		return entityError.ErrRosterFrozen
	}

	if captainId == targetUserId {
		return entityError.ErrCannotLeaveAsOnlyMember
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

		targetUser, err := uc.userRepo.GetByID(ctx, targetUserId)
		if err != nil {
			return fmt.Errorf("GetByID target: %w", err)
		}

		if targetUser.TeamId == nil || *targetUser.TeamId != team.Id {
			return entityError.ErrUserNotFound
		}

		if err := uc.txRepo.UpdateUserTeamIDTx(ctx, tx, targetUserId, nil); err != nil {
			return fmt.Errorf("UpdateUserTeamIDTx: %w", err)
		}

		auditLog := &entity.TeamAuditLog{
			TeamId: team.Id,
			UserId: captainId,
			Action: entity.TeamActionMemberKicked,
			Details: map[string]any{
				"target_user_id": targetUserId.String(),
			},
		}
		if err := uc.txRepo.CreateTeamAuditLogTx(ctx, tx, auditLog); err != nil {
			return fmt.Errorf("CreateTeamAuditLogTx: %w", err)
		}

		return nil
	})
}
