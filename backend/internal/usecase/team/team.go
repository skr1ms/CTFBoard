package team

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

//nolint:gocognit,gocyclo
func (uc *TeamUseCase) Create(ctx context.Context, name string, captainID uuid.UUID, isSolo, confirmReset bool) (*entity.Team, error) {
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
		if err := uc.txRepo.LockUserTx(ctx, tx, captainID); err != nil {
			return fmt.Errorf("LockUserTx: %w", err)
		}

		_, err := uc.txRepo.GetTeamByNameTx(ctx, tx, name)
		if err == nil {
			return fmt.Errorf("%w: name", entityError.ErrTeamAlreadyExists)
		}
		if !errors.Is(err, entityError.ErrTeamNotFound) {
			return fmt.Errorf("GetTeamByNameTx: %w", err)
		}

		user, err := uc.userRepo.GetByID(ctx, captainID)
		if err != nil {
			return fmt.Errorf("GetByID: %w", err)
		}

		if user.TeamID != nil {
			if err := uc.handleSoloTeamCleanup(ctx, tx, user, captainID, confirmReset); err != nil {
				return err
			}
		}

		inviteToken := uuid.New()

		team = &entity.Team{
			Name:        name,
			InviteToken: inviteToken,
			CaptainID:   captainID,
			IsSolo:      isSolo,
		}

		if err := uc.txRepo.CreateTeamTx(ctx, tx, team); err != nil {
			return fmt.Errorf("CreateTeamTx: %w", err)
		}

		if err := uc.txRepo.UpdateUserTeamIDTx(ctx, tx, captainID, &team.ID); err != nil {
			return fmt.Errorf("UpdateUserTeamIDTx: %w", err)
		}

		auditLog := &entity.TeamAuditLog{
			TeamID: team.ID,
			UserID: captainID,
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

//nolint:gocognit,gocyclo
func (uc *TeamUseCase) Join(ctx context.Context, inviteToken, userID uuid.UUID, confirmReset bool) (*entity.Team, error) {
	comp, err := uc.compRepo.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("GetCompetition: %w", err)
	}
	if !comp.AllowTeamSwitch {
		return nil, entityError.ErrRosterFrozen
	}

	var team *entity.Team

	err = uc.txRepo.RunTransaction(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if err := uc.txRepo.LockUserTx(ctx, tx, userID); err != nil {
			return fmt.Errorf("LockUserTx: %w", err)
		}

		var err error
		team, err = uc.txRepo.GetTeamByInviteTokenTx(ctx, tx, inviteToken)
		if err != nil {
			return fmt.Errorf("GetTeamByInviteTokenTx: %w", err)
		}

		if err := uc.txRepo.LockTeamTx(ctx, tx, team.ID); err != nil {
			return fmt.Errorf("LockTeamTx: %w", err)
		}

		targetMembers, err := uc.txRepo.GetUsersByTeamIDTx(ctx, tx, team.ID)
		if err != nil {
			return fmt.Errorf("GetUsersByTeamIDTx target: %w", err)
		}

		if len(targetMembers) >= uc.maxTeamSize {
			return entityError.ErrTeamFull
		}

		user, err := uc.userRepo.GetByID(ctx, userID)
		if err != nil {
			return fmt.Errorf("GetByID: %w", err)
		}

		if user.TeamID != nil {
			if err := uc.handleSoloTeamCleanup(ctx, tx, user, userID, confirmReset); err != nil {
				return err
			}
		}

		if err := uc.txRepo.UpdateUserTeamIDTx(ctx, tx, userID, &team.ID); err != nil {
			return fmt.Errorf("UpdateUserTeamIDTx: %w", err)
		}

		auditLog := &entity.TeamAuditLog{
			TeamID: team.ID,
			UserID: userID,
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

//nolint:gocognit,gocyclo
func (uc *TeamUseCase) Leave(ctx context.Context, userID uuid.UUID) error {
	comp, err := uc.compRepo.Get(ctx)
	if err != nil {
		return fmt.Errorf("GetCompetition: %w", err)
	}
	if !comp.AllowTeamSwitch {
		return entityError.ErrRosterFrozen
	}

	return uc.txRepo.RunTransaction(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if err := uc.txRepo.LockUserTx(ctx, tx, userID); err != nil {
			return fmt.Errorf("LockUserTx: %w", err)
		}

		user, err := uc.userRepo.GetByID(ctx, userID)
		if err != nil {
			return fmt.Errorf("GetByID: %w", err)
		}

		if user.TeamID == nil {
			return entityError.ErrTeamNotFound
		}

		if err := uc.txRepo.LockTeamTx(ctx, tx, *user.TeamID); err != nil {
			return fmt.Errorf("LockTeamTx: %w", err)
		}

		team, err := uc.teamRepo.GetByID(ctx, *user.TeamID)
		if err != nil {
			return fmt.Errorf("GetByID team: %w", err)
		}

		members, err := uc.txRepo.GetUsersByTeamIDTx(ctx, tx, *user.TeamID)
		if err != nil {
			return fmt.Errorf("GetUsersByTeamIDTx: %w", err)
		}

		if len(members) == 1 {
			return entityError.ErrCannotLeaveAsOnlyMember
		}

		if team.CaptainID == userID {
			return entityError.ErrNotCaptain
		}

		if err := uc.txRepo.UpdateUserTeamIDTx(ctx, tx, userID, nil); err != nil {
			return fmt.Errorf("UpdateUserTeamIDTx: %w", err)
		}

		auditLog := &entity.TeamAuditLog{
			TeamID: team.ID,
			UserID: userID,
			Action: entity.TeamActionLeft,
		}
		if err := uc.txRepo.CreateTeamAuditLogTx(ctx, tx, auditLog); err != nil {
			return fmt.Errorf("CreateTeamAuditLogTx: %w", err)
		}

		return nil
	})
}

//nolint:gocognit,gocyclo
func (uc *TeamUseCase) TransferCaptain(ctx context.Context, captainID, newCaptainID uuid.UUID) error {
	comp, err := uc.compRepo.Get(ctx)
	if err != nil {
		return fmt.Errorf("GetCompetition: %w", err)
	}
	if !comp.AllowTeamSwitch {
		return entityError.ErrRosterFrozen
	}

	if captainID == newCaptainID {
		return entityError.ErrCannotTransferToSelf
	}

	return uc.txRepo.RunTransaction(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if err := uc.txRepo.LockUserTx(ctx, tx, captainID); err != nil {
			return fmt.Errorf("LockUserTx captain: %w", err)
		}

		captain, err := uc.userRepo.GetByID(ctx, captainID)
		if err != nil {
			return fmt.Errorf("GetByID captain: %w", err)
		}

		if captain.TeamID == nil {
			return entityError.ErrTeamNotFound
		}

		if err := uc.txRepo.LockTeamTx(ctx, tx, *captain.TeamID); err != nil {
			return fmt.Errorf("LockTeamTx: %w", err)
		}

		team, err := uc.teamRepo.GetByID(ctx, *captain.TeamID)
		if err != nil {
			return fmt.Errorf("GetByID team: %w", err)
		}

		if team.CaptainID != captainID {
			return entityError.ErrNotCaptain
		}

		newCaptain, err := uc.userRepo.GetByID(ctx, newCaptainID)
		if err != nil {
			return fmt.Errorf("GetByID newCaptain: %w", err)
		}

		if newCaptain.TeamID == nil || *newCaptain.TeamID != team.ID {
			return entityError.ErrNewCaptainNotInTeam
		}

		if err := uc.txRepo.UpdateTeamCaptainTx(ctx, tx, team.ID, newCaptainID); err != nil {
			return fmt.Errorf("UpdateTeamCaptainTx: %w", err)
		}

		auditLog := &entity.TeamAuditLog{
			TeamID: team.ID,
			UserID: captainID,
			Action: entity.TeamActionCaptainTransfer,
			Details: map[string]any{
				"from": captainID.String(),
				"to":   newCaptainID.String(),
			},
		}
		if err := uc.txRepo.CreateTeamAuditLogTx(ctx, tx, auditLog); err != nil {
			return fmt.Errorf("CreateTeamAuditLogTx: %w", err)
		}

		return nil
	})
}

func (uc *TeamUseCase) GetByID(ctx context.Context, ID uuid.UUID) (*entity.Team, error) {
	team, err := uc.teamRepo.GetByID(ctx, ID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - GetByID: %w", err)
	}
	return team, nil
}

func (uc *TeamUseCase) GetMyTeam(ctx context.Context, userID uuid.UUID) (*entity.Team, []*entity.User, error) {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("TeamUseCase - GetMyTeam - GetByID: %w", err)
	}

	if user.TeamID == nil {
		return nil, nil, entityError.ErrTeamNotFound
	}

	team, err := uc.teamRepo.GetByID(ctx, *user.TeamID)
	if err != nil {
		return nil, nil, fmt.Errorf("TeamUseCase - GetMyTeam - GetByID: %w", err)
	}

	members, err := uc.userRepo.GetByTeamID(ctx, *user.TeamID)
	if err != nil {
		return nil, nil, fmt.Errorf("TeamUseCase - GetMyTeam - GetByTeamID: %w", err)
	}

	return team, members, nil
}

func (uc *TeamUseCase) GetTeamMembers(ctx context.Context, teamID uuid.UUID) ([]*entity.User, error) {
	users, err := uc.userRepo.GetByTeamID(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - GetTeamMembers: %w", err)
	}
	return users, nil
}

//nolint:gocognit,gocyclo
func (uc *TeamUseCase) CreateSoloTeam(ctx context.Context, userID uuid.UUID, confirmReset bool) (*entity.Team, error) {
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
		if err := uc.txRepo.LockUserTx(ctx, tx, userID); err != nil {
			return fmt.Errorf("LockUserTx: %w", err)
		}

		user, err := uc.userRepo.GetByID(ctx, userID)
		if err != nil {
			return fmt.Errorf("GetByID: %w", err)
		}

		if user.TeamID != nil {
			if err := uc.handleSoloTeamCleanup(ctx, tx, user, userID, confirmReset); err != nil {
				return err
			}
		}

		inviteToken := uuid.New()

		team = &entity.Team{
			Name:          user.Username,
			InviteToken:   inviteToken,
			CaptainID:     userID,
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

		if err := uc.txRepo.UpdateUserTeamIDTx(ctx, tx, userID, &team.ID); err != nil {
			return fmt.Errorf("UpdateUserTeamIDTx: %w", err)
		}

		auditLog := &entity.TeamAuditLog{
			TeamID: team.ID,
			UserID: userID,
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

func (uc *TeamUseCase) handleSoloTeamCleanup(ctx context.Context, tx pgx.Tx, user *entity.User, actorID uuid.UUID, confirmReset bool) error {
	oldTeam, err := uc.txRepo.GetTeamByIDTx(ctx, tx, *user.TeamID)
	if err != nil {
		return fmt.Errorf("GetTeamByIDTx: %w", err)
	}

	members, err := uc.txRepo.GetUsersByTeamIDTx(ctx, tx, *user.TeamID)
	if err != nil {
		return fmt.Errorf("GetUsersByTeamIDTx: %w", err)
	}

	if !uc.shouldCleanupSoloTeam(user, members, oldTeam) {
		return entityError.ErrUserAlreadyInTeam
	}

	if !confirmReset {
		return entityError.ErrConfirmationRequired
	}

	oldTeamID := *user.TeamID
	if err := uc.txRepo.SoftDeleteTeamTx(ctx, tx, oldTeamID); err != nil {
		return fmt.Errorf("SoftDeleteTeamTx: %w", err)
	}

	if err := uc.txRepo.DeleteSolvesByTeamIDTx(ctx, tx, oldTeamID); err != nil {
		return fmt.Errorf("DeleteSolvesByTeamIDTx: %w", err)
	}

	auditLog := &entity.TeamAuditLog{
		TeamID: oldTeamID,
		UserID: actorID,
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

func (uc *TeamUseCase) shouldCleanupSoloTeam(user *entity.User, members []*entity.User, oldTeam *entity.Team) bool {
	return len(members) == 1 && members[0].ID == user.ID && (oldTeam.IsSolo || oldTeam.IsAutoCreated)
}

//nolint:gocognit,gocyclo
func (uc *TeamUseCase) DisbandTeam(ctx context.Context, captainID uuid.UUID) error {
	comp, err := uc.compRepo.Get(ctx)
	if err != nil {
		return fmt.Errorf("GetCompetition: %w", err)
	}
	if !comp.AllowTeamSwitch {
		return entityError.ErrRosterFrozen
	}

	return uc.txRepo.RunTransaction(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if err := uc.txRepo.LockUserTx(ctx, tx, captainID); err != nil {
			return fmt.Errorf("LockUserTx: %w", err)
		}

		user, err := uc.userRepo.GetByID(ctx, captainID)
		if err != nil {
			return fmt.Errorf("GetByID: %w", err)
		}

		if user.TeamID == nil {
			return entityError.ErrTeamNotFound
		}

		if err := uc.txRepo.LockTeamTx(ctx, tx, *user.TeamID); err != nil {
			return fmt.Errorf("LockTeamTx: %w", err)
		}

		team, err := uc.teamRepo.GetByID(ctx, *user.TeamID)
		if err != nil {
			return fmt.Errorf("GetByID team: %w", err)
		}

		if team.CaptainID != captainID {
			return entityError.ErrNotCaptain
		}

		if err := uc.txRepo.SoftDeleteTeamTx(ctx, tx, team.ID); err != nil {
			return fmt.Errorf("SoftDeleteTeamTx: %w", err)
		}

		members, err := uc.txRepo.GetUsersByTeamIDTx(ctx, tx, team.ID)
		if err != nil {
			return fmt.Errorf("GetUsersByTeamIDTx: %w", err)
		}

		for _, member := range members {
			if err := uc.txRepo.UpdateUserTeamIDTx(ctx, tx, member.ID, nil); err != nil {
				return fmt.Errorf("UpdateUserTeamIDTx member %s: %w", member.ID, err)
			}
		}

		auditLog := &entity.TeamAuditLog{
			TeamID: team.ID,
			UserID: captainID,
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

//nolint:gocognit,gocyclo
func (uc *TeamUseCase) KickMember(ctx context.Context, captainID, targetUserID uuid.UUID) error {
	comp, err := uc.compRepo.Get(ctx)
	if err != nil {
		return fmt.Errorf("GetCompetition: %w", err)
	}
	if !comp.AllowTeamSwitch {
		return entityError.ErrRosterFrozen
	}

	if captainID == targetUserID {
		return entityError.ErrCannotLeaveAsOnlyMember
	}

	return uc.txRepo.RunTransaction(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if err := uc.txRepo.LockUserTx(ctx, tx, captainID); err != nil {
			return fmt.Errorf("LockUserTx captain: %w", err)
		}

		captain, err := uc.userRepo.GetByID(ctx, captainID)
		if err != nil {
			return fmt.Errorf("GetByID captain: %w", err)
		}

		if captain.TeamID == nil {
			return entityError.ErrTeamNotFound
		}

		if err := uc.txRepo.LockTeamTx(ctx, tx, *captain.TeamID); err != nil {
			return fmt.Errorf("LockTeamTx: %w", err)
		}

		team, err := uc.teamRepo.GetByID(ctx, *captain.TeamID)
		if err != nil {
			return fmt.Errorf("GetByID team: %w", err)
		}

		if team.CaptainID != captainID {
			return entityError.ErrNotCaptain
		}

		targetUser, err := uc.userRepo.GetByID(ctx, targetUserID)
		if err != nil {
			return fmt.Errorf("GetByID target: %w", err)
		}

		if targetUser.TeamID == nil || *targetUser.TeamID != team.ID {
			return entityError.ErrUserNotFound
		}

		if err := uc.txRepo.UpdateUserTeamIDTx(ctx, tx, targetUserID, nil); err != nil {
			return fmt.Errorf("UpdateUserTeamIDTx: %w", err)
		}

		auditLog := &entity.TeamAuditLog{
			TeamID: team.ID,
			UserID: captainID,
			Action: entity.TeamActionMemberKicked,
			Details: map[string]any{
				"target_user_id": targetUserID.String(),
			},
		}
		if err := uc.txRepo.CreateTeamAuditLogTx(ctx, tx, auditLog); err != nil {
			return fmt.Errorf("CreateTeamAuditLogTx: %w", err)
		}

		return nil
	})
}
