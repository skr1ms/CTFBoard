package team

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/repo"
	"github.com/skr1ms/CTFBoard/internal/usecase/competition"
	"github.com/skr1ms/CTFBoard/pkg/cache"
	"github.com/skr1ms/CTFBoard/pkg/usecaseutil"
)

const DefaultMaxTeamSize = 10

type TeamUseCase struct {
	teamRepo        repo.TeamRepository
	userRepo        repo.UserRepository
	compRepo        repo.CompetitionRepository
	txRepo          repo.TxRepository
	guard           *competition.Guard
	scoreboardCache cache.ScoreboardCacheInvalidator
	maxTeamSize     int
}

func NewTeamUseCase(
	teamRepo repo.TeamRepository,
	userRepo repo.UserRepository,
	compRepo repo.CompetitionRepository,
	txRepo repo.TxRepository,
	scoreboardCache cache.ScoreboardCacheInvalidator,
) *TeamUseCase {
	return &TeamUseCase{
		teamRepo:        teamRepo,
		userRepo:        userRepo,
		compRepo:        compRepo,
		txRepo:          txRepo,
		guard:           competition.NewGuard(compRepo),
		scoreboardCache: scoreboardCache,
		maxTeamSize:     DefaultMaxTeamSize,
	}
}

func NewTeamUseCaseWithSize(
	teamRepo repo.TeamRepository,
	userRepo repo.UserRepository,
	compRepo repo.CompetitionRepository,
	txRepo repo.TxRepository,
	scoreboardCache cache.ScoreboardCacheInvalidator,
	maxTeamSize int,
) *TeamUseCase {
	return &TeamUseCase{
		teamRepo:        teamRepo,
		userRepo:        userRepo,
		compRepo:        compRepo,
		txRepo:          txRepo,
		guard:           competition.NewGuard(compRepo),
		scoreboardCache: scoreboardCache,
		maxTeamSize:     maxTeamSize,
	}
}

func (uc *TeamUseCase) Create(ctx context.Context, name string, captainID uuid.UUID, isSolo, confirmReset bool) (*entity.Team, error) {
	_, err := uc.guard.RequireTeamSwitchAndTeamsMode(ctx)
	if err != nil {
		return nil, usecaseutil.Wrap(err, "TeamUseCase - Create - Guard")
	}
	var team *entity.Team
	err = uc.txRepo.RunTransaction(ctx, func(ctx context.Context, tx repo.Transaction) error {
		var err2 error
		team, err2 = uc.createTx(ctx, tx, name, captainID, isSolo, confirmReset)
		return err2
	})
	if err != nil {
		return nil, usecaseutil.Wrap(err, "TeamUseCase - Create - Transaction")
	}
	return team, nil
}

func (uc *TeamUseCase) createTx(ctx context.Context, tx repo.Transaction, name string, captainID uuid.UUID, isSolo, confirmReset bool) (*entity.Team, error) {
	if err := uc.txRepo.LockUserTx(ctx, tx, captainID); err != nil {
		return nil, usecaseutil.Wrap(err, "LockUserTx")
	}
	if err := uc.validateTeamNameAvailableTx(ctx, tx, name); err != nil {
		return nil, err
	}
	user, err := uc.userRepo.GetByID(ctx, captainID)
	if err != nil {
		return nil, usecaseutil.Wrap(err, "GetByID")
	}
	if user.TeamID != nil {
		if err := uc.handleSoloTeamCleanup(ctx, tx, user, captainID, confirmReset); err != nil {
			return nil, err
		}
	}
	team := &entity.Team{
		Name:        name,
		InviteToken: uuid.New(),
		CaptainID:   captainID,
		IsSolo:      isSolo,
	}
	if err := uc.txRepo.CreateTeamTx(ctx, tx, team); err != nil {
		return nil, usecaseutil.Wrap(err, "CreateTeamTx")
	}
	if err := uc.txRepo.UpdateUserTeamIDTx(ctx, tx, captainID, &team.ID); err != nil {
		return nil, usecaseutil.Wrap(err, "UpdateUserTeamIDTx")
	}
	auditLog := &entity.TeamAuditLog{TeamID: team.ID, UserID: captainID, Action: entity.TeamActionCreated}
	if err := uc.txRepo.CreateTeamAuditLogTx(ctx, tx, auditLog); err != nil {
		return nil, usecaseutil.Wrap(err, "CreateTeamAuditLogTx")
	}
	return team, nil
}

func (uc *TeamUseCase) validateTeamNameAvailableTx(ctx context.Context, tx repo.Transaction, name string) error {
	_, err := uc.txRepo.GetTeamByNameTx(ctx, tx, name)
	if err == nil {
		return fmt.Errorf("%w: name", entityError.ErrTeamAlreadyExists)
	}
	if !errors.Is(err, entityError.ErrTeamNotFound) {
		return usecaseutil.Wrap(err, "GetTeamByNameTx")
	}
	return nil
}

func (uc *TeamUseCase) TryCreate(ctx context.Context, name string, captainID uuid.UUID, isSolo bool) (*OperationResult, error) {
	_, err := uc.guard.RequireTeamSwitchAndTeamsMode(ctx)
	if err != nil {
		return nil, usecaseutil.Wrap(err, "TeamUseCase - TryCreate - Guard")
	}
	user, err := uc.userRepo.GetByID(ctx, captainID)
	if err != nil {
		return nil, usecaseutil.Wrap(err, "TeamUseCase - TryCreate - GetByID")
	}
	if user.TeamID == nil {
		team, err := uc.Create(ctx, name, captainID, isSolo, false)
		if err != nil {
			return nil, err
		}
		return &OperationResult{Team: team}, nil
	}
	return uc.tryCreateWhenInTeam(ctx, user, name, captainID, isSolo)
}

func (uc *TeamUseCase) tryCreateWhenInTeam(ctx context.Context, user *entity.User, _ string, captainID uuid.UUID, _ bool) (*OperationResult, error) {
	var result *OperationResult
	err := uc.txRepo.RunTransaction(ctx, func(ctx context.Context, tx repo.Transaction) error {
		if err := uc.txRepo.LockUserTx(ctx, tx, captainID); err != nil {
			return err
		}
		oldTeam, err := uc.txRepo.GetTeamByIDTx(ctx, tx, *user.TeamID)
		if err != nil {
			return usecaseutil.Wrap(err, "GetTeamByIDTx")
		}
		members, err := uc.txRepo.GetUsersByTeamIDTx(ctx, tx, *user.TeamID)
		if err != nil {
			return usecaseutil.Wrap(err, "GetUsersByTeamIDTx")
		}
		if !uc.shouldCleanupSoloTeam(user, members, oldTeam) {
			return entityError.ErrUserAlreadyInTeam
		}
		points, err := uc.txRepo.GetTeamScoreTx(ctx, tx, *user.TeamID)
		if err != nil {
			return usecaseutil.Wrap(err, "GetTeamScoreTx")
		}
		result = &OperationResult{
			RequiresConfirm:    true,
			ConfirmationReason: ConfirmReasonSoloTeamReset,
			AffectedData:       &AffectedData{Points: points, SolveCount: 0},
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (uc *TeamUseCase) ConfirmCreate(ctx context.Context, name string, captainID uuid.UUID, isSolo bool) (*entity.Team, error) {
	return uc.Create(ctx, name, captainID, isSolo, true)
}

func (uc *TeamUseCase) Join(ctx context.Context, inviteToken, userID uuid.UUID, confirmReset bool) (*entity.Team, error) {
	_, err := uc.guard.RequireTeamSwitch(ctx)
	if err != nil {
		return nil, usecaseutil.Wrap(err, "TeamUseCase - Join - Guard")
	}
	var team *entity.Team
	err = uc.txRepo.RunTransaction(ctx, func(ctx context.Context, tx repo.Transaction) error {
		var err2 error
		team, err2 = uc.joinTx(ctx, tx, inviteToken, userID, confirmReset)
		return err2
	})
	if err != nil {
		return nil, usecaseutil.Wrap(err, "TeamUseCase - Join - Transaction")
	}
	return team, nil
}

func (uc *TeamUseCase) joinTx(ctx context.Context, tx repo.Transaction, inviteToken, userID uuid.UUID, confirmReset bool) (*entity.Team, error) {
	if err := uc.txRepo.LockUserTx(ctx, tx, userID); err != nil {
		return nil, usecaseutil.Wrap(err, "LockUserTx")
	}
	team, err := uc.txRepo.GetTeamByInviteTokenTx(ctx, tx, inviteToken)
	if err != nil {
		return nil, usecaseutil.Wrap(err, "GetTeamByInviteTokenTx")
	}
	if err := uc.txRepo.LockTeamTx(ctx, tx, team.ID); err != nil {
		return nil, usecaseutil.Wrap(err, "LockTeamTx")
	}
	members, err := uc.txRepo.GetUsersByTeamIDTx(ctx, tx, team.ID)
	if err != nil {
		return nil, usecaseutil.Wrap(err, "GetUsersByTeamIDTx target")
	}
	if len(members) >= uc.maxTeamSize {
		return nil, entityError.ErrTeamFull
	}
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, usecaseutil.Wrap(err, "GetByID")
	}
	if user.TeamID != nil {
		if err := uc.handleSoloTeamCleanup(ctx, tx, user, userID, confirmReset); err != nil {
			return nil, err
		}
	}
	if err := uc.txRepo.UpdateUserTeamIDTx(ctx, tx, userID, &team.ID); err != nil {
		return nil, usecaseutil.Wrap(err, "UpdateUserTeamIDTx")
	}
	auditLog := &entity.TeamAuditLog{TeamID: team.ID, UserID: userID, Action: entity.TeamActionJoined}
	if err := uc.txRepo.CreateTeamAuditLogTx(ctx, tx, auditLog); err != nil {
		return nil, usecaseutil.Wrap(err, "CreateTeamAuditLogTx")
	}
	return team, nil
}

func (uc *TeamUseCase) Leave(ctx context.Context, userID uuid.UUID) error {
	_, err := uc.guard.RequireTeamSwitch(ctx)
	if err != nil {
		return usecaseutil.Wrap(err, "TeamUseCase - Leave - Guard")
	}
	return uc.txRepo.RunTransaction(ctx, uc.leaveTx(userID))
}

func (uc *TeamUseCase) leaveTx(userID uuid.UUID) func(ctx context.Context, tx repo.Transaction) error {
	return func(ctx context.Context, tx repo.Transaction) error {
		user, team, members, err := uc.leavePrepare(ctx, tx, userID)
		if err != nil {
			return err
		}
		if err := uc.leaveValidate(user, team, members); err != nil {
			return err
		}
		return uc.leaveExecute(ctx, tx, userID, team)
	}
}

func (uc *TeamUseCase) leavePrepare(ctx context.Context, tx repo.Transaction, userID uuid.UUID) (*entity.User, *entity.Team, []*entity.User, error) {
	if err := uc.txRepo.LockUserTx(ctx, tx, userID); err != nil {
		return nil, nil, nil, usecaseutil.Wrap(err, "LockUserTx")
	}
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, nil, nil, usecaseutil.Wrap(err, "GetByID")
	}
	if user.TeamID == nil {
		return nil, nil, nil, entityError.ErrTeamNotFound
	}
	if err := uc.txRepo.LockTeamTx(ctx, tx, *user.TeamID); err != nil {
		return nil, nil, nil, usecaseutil.Wrap(err, "LockTeamTx")
	}
	team, err := uc.teamRepo.GetByID(ctx, *user.TeamID)
	if err != nil {
		return nil, nil, nil, usecaseutil.Wrap(err, "GetByID team")
	}
	members, err := uc.txRepo.GetUsersByTeamIDTx(ctx, tx, *user.TeamID)
	if err != nil {
		return nil, nil, nil, usecaseutil.Wrap(err, "GetUsersByTeamIDTx")
	}
	return user, team, members, nil
}

func (uc *TeamUseCase) leaveValidate(user *entity.User, team *entity.Team, members []*entity.User) error {
	if len(members) == 1 {
		return entityError.ErrCannotLeaveAsOnlyMember
	}
	if team.CaptainID == user.ID {
		return entityError.ErrNotCaptain
	}
	return nil
}

func (uc *TeamUseCase) leaveExecute(ctx context.Context, tx repo.Transaction, userID uuid.UUID, team *entity.Team) error {
	if err := uc.txRepo.UpdateUserTeamIDTx(ctx, tx, userID, nil); err != nil {
		return usecaseutil.Wrap(err, "UpdateUserTeamIDTx")
	}
	auditLog := &entity.TeamAuditLog{TeamID: team.ID, UserID: userID, Action: entity.TeamActionLeft}
	if err := uc.txRepo.CreateTeamAuditLogTx(ctx, tx, auditLog); err != nil {
		return usecaseutil.Wrap(err, "CreateTeamAuditLogTx")
	}
	return nil
}

func (uc *TeamUseCase) TransferCaptain(ctx context.Context, captainID, newCaptainID uuid.UUID) error {
	_, err := uc.guard.RequireTeamSwitch(ctx)
	if err != nil {
		return usecaseutil.Wrap(err, "TeamUseCase - TransferCaptain - Guard")
	}
	if captainID == newCaptainID {
		return entityError.ErrCannotTransferToSelf
	}
	return uc.txRepo.RunTransaction(ctx, uc.transferCaptainTx(captainID, newCaptainID))
}

func (uc *TeamUseCase) transferCaptainTx(captainID, newCaptainID uuid.UUID) func(ctx context.Context, tx repo.Transaction) error {
	return func(ctx context.Context, tx repo.Transaction) error {
		captain, team, newCaptain, err := uc.transferCaptainPrepare(ctx, tx, captainID, newCaptainID)
		if err != nil {
			return err
		}
		if err := uc.transferCaptainValidate(captain, team, newCaptain, captainID); err != nil {
			return err
		}
		return uc.transferCaptainExecute(ctx, tx, captainID, newCaptainID, team)
	}
}

func (uc *TeamUseCase) transferCaptainPrepare(ctx context.Context, tx repo.Transaction, captainID, newCaptainID uuid.UUID) (*entity.User, *entity.Team, *entity.User, error) {
	if err := uc.txRepo.LockUserTx(ctx, tx, captainID); err != nil {
		return nil, nil, nil, usecaseutil.Wrap(err, "LockUserTx captain")
	}
	captain, err := uc.userRepo.GetByID(ctx, captainID)
	if err != nil {
		return nil, nil, nil, usecaseutil.Wrap(err, "GetByID captain")
	}
	if captain.TeamID == nil {
		return nil, nil, nil, entityError.ErrTeamNotFound
	}
	if err := uc.txRepo.LockTeamTx(ctx, tx, *captain.TeamID); err != nil {
		return nil, nil, nil, usecaseutil.Wrap(err, "LockTeamTx")
	}
	team, err := uc.teamRepo.GetByID(ctx, *captain.TeamID)
	if err != nil {
		return nil, nil, nil, usecaseutil.Wrap(err, "GetByID team")
	}
	newCaptain, err := uc.userRepo.GetByID(ctx, newCaptainID)
	if err != nil {
		return nil, nil, nil, usecaseutil.Wrap(err, "GetByID newCaptain")
	}
	return captain, team, newCaptain, nil
}

func (uc *TeamUseCase) transferCaptainValidate(_ *entity.User, team *entity.Team, newCaptain *entity.User, captainID uuid.UUID) error {
	if team.CaptainID != captainID {
		return entityError.ErrNotCaptain
	}
	if newCaptain.TeamID == nil || *newCaptain.TeamID != team.ID {
		return entityError.ErrNewCaptainNotInTeam
	}
	return nil
}

func (uc *TeamUseCase) transferCaptainExecute(ctx context.Context, tx repo.Transaction, captainID, newCaptainID uuid.UUID, team *entity.Team) error {
	if err := uc.txRepo.UpdateTeamCaptainTx(ctx, tx, team.ID, newCaptainID); err != nil {
		return usecaseutil.Wrap(err, "UpdateTeamCaptainTx")
	}
	auditLog := &entity.TeamAuditLog{
		TeamID: team.ID, UserID: captainID, Action: entity.TeamActionCaptainTransfer,
		Details: map[string]any{"from": captainID.String(), "to": newCaptainID.String()},
	}
	if err := uc.txRepo.CreateTeamAuditLogTx(ctx, tx, auditLog); err != nil {
		return usecaseutil.Wrap(err, "CreateTeamAuditLogTx")
	}
	return nil
}

func (uc *TeamUseCase) GetByID(ctx context.Context, ID uuid.UUID) (*entity.Team, error) {
	team, err := uc.teamRepo.GetByID(ctx, ID)
	if err != nil {
		return nil, usecaseutil.Wrap(err, "TeamUseCase - GetByID")
	}
	return team, nil
}

func (uc *TeamUseCase) GetMyTeam(ctx context.Context, userID uuid.UUID) (*entity.Team, []*entity.User, error) {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, nil, usecaseutil.Wrap(err, "TeamUseCase - GetMyTeam - GetByID")
	}

	if user.TeamID == nil {
		return nil, nil, entityError.ErrTeamNotFound
	}

	team, err := uc.teamRepo.GetByID(ctx, *user.TeamID)
	if err != nil {
		return nil, nil, usecaseutil.Wrap(err, "TeamUseCase - GetMyTeam - GetByID")
	}

	members, err := uc.userRepo.GetByTeamID(ctx, *user.TeamID)
	if err != nil {
		return nil, nil, usecaseutil.Wrap(err, "TeamUseCase - GetMyTeam - GetByTeamID")
	}

	return team, members, nil
}

func (uc *TeamUseCase) GetTeamMembers(ctx context.Context, teamID uuid.UUID) ([]*entity.User, error) {
	users, err := uc.userRepo.GetByTeamID(ctx, teamID)
	if err != nil {
		return nil, usecaseutil.Wrap(err, "TeamUseCase - GetTeamMembers")
	}
	return users, nil
}

func (uc *TeamUseCase) CreateSoloTeam(ctx context.Context, userID uuid.UUID, confirmReset bool) (*entity.Team, error) {
	_, err := uc.guard.RequireTeamSwitchAndSoloMode(ctx)
	if err != nil {
		return nil, usecaseutil.Wrap(err, "TeamUseCase - CreateSoloTeam - Guard")
	}
	var team *entity.Team
	err = uc.txRepo.RunTransaction(ctx, func(ctx context.Context, tx repo.Transaction) error {
		var err2 error
		team, err2 = uc.createSoloTeamTx(ctx, tx, userID, confirmReset)
		return err2
	})
	if err != nil {
		return nil, usecaseutil.Wrap(err, "TeamUseCase - CreateSoloTeam - Transaction")
	}
	return team, nil
}

func (uc *TeamUseCase) createSoloTeamTx(ctx context.Context, tx repo.Transaction, userID uuid.UUID, confirmReset bool) (*entity.Team, error) {
	if err := uc.txRepo.LockUserTx(ctx, tx, userID); err != nil {
		return nil, usecaseutil.Wrap(err, "LockUserTx")
	}
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, usecaseutil.Wrap(err, "GetByID")
	}
	if user.TeamID != nil {
		if err := uc.handleSoloTeamCleanup(ctx, tx, user, userID, confirmReset); err != nil {
			return nil, err
		}
	}
	team := &entity.Team{
		Name: user.Username, InviteToken: uuid.New(), CaptainID: userID,
		IsSolo: true, IsAutoCreated: false,
	}
	if _, err := uc.txRepo.GetTeamByNameTx(ctx, tx, team.Name); err == nil {
		team.Name = fmt.Sprintf("%s (Solo)", user.Username)
	}
	if err := uc.txRepo.CreateTeamTx(ctx, tx, team); err != nil {
		return nil, usecaseutil.Wrap(err, "CreateTeamTx")
	}
	if err := uc.txRepo.UpdateUserTeamIDTx(ctx, tx, userID, &team.ID); err != nil {
		return nil, usecaseutil.Wrap(err, "UpdateUserTeamIDTx")
	}
	auditLog := &entity.TeamAuditLog{
		TeamID: team.ID, UserID: userID, Action: entity.TeamActionCreated,
		Details: map[string]any{"mode": "solo"},
	}
	if err := uc.txRepo.CreateTeamAuditLogTx(ctx, tx, auditLog); err != nil {
		return nil, usecaseutil.Wrap(err, "CreateTeamAuditLogTx")
	}
	return team, nil
}

func (uc *TeamUseCase) handleSoloTeamCleanup(ctx context.Context, tx repo.Transaction, user *entity.User, actorID uuid.UUID, confirmReset bool) error {
	oldTeam, err := uc.txRepo.GetTeamByIDTx(ctx, tx, *user.TeamID)
	if err != nil {
		return usecaseutil.Wrap(err, "GetTeamByIDTx")
	}

	members, err := uc.txRepo.GetUsersByTeamIDTx(ctx, tx, *user.TeamID)
	if err != nil {
		return usecaseutil.Wrap(err, "GetUsersByTeamIDTx")
	}

	if !uc.shouldCleanupSoloTeam(user, members, oldTeam) {
		return entityError.ErrUserAlreadyInTeam
	}

	if !confirmReset {
		return entityError.ErrConfirmationRequired
	}

	oldTeamID := *user.TeamID
	if err := uc.txRepo.SoftDeleteTeamTx(ctx, tx, oldTeamID); err != nil {
		return usecaseutil.Wrap(err, "SoftDeleteTeamTx")
	}

	if err := uc.txRepo.DeleteSolvesByTeamIDTx(ctx, tx, oldTeamID); err != nil {
		return usecaseutil.Wrap(err, "DeleteSolvesByTeamIDTx")
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
		return usecaseutil.Wrap(err, "CreateTeamAuditLogTx")
	}
	return nil
}

func (uc *TeamUseCase) shouldCleanupSoloTeam(user *entity.User, members []*entity.User, oldTeam *entity.Team) bool {
	return len(members) == 1 && members[0].ID == user.ID && (oldTeam.IsSolo || oldTeam.IsAutoCreated)
}

func (uc *TeamUseCase) DisbandTeam(ctx context.Context, captainID uuid.UUID) error {
	_, err := uc.guard.RequireTeamSwitch(ctx)
	if err != nil {
		return usecaseutil.Wrap(err, "TeamUseCase - DisbandTeam - Guard")
	}
	return uc.txRepo.RunTransaction(ctx, uc.disbandTeamTx(captainID))
}

func (uc *TeamUseCase) disbandTeamTx(captainID uuid.UUID) func(ctx context.Context, tx repo.Transaction) error {
	return func(ctx context.Context, tx repo.Transaction) error {
		_, team, members, err := uc.disbandPrepare(ctx, tx, captainID)
		if err != nil {
			return err
		}
		if err := uc.disbandValidate(team, captainID); err != nil {
			return err
		}
		return uc.disbandExecute(ctx, tx, team, members, captainID)
	}
}

func (uc *TeamUseCase) disbandPrepare(ctx context.Context, tx repo.Transaction, captainID uuid.UUID) (*entity.User, *entity.Team, []*entity.User, error) {
	if err := uc.txRepo.LockUserTx(ctx, tx, captainID); err != nil {
		return nil, nil, nil, usecaseutil.Wrap(err, "LockUserTx")
	}
	user, err := uc.userRepo.GetByID(ctx, captainID)
	if err != nil {
		return nil, nil, nil, usecaseutil.Wrap(err, "GetByID")
	}
	if user.TeamID == nil {
		return nil, nil, nil, entityError.ErrTeamNotFound
	}
	if err := uc.txRepo.LockTeamTx(ctx, tx, *user.TeamID); err != nil {
		return nil, nil, nil, usecaseutil.Wrap(err, "LockTeamTx")
	}
	team, err := uc.teamRepo.GetByID(ctx, *user.TeamID)
	if err != nil {
		return nil, nil, nil, usecaseutil.Wrap(err, "GetByID team")
	}
	members, err := uc.txRepo.GetUsersByTeamIDTx(ctx, tx, team.ID)
	if err != nil {
		return nil, nil, nil, usecaseutil.Wrap(err, "GetUsersByTeamIDTx")
	}
	return user, team, members, nil
}

func (uc *TeamUseCase) disbandValidate(team *entity.Team, captainID uuid.UUID) error {
	if team.CaptainID != captainID {
		return entityError.ErrNotCaptain
	}
	return nil
}

func (uc *TeamUseCase) disbandExecute(ctx context.Context, tx repo.Transaction, team *entity.Team, members []*entity.User, captainID uuid.UUID) error {
	if err := uc.txRepo.SoftDeleteTeamTx(ctx, tx, team.ID); err != nil {
		return usecaseutil.Wrap(err, "SoftDeleteTeamTx")
	}
	for _, member := range members {
		if err := uc.txRepo.UpdateUserTeamIDTx(ctx, tx, member.ID, nil); err != nil {
			return fmt.Errorf("UpdateUserTeamIDTx member %s: %w", member.ID, err)
		}
	}
	auditLog := &entity.TeamAuditLog{
		TeamID: team.ID, UserID: captainID, Action: entity.TeamActionDeleted,
		Details: map[string]any{"reason": "disbanded_by_captain"},
	}
	return uc.txRepo.CreateTeamAuditLogTx(ctx, tx, auditLog)
}

func (uc *TeamUseCase) KickMember(ctx context.Context, captainID, targetUserID uuid.UUID) error {
	_, err := uc.guard.RequireTeamSwitch(ctx)
	if err != nil {
		return usecaseutil.Wrap(err, "TeamUseCase - KickMember - Guard")
	}
	if captainID == targetUserID {
		return entityError.ErrCannotLeaveAsOnlyMember
	}
	return uc.txRepo.RunTransaction(ctx, uc.kickMemberTx(captainID, targetUserID))
}

func (uc *TeamUseCase) kickMemberTx(captainID, targetUserID uuid.UUID) func(ctx context.Context, tx repo.Transaction) error {
	return func(ctx context.Context, tx repo.Transaction) error {
		captain, team, targetUser, err := uc.kickMemberPrepare(ctx, tx, captainID, targetUserID)
		if err != nil {
			return err
		}
		if err := uc.kickMemberValidate(captain, team, targetUser, captainID, targetUserID); err != nil {
			return err
		}
		return uc.kickMemberExecute(ctx, tx, team.ID, captainID, targetUserID)
	}
}

func (uc *TeamUseCase) kickMemberPrepare(ctx context.Context, tx repo.Transaction, captainID, targetUserID uuid.UUID) (*entity.User, *entity.Team, *entity.User, error) {
	if err := uc.txRepo.LockUserTx(ctx, tx, captainID); err != nil {
		return nil, nil, nil, usecaseutil.Wrap(err, "LockUserTx captain")
	}
	captain, err := uc.userRepo.GetByID(ctx, captainID)
	if err != nil {
		return nil, nil, nil, usecaseutil.Wrap(err, "GetByID captain")
	}
	if captain.TeamID == nil {
		return nil, nil, nil, entityError.ErrTeamNotFound
	}
	if err := uc.txRepo.LockTeamTx(ctx, tx, *captain.TeamID); err != nil {
		return nil, nil, nil, usecaseutil.Wrap(err, "LockTeamTx")
	}
	team, err := uc.teamRepo.GetByID(ctx, *captain.TeamID)
	if err != nil {
		return nil, nil, nil, usecaseutil.Wrap(err, "GetByID team")
	}
	targetUser, err := uc.userRepo.GetByID(ctx, targetUserID)
	if err != nil {
		return nil, nil, nil, usecaseutil.Wrap(err, "GetByID target")
	}
	return captain, team, targetUser, nil
}

func (uc *TeamUseCase) kickMemberValidate(_ *entity.User, team *entity.Team, targetUser *entity.User, captainID, _ uuid.UUID) error {
	if team.CaptainID != captainID {
		return entityError.ErrNotCaptain
	}
	if targetUser.TeamID == nil || *targetUser.TeamID != team.ID {
		return entityError.ErrUserNotFound
	}
	return nil
}

func (uc *TeamUseCase) kickMemberExecute(ctx context.Context, tx repo.Transaction, teamID, captainID, targetUserID uuid.UUID) error {
	if err := uc.txRepo.UpdateUserTeamIDTx(ctx, tx, targetUserID, nil); err != nil {
		return usecaseutil.Wrap(err, "UpdateUserTeamIDTx")
	}
	auditLog := &entity.TeamAuditLog{
		TeamID: teamID, UserID: captainID, Action: entity.TeamActionMemberKicked,
		Details: map[string]any{"target_user_id": targetUserID.String()},
	}
	return uc.txRepo.CreateTeamAuditLogTx(ctx, tx, auditLog)
}

func (uc *TeamUseCase) BanTeam(ctx context.Context, teamID uuid.UUID, reason string) error {
	_, err := uc.teamRepo.GetByID(ctx, teamID)
	if err != nil {
		return usecaseutil.Wrap(err, "TeamUseCase - BanTeam - GetByID")
	}

	if err := uc.teamRepo.Ban(ctx, teamID, reason); err != nil {
		return usecaseutil.Wrap(err, "TeamUseCase - BanTeam - Ban")
	}

	uc.invalidateScoreboardCache(ctx)
	return nil
}

func (uc *TeamUseCase) UnbanTeam(ctx context.Context, teamID uuid.UUID) error {
	_, err := uc.teamRepo.GetByID(ctx, teamID)
	if err != nil {
		return usecaseutil.Wrap(err, "TeamUseCase - UnbanTeam - GetByID")
	}

	if err := uc.teamRepo.Unban(ctx, teamID); err != nil {
		return usecaseutil.Wrap(err, "TeamUseCase - UnbanTeam - Unban")
	}

	uc.invalidateScoreboardCache(ctx)
	return nil
}

func (uc *TeamUseCase) SetHidden(ctx context.Context, teamID uuid.UUID, hidden bool) error {
	_, err := uc.teamRepo.GetByID(ctx, teamID)
	if err != nil {
		return usecaseutil.Wrap(err, "TeamUseCase - SetHidden - GetByID")
	}

	if err := uc.teamRepo.SetHidden(ctx, teamID, hidden); err != nil {
		return usecaseutil.Wrap(err, "TeamUseCase - SetHidden - SetHidden")
	}

	uc.invalidateScoreboardCache(ctx)
	return nil
}

func (uc *TeamUseCase) SetBracket(ctx context.Context, teamID uuid.UUID, bracketID *uuid.UUID) error {
	_, err := uc.teamRepo.GetByID(ctx, teamID)
	if err != nil {
		return usecaseutil.Wrap(err, "TeamUseCase - SetBracket - GetByID")
	}
	if err := uc.teamRepo.SetBracket(ctx, teamID, bracketID); err != nil {
		return usecaseutil.Wrap(err, "TeamUseCase - SetBracket")
	}
	uc.invalidateScoreboardCache(ctx)
	return nil
}

func (uc *TeamUseCase) invalidateScoreboardCache(ctx context.Context) {
	if uc.scoreboardCache != nil {
		uc.scoreboardCache.InvalidateAll(ctx)
	}
}
