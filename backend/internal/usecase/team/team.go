package team

import (
	"context"
	"errors"
	"fmt"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/cache"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/scoring"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

type TeamUseCase struct {
	deps TeamDeps
}

type TeamDeps struct {
	TeamRepo           repo.TeamRepository
	UserRepo           repo.UserRepository
	SolveRepo          repo.SolveRepository
	SubmissionRepo     repo.SubmissionRepository
	AwardRepo          repo.AwardRepository
	CompRepo           repo.CompetitionRepository
	SettingsGetter     usecase.SettingsGetter
	ChallengeRepo      repo.ChallengeRepository
	TM                 repo.TransactionManager
	Guard              usecase.CompetitionGuard
	ScoreboardCache    cache.ScoreboardCacheInvalidator
	UserCache          cache.UserCacheInvalidator
	TeamCache          *cache.Cache
	HintRepo           repo.HintRepository
	DefaultMaxTeamSize int
}

var _ usecase.TeamUseCase = (*TeamUseCase)(nil)

func NewTeamUseCase(deps TeamDeps) *TeamUseCase {
	if deps.DefaultMaxTeamSize <= 0 {
		deps.DefaultMaxTeamSize = 10
	}
	return &TeamUseCase{deps: deps}
}

func (uc *TeamUseCase) Create(ctx context.Context, name string, captainID uuid.UUID, isSolo, confirmReset bool) (*entity.Team, error) {
	var team *entity.Team
	err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		var err2 error
		team, err2 = uc.createTx(ctx, name, captainID, isSolo, confirmReset)
		if err2 != nil {
			return fmt.Errorf("TeamUseCase - Create - createTx: %w", err2)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - Create - TM.Run: %w", err)
	}
	uc.invalidateUserCache(ctx, captainID)
	uc.invalidateScoreboardCache(ctx)
	return team, nil
}

func (uc *TeamUseCase) checkMaxTeams(ctx context.Context) error {
	appSettings, err := uc.deps.SettingsGetter.Get(ctx)
	if err != nil {
		return fmt.Errorf("TeamUseCase - checkMaxTeams - SettingsGetter.Get: %w", err)
	}
	if appSettings.MaxTeams <= 0 {
		return nil
	}
	// Acquire an advisory lock for the duration of the transaction to serialize
	// concurrent team-creation requests and prevent exceeding MaxTeams via TOCTOU.
	const maxTeamsLockKey int64 = 0x4354467465616D73 // "CTFteams"
	if err := uc.deps.TeamRepo.AcquireAdvisoryLock(ctx, maxTeamsLockKey); err != nil {
		return fmt.Errorf("TeamUseCase - checkMaxTeams - AcquireAdvisoryLock: %w", err)
	}
	currentCount, err := uc.deps.TeamRepo.CountActiveTeams(ctx)
	if err != nil {
		return fmt.Errorf("TeamUseCase - checkMaxTeams - TeamRepo.CountActiveTeams: %w", err)
	}
	if currentCount >= appSettings.MaxTeams {
		return httperr.ErrMaxTeamsReached
	}
	return nil
}

func (uc *TeamUseCase) createTx(ctx context.Context, name string, captainID uuid.UUID, isSolo, confirmReset bool) (*entity.Team, error) {
	// Guard is checked inside the transaction so the competition state is
	// consistent with the rest of the write (prevents TOCTOU on AllowTeamSwitch/Mode).
	comp, err := uc.deps.Guard.RequireTeamSwitchAndTeamsMode(ctx)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - createTx - Guard: %w", err)
	}
	if isSolo && !comp.Mode.AllowsSolo() {
		return nil, httperr.ErrSoloModeNotAllowed
	}
	if err := uc.checkMaxTeams(ctx); err != nil {
		return nil, fmt.Errorf("TeamUseCase - createTx - checkMaxTeams: %w", err)
	}
	if err := uc.deps.UserRepo.Lock(ctx, captainID); err != nil {
		return nil, fmt.Errorf("TeamUseCase - createTx - UserRepo.Lock: %w", err)
	}
	if err := uc.validateTeamNameAvailable(ctx, name); err != nil {
		return nil, fmt.Errorf("TeamUseCase - createTx - validateTeamNameAvailable: %w", err)
	}
	user, err := uc.deps.UserRepo.GetByID(ctx, captainID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - createTx - UserRepo.GetByID: %w", err)
	}
	if user.IsBanned {
		return nil, httperr.ErrUserBanned
	}
	if user.TeamID != nil {
		if err := uc.handleSoloTeamCleanup(ctx, user, captainID, confirmReset); err != nil {
			return nil, fmt.Errorf("TeamUseCase - createTx - handleSoloTeamCleanup: %w", err)
		}
	}
	team := &entity.Team{
		Name:        name,
		InviteToken: uuid.New(),
		CaptainID:   captainID,
		IsSolo:      isSolo,
	}
	if err := uc.deps.TeamRepo.Create(ctx, team); err != nil {
		return nil, fmt.Errorf("TeamUseCase - createTx - TeamRepo.Create: %w", err)
	}
	if err := uc.deps.UserRepo.UpdateTeamID(ctx, captainID, &team.ID); err != nil {
		return nil, fmt.Errorf("TeamUseCase - createTx - UserRepo.UpdateTeamID: %w", err)
	}
	auditLog := &entity.TeamAuditLog{TeamID: team.ID, UserID: captainID, Action: entity.TeamActionCreated}
	if err := uc.deps.TeamRepo.CreateAuditLog(ctx, auditLog); err != nil {
		return nil, fmt.Errorf("TeamUseCase - createTx - TeamRepo.CreateAuditLog: %w", err)
	}
	return team, nil
}

func (uc *TeamUseCase) validateTeamNameAvailable(ctx context.Context, name string) error {
	_, err := uc.deps.TeamRepo.GetByName(ctx, name)
	if err == nil {
		return httperr.ErrTeamAlreadyExists
	}
	if !errors.Is(err, httperr.ErrTeamNotFound) {
		return fmt.Errorf("TeamUseCase - validateTeamNameAvailable - TeamRepo.GetByName: %w", err)
	}
	return nil
}

func (uc *TeamUseCase) TryCreate(ctx context.Context, name string, captainID uuid.UUID, isSolo bool) (*usecase.TeamCreateResult, error) {
	comp, err := uc.deps.Guard.RequireTeamSwitchAndTeamsMode(ctx)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - TryCreate - Guard: %w", err)
	}
	if isSolo && !comp.Mode.AllowsSolo() {
		return nil, httperr.ErrSoloModeNotAllowed
	}
	user, err := uc.deps.UserRepo.GetByID(ctx, captainID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - TryCreate - UserRepo.GetByID: %w", err)
	}
	if user.TeamID == nil {
		team, err := uc.Create(ctx, name, captainID, isSolo, false)
		if err != nil {
			return nil, fmt.Errorf("TeamUseCase - TryCreate - createTx: %w", err)
		}
		return &usecase.TeamCreateResult{Team: team}, nil
	}
	opResult, err := uc.tryCreateWhenInTeam(ctx, captainID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - TryCreate - tryCreateWhenInTeam: %w", err)
	}
	return opResult, nil
}

//nolint:gocognit,gocyclo
func (uc *TeamUseCase) tryCreateWhenInTeam(ctx context.Context, captainID uuid.UUID) (*usecase.TeamCreateResult, error) {
	var result *usecase.TeamCreateResult
	var oldTeamID *uuid.UUID
	err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		if _, err := uc.deps.Guard.RequireTeamSwitchAndTeamsMode(ctx); err != nil {
			return fmt.Errorf("TeamUseCase - tryCreateWhenInTeam - Guard.RequireTeamSwitchAndTeamsMode: %w", err)
		}
		if err := uc.deps.UserRepo.Lock(ctx, captainID); err != nil {
			return fmt.Errorf("TeamUseCase - tryCreateWhenInTeam - UserRepo.Lock: %w", err)
		}
		freshUser, err := uc.deps.UserRepo.GetByID(ctx, captainID)
		if err != nil {
			return fmt.Errorf("TeamUseCase - tryCreateWhenInTeam - UserRepo.GetByID: %w", err)
		}
		if freshUser.IsBanned {
			return httperr.ErrUserBanned
		}
		if freshUser.TeamID == nil {
			return httperr.ErrTeamNotFound
		}
		oldTeam, err := uc.deps.TeamRepo.GetByID(ctx, *freshUser.TeamID)
		if err != nil {
			return fmt.Errorf("TeamUseCase - tryCreateWhenInTeam - TeamRepo.GetByID: %w", err)
		}
		members, err := uc.deps.UserRepo.GetByTeamID(ctx, *freshUser.TeamID)
		if err != nil {
			return fmt.Errorf("TeamUseCase - tryCreateWhenInTeam - UserRepo.GetByTeamID: %w", err)
		}
		if !uc.shouldCleanupSoloTeam(freshUser, members, oldTeam) {
			return httperr.ErrUserAlreadyInTeam
		}
		points, err := uc.deps.SolveRepo.GetTeamScore(ctx, *freshUser.TeamID)
		if err != nil {
			return fmt.Errorf("TeamUseCase - tryCreateWhenInTeam - SolveRepo.GetTeamScore: %w", err)
		}
		oldTeamID = freshUser.TeamID
		result = &usecase.TeamCreateResult{
			RequiresConfirm:    true,
			ConfirmationReason: usecase.ConfirmReasonSoloTeamReset,
			AffectedData:       &usecase.TeamCreateAffectedData{Points: points, SolveCount: 0},
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - TryCreate - TM.Run: %w", err)
	}
	if result != nil && result.AffectedData != nil && oldTeamID != nil && uc.deps.SolveRepo != nil {
		if solves, err := uc.deps.SolveRepo.GetByTeamIDWithDetails(ctx, *oldTeamID); err == nil {
			result.AffectedData.SolveCount = len(solves)
		}
	}
	if result != nil && result.AffectedData != nil && oldTeamID != nil {
		if uc.deps.AwardRepo != nil {
			if total, err := uc.deps.AwardRepo.GetTeamTotalAwards(ctx, *oldTeamID); err == nil {
				result.AffectedData.AwardsTotal = total
			}
		}
		if uc.deps.HintRepo != nil {
			if n, err := uc.deps.HintRepo.CountByTeamID(ctx, *oldTeamID); err == nil {
				result.AffectedData.HintUnlockCount = n
			}
		}
	}
	return result, nil
}

func (uc *TeamUseCase) ConfirmCreate(ctx context.Context, name string, captainID uuid.UUID, isSolo bool) (*entity.Team, error) {
	return uc.Create(ctx, name, captainID, isSolo, true)
}

func (uc *TeamUseCase) Join(ctx context.Context, inviteToken, userID uuid.UUID, confirmReset bool) (*entity.Team, error) {
	_, err := uc.deps.Guard.RequireTeamSwitch(ctx)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - Join - Guard.RequireTeamSwitch: %w", err)
	}
	var team *entity.Team
	err = uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		var err2 error
		team, err2 = uc.joinTx(ctx, inviteToken, userID, confirmReset)
		if err2 != nil {
			return fmt.Errorf("TeamUseCase - Join - joinTx: %w", err2)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - Join - TM.Run: %w", err)
	}
	uc.invalidateUserCache(ctx, userID)
	uc.invalidateScoreboardCache(ctx)
	return team, nil
}

func (uc *TeamUseCase) joinTx(ctx context.Context, inviteToken, userID uuid.UUID, confirmReset bool) (*entity.Team, error) {
	team, user, err := uc.joinTxPrepare(ctx, inviteToken, userID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - joinTx - joinTxPrepare: %w", err)
	}
	if user.TeamID != nil {
		if err := uc.handleSoloTeamCleanup(ctx, user, userID, confirmReset); err != nil {
			return nil, fmt.Errorf("TeamUseCase - joinTx - handleSoloTeamCleanup: %w", err)
		}
	}
	if err := uc.deps.UserRepo.UpdateTeamID(ctx, userID, &team.ID); err != nil {
		return nil, fmt.Errorf("TeamUseCase - joinTx - UserRepo.UpdateTeamID: %w", err)
	}
	auditLog := &entity.TeamAuditLog{TeamID: team.ID, UserID: userID, Action: entity.TeamActionJoined}
	if err := uc.deps.TeamRepo.CreateAuditLog(ctx, auditLog); err != nil {
		return nil, fmt.Errorf("TeamUseCase - joinTx - TeamRepo.CreateAuditLog: %w", err)
	}
	return team, nil
}

func (uc *TeamUseCase) joinTxPrepare(ctx context.Context, inviteToken, userID uuid.UUID) (*entity.Team, *entity.User, error) {
	comp, err := uc.deps.Guard.RequireTeamSwitch(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("TeamUseCase - joinTx - Guard.RequireTeamSwitch: %w", err)
	}
	if !comp.Mode.AllowsTeams() {
		return nil, nil, httperr.ErrTeamsNotAllowed
	}
	if err := uc.deps.UserRepo.Lock(ctx, userID); err != nil {
		return nil, nil, fmt.Errorf("TeamUseCase - joinTx - UserRepo.Lock: %w", err)
	}
	team, err := uc.deps.TeamRepo.GetByInviteToken(ctx, inviteToken)
	if err != nil {
		return nil, nil, fmt.Errorf("TeamUseCase - joinTx - TeamRepo.GetByInviteToken: %w", err)
	}
	if team.IsBanned {
		return nil, nil, httperr.ErrTeamBanned
	}
	if team.IsSolo {
		return nil, nil, httperr.ErrTeamNotFound
	}
	if err := uc.deps.TeamRepo.Lock(ctx, team.ID); err != nil {
		return nil, nil, fmt.Errorf("TeamUseCase - joinTx - TeamRepo.Lock: %w", err)
	}
	team, err = uc.deps.TeamRepo.GetByID(ctx, team.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("TeamUseCase - joinTx - TeamRepo.GetByID: %w", err)
	}
	if team.IsBanned {
		return nil, nil, httperr.ErrTeamBanned
	}
	if team.IsSolo {
		return nil, nil, httperr.ErrTeamNotFound
	}
	members, err := uc.deps.UserRepo.GetByTeamID(ctx, team.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("TeamUseCase - joinTx - UserRepo.GetByTeamID: %w", err)
	}
	maxSize := comp.MaxTeamSize
	if maxSize <= 0 {
		maxSize = uc.deps.DefaultMaxTeamSize
	}
	if len(members) >= maxSize {
		return nil, nil, httperr.ErrTeamFull
	}
	user, err := uc.deps.UserRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("TeamUseCase - joinTx - UserRepo.GetByID: %w", err)
	}
	if user.IsBanned {
		return nil, nil, httperr.ErrUserBanned
	}
	return team, user, nil
}

func (uc *TeamUseCase) Leave(ctx context.Context, userID uuid.UUID) error {
	_, err := uc.deps.Guard.RequireTeamSwitch(ctx)
	if err != nil {
		return fmt.Errorf("TeamUseCase - Leave - Guard: %w", err)
	}
	if err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		return uc.leaveTx(ctx, userID)
	}); err != nil {
		return fmt.Errorf("TeamUseCase - Leave - TM.Run: %w", err)
	}
	uc.invalidateUserCache(ctx, userID)
	uc.invalidateScoreboardCache(ctx)
	return nil
}

func (uc *TeamUseCase) leaveTx(ctx context.Context, userID uuid.UUID) error {
	if _, err := uc.deps.Guard.RequireTeamSwitch(ctx); err != nil {
		return fmt.Errorf("TeamUseCase - leaveTx - Guard.RequireTeamSwitch: %w", err)
	}
	comp, err := uc.deps.CompRepo.Get(ctx)
	if err != nil {
		return fmt.Errorf("TeamUseCase - leaveTx - CompetitionRepo.Get: %w", err)
	}
	user, team, members, err := uc.leavePrepare(ctx, userID)
	if err != nil {
		return fmt.Errorf("TeamUseCase - leaveTx - leavePrepare: %w", err)
	}
	if err := uc.leaveValidate(user, team, members, comp); err != nil {
		return fmt.Errorf("TeamUseCase - leaveTx - leaveValidate: %w", err)
	}
	return uc.leaveExecute(ctx, userID, team)
}

func (uc *TeamUseCase) leavePrepare(ctx context.Context, userID uuid.UUID) (*entity.User, *entity.Team, []*entity.User, error) {
	if err := uc.deps.UserRepo.Lock(ctx, userID); err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - leavePrepare - UserRepo.Lock: %w", err)
	}
	user, err := uc.deps.UserRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - leavePrepare - UserRepo.GetByID: %w", err)
	}
	if user.TeamID == nil {
		return nil, nil, nil, httperr.ErrTeamNotFound
	}
	if err := uc.deps.TeamRepo.Lock(ctx, *user.TeamID); err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - leavePrepare - TeamRepo.Lock: %w", err)
	}
	team, err := uc.deps.TeamRepo.GetByID(ctx, *user.TeamID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - leavePrepare - TeamRepo.GetByID: %w", err)
	}
	members, err := uc.deps.UserRepo.GetByTeamID(ctx, *user.TeamID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - leavePrepare - UserRepo.GetByTeamID: %w", err)
	}
	return user, team, members, nil
}

func (uc *TeamUseCase) leaveValidate(user *entity.User, team *entity.Team, members []*entity.User, comp *entity.Competition) error {
	if len(members) == 1 {
		return httperr.ErrCannotLeaveAsOnlyMember
	}
	if team.CaptainID == user.ID {
		return httperr.ErrCaptainCannotLeave
	}
	if team.IsBanned {
		return httperr.ErrTeamBanned
	}
	minSize := comp.MinTeamSize
	if minSize > 0 && len(members)-1 < minSize {
		return httperr.ErrTeamBelowMinSize
	}
	return nil
}

func (uc *TeamUseCase) leaveExecute(ctx context.Context, userID uuid.UUID, team *entity.Team) error {
	if err := uc.deps.UserRepo.UpdateTeamID(ctx, userID, nil); err != nil {
		return fmt.Errorf("TeamUseCase - leaveExecute - UserRepo.UpdateTeamID: %w", err)
	}
	auditLog := &entity.TeamAuditLog{TeamID: team.ID, UserID: userID, Action: entity.TeamActionLeft}
	if err := uc.deps.TeamRepo.CreateAuditLog(ctx, auditLog); err != nil {
		return fmt.Errorf("TeamUseCase - leaveExecute - TeamRepo.CreateAuditLog: %w", err)
	}
	return nil
}

func (uc *TeamUseCase) TransferCaptain(ctx context.Context, captainID, newCaptainID uuid.UUID) error {
	_, err := uc.deps.Guard.RequireTeamSwitch(ctx)
	if err != nil {
		return fmt.Errorf("TeamUseCase - TransferCaptain - Guard: %w", err)
	}
	if captainID == newCaptainID {
		return httperr.ErrCannotTransferToSelf
	}
	var teamID uuid.UUID
	if err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		var txErr error
		teamID, txErr = uc.transferCaptainTx(ctx, captainID, newCaptainID)
		return txErr
	}); err != nil {
		return err
	}
	uc.invalidateUserCache(ctx, newCaptainID)
	uc.invalidateTeamCache(ctx, teamID)
	uc.invalidateScoreboardCache(ctx)
	return nil
}

func (uc *TeamUseCase) transferCaptainTx(ctx context.Context, captainID, newCaptainID uuid.UUID) (uuid.UUID, error) {
	if _, err := uc.deps.Guard.RequireTeamSwitch(ctx); err != nil {
		return uuid.Nil, fmt.Errorf("TeamUseCase - transferCaptainTx - Guard.RequireTeamSwitch: %w", err)
	}
	captain, team, newCaptain, err := uc.transferCaptainPrepare(ctx, captainID, newCaptainID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("TeamUseCase - transferCaptainTx - transferCaptainPrepare: %w", err)
	}
	if err := uc.transferCaptainValidate(captain, team, newCaptain, captainID); err != nil {
		return uuid.Nil, fmt.Errorf("TeamUseCase - transferCaptainTx - transferCaptainValidate: %w", err)
	}
	return team.ID, uc.transferCaptainExecute(ctx, captainID, newCaptainID, team)
}

func (uc *TeamUseCase) transferCaptainPrepare(ctx context.Context, captainID, newCaptainID uuid.UUID) (*entity.User, *entity.Team, *entity.User, error) {
	firstID, secondID := captainID, newCaptainID
	if captainID.String() > newCaptainID.String() {
		firstID, secondID = newCaptainID, captainID
	}
	if err := uc.deps.UserRepo.Lock(ctx, firstID); err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - transferCaptainPrepare - UserRepo.Lock(first): %w", err)
	}
	if err := uc.deps.UserRepo.Lock(ctx, secondID); err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - transferCaptainPrepare - UserRepo.Lock(second): %w", err)
	}
	captain, err := uc.deps.UserRepo.GetByID(ctx, captainID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - transferCaptainPrepare - UserRepo.GetByID: %w", err)
	}
	if captain.TeamID == nil {
		return nil, nil, nil, httperr.ErrTeamNotFound
	}
	if err := uc.deps.TeamRepo.Lock(ctx, *captain.TeamID); err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - transferCaptainPrepare - TeamRepo.Lock: %w", err)
	}
	team, err := uc.deps.TeamRepo.GetByID(ctx, *captain.TeamID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - transferCaptainPrepare - TeamRepo.GetByID: %w", err)
	}
	newCaptain, err := uc.deps.UserRepo.GetByID(ctx, newCaptainID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - transferCaptainPrepare - UserRepo.GetByID(newCaptain): %w", err)
	}
	return captain, team, newCaptain, nil
}

func (uc *TeamUseCase) transferCaptainValidate(captain *entity.User, team *entity.Team, newCaptain *entity.User, captainID uuid.UUID) error {
	if team.CaptainID != captainID {
		return httperr.ErrNotCaptain
	}
	if captain.IsBanned {
		return httperr.ErrUserBanned
	}
	if team.IsBanned {
		return httperr.ErrTeamBanned
	}
	if newCaptain.TeamID == nil || *newCaptain.TeamID != team.ID {
		return httperr.ErrNewCaptainNotInTeam
	}
	if newCaptain.IsBanned {
		return httperr.ErrUserBanned
	}
	return nil
}

func (uc *TeamUseCase) transferCaptainExecute(ctx context.Context, captainID, newCaptainID uuid.UUID, team *entity.Team) error {
	if err := uc.deps.TeamRepo.UpdateCaptain(ctx, team.ID, newCaptainID); err != nil {
		return fmt.Errorf("TeamUseCase - transferCaptainExecute - TeamRepo.UpdateCaptain: %w", err)
	}
	auditLog := &entity.TeamAuditLog{
		TeamID: team.ID, UserID: captainID, Action: entity.TeamActionCaptainTransfer,
		Details: map[string]any{"from": captainID.String(), "to": newCaptainID.String()},
	}
	if err := uc.deps.TeamRepo.CreateAuditLog(ctx, auditLog); err != nil {
		return fmt.Errorf("TeamUseCase - transferCaptainExecute - TeamRepo.CreateAuditLog: %w", err)
	}
	return nil
}

// GetByID returns a team by ID without filtering banned or hidden status.
// Admin-facing callers (e.g. admin panel) need unrestricted access; public-facing
// handlers (e.g. GET /teams/{ID}) should apply their own visibility checks if needed.
func (uc *TeamUseCase) GetByID(ctx context.Context, ID uuid.UUID) (*entity.Team, error) {
	team, err := uc.deps.TeamRepo.GetByID(ctx, ID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - GetByID - TeamRepo.GetByID: %w", err)
	}
	return team, nil
}

func (uc *TeamUseCase) GetMyTeam(ctx context.Context, userID uuid.UUID) (*entity.Team, []*entity.User, int, bool, error) {
	user, err := uc.deps.UserRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, nil, 0, false, fmt.Errorf("TeamUseCase - GetMyTeam - UserRepo.GetByID: %w", err)
	}
	if user.TeamID == nil {
		return nil, nil, 0, false, httperr.ErrUserNotInTeam
	}
	teamID := *user.TeamID

	var team *entity.Team
	var members []*entity.User
	g, gCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		var err2 error
		team, err2 = uc.deps.TeamRepo.GetByID(gCtx, teamID)
		if err2 != nil {
			return fmt.Errorf("TeamUseCase - GetMyTeam - TeamRepo.GetByID: %w", err2)
		}
		return nil
	})
	g.Go(func() error {
		var err2 error
		members, err2 = uc.deps.UserRepo.GetByTeamID(gCtx, teamID)
		if err2 != nil {
			return fmt.Errorf("TeamUseCase - GetMyTeam - UserRepo.GetByTeamID: %w", err2)
		}
		return nil
	})
	if err := g.Wait(); err != nil {
		return nil, nil, 0, false, fmt.Errorf("TeamUseCase - GetMyTeam - errgroup.Wait: %w", err)
	}
	minTeamSize := 0
	meetsMinSize := true
	if uc.deps.CompRepo != nil {
		comp, err := uc.deps.CompRepo.Get(ctx)
		if err == nil && comp != nil && comp.MinTeamSize > 0 {
			minTeamSize = comp.MinTeamSize
			meetsMinSize = len(members) >= comp.MinTeamSize
		}
	}
	return team, members, minTeamSize, meetsMinSize, nil
}

func (uc *TeamUseCase) GetTeamMembers(ctx context.Context, teamID uuid.UUID) ([]*entity.User, error) {
	users, err := uc.deps.UserRepo.GetByTeamID(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - GetTeamMembers - UserRepo.GetByTeamID: %w", err)
	}
	return users, nil
}

func (uc *TeamUseCase) CreateSoloTeam(ctx context.Context, userID uuid.UUID, confirmReset bool) (*entity.Team, error) {
	_, err := uc.deps.Guard.RequireTeamSwitchAndSoloMode(ctx)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - CreateSoloTeam - Guard: %w", err)
	}
	var team *entity.Team
	err = uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		var err2 error
		team, err2 = uc.createSoloTeamTx(ctx, userID, confirmReset, false, false)
		if err2 != nil {
			return fmt.Errorf("TeamUseCase - CreateSoloTeam - createSoloTeamTx: %w", err2)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - CreateSoloTeam - TM.Run: %w", err)
	}
	uc.invalidateUserCache(ctx, userID)
	return team, nil
}

// requireSoloModeOnly checks that the competition mode allows solo teams, without checking AllowTeamSwitch.
// Used for auto-creating solo teams on registration, where the roster-lock should not block new users.
func (uc *TeamUseCase) requireSoloModeOnly(ctx context.Context) error {
	comp, err := uc.deps.CompRepo.Get(ctx)
	if err != nil {
		return fmt.Errorf("TeamUseCase - requireSoloModeOnly - CompetitionRepo.Get: %w", err)
	}
	if !comp.Mode.AllowsSolo() {
		return httperr.ErrSoloModeNotAllowed
	}
	return nil
}

// CreateSoloTeamForNewUser creates a solo team with IsAutoCreated true (e.g. on registration when mode is solo_only).
// AllowTeamSwitch is intentionally bypassed here: new users must always be able to get their auto-team on registration.
func (uc *TeamUseCase) CreateSoloTeamForNewUser(ctx context.Context, userID uuid.UUID) (*entity.Team, error) {
	if err := uc.requireSoloModeOnly(ctx); err != nil {
		return nil, fmt.Errorf("TeamUseCase - CreateSoloTeamForNewUser - requireSoloModeOnly: %w", err)
	}
	var team *entity.Team
	err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		var err2 error
		team, err2 = uc.createSoloTeamTx(ctx, userID, false, true, true)
		if err2 != nil {
			return fmt.Errorf("TeamUseCase - CreateSoloTeamForNewUser - createSoloTeamTx: %w", err2)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - CreateSoloTeamForNewUser - TM.Run: %w", err)
	}
	uc.invalidateUserCache(ctx, userID)
	return team, nil
}

//nolint:gocognit,gocyclo // solo team creation flow
func (uc *TeamUseCase) createSoloTeamTx(ctx context.Context, userID uuid.UUID, confirmReset, isAutoCreated, skipTeamSwitchCheck bool) (*entity.Team, error) {
	if skipTeamSwitchCheck {
		if err := uc.requireSoloModeOnly(ctx); err != nil {
			return nil, fmt.Errorf("TeamUseCase - createSoloTeamTx - requireSoloModeOnly: %w", err)
		}
	} else {
		if _, err := uc.deps.Guard.RequireTeamSwitchAndSoloMode(ctx); err != nil {
			return nil, fmt.Errorf("TeamUseCase - createSoloTeamTx - Guard.RequireTeamSwitchAndSoloMode: %w", err)
		}
	}
	// In flexible mode, enforce MaxTeams so solo teams count toward the cap. In solo_only mode skip so every participant can have a team.
	comp, err := uc.deps.CompRepo.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - createSoloTeamTx - CompetitionRepo.Get: %w", err)
	}
	if comp.Mode == entity.ModeFlexible {
		if err := uc.checkMaxTeams(ctx); err != nil {
			return nil, fmt.Errorf("TeamUseCase - createSoloTeamTx - checkMaxTeams: %w", err)
		}
	}
	if err := uc.deps.UserRepo.Lock(ctx, userID); err != nil {
		return nil, fmt.Errorf("TeamUseCase - createSoloTeamTx - UserRepo.Lock: %w", err)
	}
	user, err := uc.deps.UserRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - createSoloTeamTx - UserRepo.GetByID: %w", err)
	}
	if user.IsBanned {
		return nil, httperr.ErrUserBanned
	}
	if user.TeamID != nil {
		if err := uc.handleSoloTeamCleanup(ctx, user, userID, confirmReset); err != nil {
			return nil, fmt.Errorf("TeamUseCase - createSoloTeamTx - handleSoloTeamCleanup: %w", err)
		}
	}
	team := &entity.Team{
		Name: user.Username, InviteToken: uuid.New(), CaptainID: userID,
		IsSolo: true, IsAutoCreated: isAutoCreated,
	}
	if _, err := uc.deps.TeamRepo.GetByName(ctx, team.Name); err == nil {
		fallback := fmt.Sprintf("%s (Solo)", user.Username)
		if _, err := uc.deps.TeamRepo.GetByName(ctx, fallback); err == nil {
			fallback = fmt.Sprintf("%s-%s", user.Username, team.InviteToken.String()[:8])
			if _, err := uc.deps.TeamRepo.GetByName(ctx, fallback); err == nil {
				fallback = fmt.Sprintf("%s-%s", user.Username, team.InviteToken.String())
			}
		}
		team.Name = fallback
	}
	if err := uc.deps.TeamRepo.Create(ctx, team); err != nil {
		return nil, fmt.Errorf("TeamUseCase - createSoloTeamTx - TeamRepo.Create: %w", err)
	}
	if err := uc.deps.UserRepo.UpdateTeamID(ctx, userID, &team.ID); err != nil {
		return nil, fmt.Errorf("TeamUseCase - createSoloTeamTx - UserRepo.UpdateTeamID: %w", err)
	}
	auditLog := &entity.TeamAuditLog{
		TeamID: team.ID, UserID: userID, Action: entity.TeamActionCreated,
		Details: map[string]any{"mode": "solo"},
	}
	if err := uc.deps.TeamRepo.CreateAuditLog(ctx, auditLog); err != nil {
		return nil, fmt.Errorf("TeamUseCase - createSoloTeamTx - TeamRepo.CreateAuditLog: %w", err)
	}
	return team, nil
}

type solveCountUpdater func(ctx context.Context, ID uuid.UUID) (int, error)

// adjustSolveCountsForTeam adjusts challenge solve_count for every challenge solved by
// the team and recalculates dynamic scoring. Pass challengeRepo.DecrementSolveCount to
// subtract (ban / delete) or challengeRepo.IncrementSolveCount to restore (unban).
//
//nolint:gocognit,gocyclo // iterates all team solves; dynamic scoring recalc requires multiple branches per challenge
func (uc *TeamUseCase) adjustSolveCountsForTeam(ctx context.Context, teamID uuid.UUID, updater solveCountUpdater) error {
	if uc.deps.ChallengeRepo == nil {
		return nil
	}
	solves, err := uc.deps.SolveRepo.GetByTeamIDWithDetails(ctx, teamID)
	if err != nil {
		return fmt.Errorf("TeamUseCase - adjustSolveCountsForTeam - SolveRepo.GetByTeamIDWithDetails: %w", err)
	}
	seen := make(map[uuid.UUID]struct{})
	var challengeIDs []uuid.UUID
	for _, s := range solves {
		if _, ok := seen[s.ChallengeID]; ok {
			continue
		}
		seen[s.ChallengeID] = struct{}{}
		challengeIDs = append(challengeIDs, s.ChallengeID)
	}
	if len(challengeIDs) == 0 {
		return nil
	}
	challengesMap, err := uc.deps.ChallengeRepo.GetByIDs(ctx, challengeIDs)
	if err != nil {
		return fmt.Errorf("TeamUseCase - adjustSolveCountsForTeam - ChallengeRepo.GetByIDs: %w", err)
	}
	for _, challengeID := range challengeIDs {
		challenge := challengesMap[challengeID]
		if challenge == nil {
			continue
		}
		solveCount, err := updater(ctx, challengeID)
		if err != nil {
			return fmt.Errorf("TeamUseCase - adjustSolveCountsForTeam - updater: %w", err)
		}
		if challenge.InitialValue > 0 && challenge.Decay > 0 {
			newPoints := scoring.CalculateDynamicScore(challenge.InitialValue, challenge.MinValue, challenge.Decay, solveCount)
			if err := uc.deps.ChallengeRepo.UpdatePoints(ctx, challengeID, newPoints); err != nil {
				return fmt.Errorf("TeamUseCase - adjustSolveCountsForTeam - ChallengeRepo.UpdatePoints: %w", err)
			}
		}
	}
	return nil
}

func (uc *TeamUseCase) handleSoloTeamCleanup(ctx context.Context, user *entity.User, actorID uuid.UUID, confirmReset bool) error {
	if user.TeamID == nil {
		return nil
	}
	if err := uc.deps.TeamRepo.Lock(ctx, *user.TeamID); err != nil {
		return fmt.Errorf("TeamUseCase - handleSoloTeamCleanup - TeamRepo.Lock: %w", err)
	}
	oldTeam, err := uc.deps.TeamRepo.GetByID(ctx, *user.TeamID)
	if err != nil {
		return fmt.Errorf("TeamUseCase - handleSoloTeamCleanup - TeamRepo.GetByID: %w", err)
	}

	members, err := uc.deps.UserRepo.GetByTeamID(ctx, *user.TeamID)
	if err != nil {
		return fmt.Errorf("TeamUseCase - handleSoloTeamCleanup - UserRepo.GetByTeamID: %w", err)
	}

	if !uc.shouldCleanupSoloTeam(user, members, oldTeam) {
		// If the team is banned and it is NOT a solo/auto-created team, block the operation.
		// A banned non-solo team requires admin intervention.
		if oldTeam.IsBanned {
			return httperr.ErrTeamBanned
		}
		return httperr.ErrUserAlreadyInTeam
	}

	if !confirmReset {
		return httperr.ErrConfirmationRequired
	}

	oldTeamID := *user.TeamID

	if err := uc.adjustSolveCountsForTeam(ctx, oldTeamID, uc.deps.ChallengeRepo.DecrementSolveCount); err != nil {
		return fmt.Errorf("TeamUseCase - handleSoloTeamCleanup - adjustSolveCountsForTeam: %w", err)
	}

	if err := uc.deps.SolveRepo.DeleteByTeamID(ctx, oldTeamID); err != nil {
		return fmt.Errorf("TeamUseCase - handleSoloTeamCleanup - SolveRepo.DeleteByTeamID: %w", err)
	}

	if err := uc.deps.SubmissionRepo.DeleteByTeamID(ctx, oldTeamID); err != nil {
		return fmt.Errorf("TeamUseCase - handleSoloTeamCleanup - SubmissionRepo.DeleteByTeamID: %w", err)
	}

	if err := uc.deps.AwardRepo.DeleteByTeamID(ctx, oldTeamID); err != nil {
		return fmt.Errorf("TeamUseCase - handleSoloTeamCleanup - AwardRepo.DeleteByTeamID: %w", err)
	}

	if err := uc.deps.UserRepo.UpdateTeamID(ctx, actorID, nil); err != nil {
		return fmt.Errorf("TeamUseCase - handleSoloTeamCleanup - UserRepo.UpdateTeamID: %w", err)
	}

	auditLog := &entity.TeamAuditLog{
		TeamID: oldTeamID,
		UserID: actorID,
		Action: entity.TeamActionDeleted,
		Details: map[string]any{
			"reason": "solo_team_cleanup",
		},
	}
	if err := uc.deps.TeamRepo.CreateAuditLog(ctx, auditLog); err != nil {
		return fmt.Errorf("TeamUseCase - handleSoloTeamCleanup - TeamRepo.CreateAuditLog: %w", err)
	}

	if err := uc.deps.TeamRepo.Delete(ctx, oldTeamID); err != nil {
		return fmt.Errorf("TeamUseCase - handleSoloTeamCleanup - TeamRepo.Delete: %w", err)
	}
	return nil
}

func (uc *TeamUseCase) shouldCleanupSoloTeam(user *entity.User, members []*entity.User, oldTeam *entity.Team) bool {
	return len(members) == 1 && members[0].ID == user.ID && (oldTeam.IsSolo || oldTeam.IsAutoCreated)
}

func (uc *TeamUseCase) DisbandTeam(ctx context.Context, captainID uuid.UUID) error {
	if _, err := uc.deps.Guard.RequireTeamSwitch(ctx); err != nil {
		return fmt.Errorf("TeamUseCase - DisbandTeam - Guard: %w", err)
	}
	var memberIDs []uuid.UUID
	err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		if _, err := uc.deps.Guard.RequireTeamSwitch(ctx); err != nil {
			return fmt.Errorf("TeamUseCase - DisbandTeam - Guard.RequireTeamSwitch: %w", err)
		}
		_, team, members, err := uc.disbandPrepare(ctx, captainID)
		if err != nil {
			return fmt.Errorf("TeamUseCase - DisbandTeam - disbandPrepare: %w", err)
		}
		if err := uc.disbandValidate(team, captainID); err != nil {
			return fmt.Errorf("TeamUseCase - DisbandTeam - disbandValidate: %w", err)
		}
		ids := make([]uuid.UUID, len(members))
		for i, m := range members {
			ids[i] = m.ID
		}
		memberIDs = ids
		return uc.disbandExecute(ctx, team, members, captainID)
	})
	if err != nil {
		return fmt.Errorf("TeamUseCase - DisbandTeam - TM.Run: %w", err)
	}
	for _, id := range memberIDs {
		uc.invalidateUserCache(ctx, id)
	}
	uc.invalidateScoreboardCache(ctx)
	return nil
}

func (uc *TeamUseCase) disbandPrepare(ctx context.Context, captainID uuid.UUID) (*entity.User, *entity.Team, []*entity.User, error) {
	if err := uc.deps.UserRepo.Lock(ctx, captainID); err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - disbandPrepare - UserRepo.Lock: %w", err)
	}
	user, err := uc.deps.UserRepo.GetByID(ctx, captainID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - disbandPrepare - UserRepo.GetByID: %w", err)
	}
	if user.TeamID == nil {
		return nil, nil, nil, httperr.ErrTeamNotFound
	}
	if err := uc.deps.TeamRepo.Lock(ctx, *user.TeamID); err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - disbandPrepare - TeamRepo.Lock: %w", err)
	}
	team, err := uc.deps.TeamRepo.GetByID(ctx, *user.TeamID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - disbandPrepare - TeamRepo.GetByID: %w", err)
	}
	members, err := uc.deps.UserRepo.GetByTeamID(ctx, team.ID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - disbandPrepare - UserRepo.GetByTeamID: %w", err)
	}
	return user, team, members, nil
}

func (uc *TeamUseCase) disbandValidate(team *entity.Team, captainID uuid.UUID) error {
	if team.CaptainID != captainID {
		return httperr.ErrNotCaptain
	}
	if team.IsBanned {
		return httperr.ErrTeamBanned
	}
	return nil
}

func (uc *TeamUseCase) disbandExecute(ctx context.Context, team *entity.Team, members []*entity.User, captainID uuid.UUID) error {
	if err := uc.adjustSolveCountsForTeam(ctx, team.ID, uc.deps.ChallengeRepo.DecrementSolveCount); err != nil {
		return fmt.Errorf("TeamUseCase - disbandExecute - adjustSolveCountsForTeam: %w", err)
	}
	if err := uc.deps.SolveRepo.DeleteByTeamID(ctx, team.ID); err != nil {
		return fmt.Errorf("TeamUseCase - disbandExecute - SolveRepo.DeleteByTeamID: %w", err)
	}
	if err := uc.deps.SubmissionRepo.DeleteByTeamID(ctx, team.ID); err != nil {
		return fmt.Errorf("TeamUseCase - disbandExecute - SubmissionRepo.DeleteByTeamID: %w", err)
	}
	if err := uc.deps.AwardRepo.DeleteByTeamID(ctx, team.ID); err != nil {
		return fmt.Errorf("TeamUseCase - disbandExecute - AwardRepo.DeleteByTeamID: %w", err)
	}
	for _, member := range members {
		if err := uc.deps.UserRepo.UpdateTeamID(ctx, member.ID, nil); err != nil {
			return fmt.Errorf("TeamUseCase - disbandExecute - UserRepo.UpdateTeamID: %w", err)
		}
	}
	auditLog := &entity.TeamAuditLog{
		TeamID: team.ID, UserID: captainID, Action: entity.TeamActionDeleted,
		Details: map[string]any{"reason": "disbanded_by_captain"},
	}
	if err := uc.deps.TeamRepo.CreateAuditLog(ctx, auditLog); err != nil {
		return fmt.Errorf("TeamUseCase - disbandExecute - TeamRepo.CreateAuditLog: %w", err)
	}
	if err := uc.deps.TeamRepo.Delete(ctx, team.ID); err != nil {
		return fmt.Errorf("TeamUseCase - disbandExecute - TeamRepo.Delete: %w", err)
	}
	return nil
}

func (uc *TeamUseCase) KickMember(ctx context.Context, captainID, targetUserID uuid.UUID) error {
	_, err := uc.deps.Guard.RequireTeamSwitch(ctx)
	if err != nil {
		return fmt.Errorf("TeamUseCase - KickMember - Guard: %w", err)
	}
	if captainID == targetUserID {
		return httperr.ErrCannotKickSelf
	}
	if err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		return uc.kickMemberTx(ctx, captainID, targetUserID)
	}); err != nil {
		return fmt.Errorf("TeamUseCase - KickMember - TM.Run: %w", err)
	}
	uc.invalidateUserCache(ctx, targetUserID)
	uc.invalidateScoreboardCache(ctx)
	return nil
}

func (uc *TeamUseCase) kickMemberTx(ctx context.Context, captainID, targetUserID uuid.UUID) error {
	if _, err := uc.deps.Guard.RequireTeamSwitch(ctx); err != nil {
		return fmt.Errorf("TeamUseCase - kickMemberTx - Guard.RequireTeamSwitch: %w", err)
	}
	comp, err := uc.deps.CompRepo.Get(ctx)
	if err != nil {
		return fmt.Errorf("TeamUseCase - kickMemberTx - CompetitionRepo.Get: %w", err)
	}
	captain, team, targetUser, err := uc.kickMemberPrepare(ctx, captainID, targetUserID)
	if err != nil {
		return fmt.Errorf("TeamUseCase - kickMemberTx - kickMemberPrepare: %w", err)
	}
	members, err := uc.deps.UserRepo.GetByTeamID(ctx, team.ID)
	if err != nil {
		return fmt.Errorf("TeamUseCase - kickMemberTx - UserRepo.GetByTeamID: %w", err)
	}
	if err := uc.kickMemberValidate(captain, team, targetUser, captainID, targetUserID, len(members), comp.MinTeamSize); err != nil {
		return fmt.Errorf("TeamUseCase - kickMemberTx - kickMemberValidate: %w", err)
	}
	return uc.kickMemberExecute(ctx, team.ID, captainID, targetUserID)
}

func (uc *TeamUseCase) kickMemberPrepare(ctx context.Context, captainID, targetUserID uuid.UUID) (*entity.User, *entity.Team, *entity.User, error) {
	if err := uc.deps.UserRepo.Lock(ctx, captainID); err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - kickMemberPrepare - UserRepo.Lock: %w", err)
	}
	captain, err := uc.deps.UserRepo.GetByID(ctx, captainID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - kickMemberPrepare - UserRepo.GetByID: %w", err)
	}
	if captain.TeamID == nil {
		return nil, nil, nil, httperr.ErrTeamNotFound
	}
	if err := uc.deps.TeamRepo.Lock(ctx, *captain.TeamID); err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - kickMemberPrepare - TeamRepo.Lock: %w", err)
	}
	team, err := uc.deps.TeamRepo.GetByID(ctx, *captain.TeamID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - kickMemberPrepare - TeamRepo.GetByID: %w", err)
	}
	targetUser, err := uc.deps.UserRepo.GetByID(ctx, targetUserID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - kickMemberPrepare - UserRepo.GetByID: %w", err)
	}
	return captain, team, targetUser, nil
}

func (uc *TeamUseCase) kickMemberValidate(_ *entity.User, team *entity.Team, targetUser *entity.User, captainID, _ uuid.UUID, memberCount, minTeamSize int) error {
	if team.CaptainID != captainID {
		return httperr.ErrNotCaptain
	}
	if targetUser.ID == team.CaptainID {
		return httperr.ErrCannotKickCaptain
	}
	if team.IsBanned {
		return httperr.ErrTeamBanned
	}
	if targetUser.TeamID == nil || *targetUser.TeamID != team.ID {
		return httperr.ErrUserNotFound
	}
	if minTeamSize > 0 && memberCount-1 < minTeamSize {
		return httperr.ErrTeamBelowMinSize
	}
	return nil
}

func (uc *TeamUseCase) kickMemberExecute(ctx context.Context, teamID, captainID, targetUserID uuid.UUID) error {
	if err := uc.deps.UserRepo.UpdateTeamID(ctx, targetUserID, nil); err != nil {
		return fmt.Errorf("TeamUseCase - kickMemberExecute - UserRepo.UpdateTeamID: %w", err)
	}
	auditLog := &entity.TeamAuditLog{
		TeamID: teamID, UserID: captainID, Action: entity.TeamActionMemberKicked,
		Details: map[string]any{"target_user_id": targetUserID.String()},
	}
	if err := uc.deps.TeamRepo.CreateAuditLog(ctx, auditLog); err != nil {
		return fmt.Errorf("TeamUseCase - kickMemberExecute - TeamRepo.CreateAuditLog: %w", err)
	}
	return nil
}

func (uc *TeamUseCase) BanTeam(ctx context.Context, teamID uuid.UUID, reason string, actorID uuid.UUID) error {
	var memberIDs []uuid.UUID
	err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		if err := uc.deps.TeamRepo.Lock(ctx, teamID); err != nil {
			return fmt.Errorf("TeamUseCase - BanTeam - TeamRepo.Lock: %w", err)
		}
		team, err := uc.deps.TeamRepo.GetByID(ctx, teamID)
		if err != nil {
			return fmt.Errorf("TeamUseCase - BanTeam - TeamRepo.GetByID: %w", err)
		}
		if team.IsBanned {
			return nil
		}
		members, err := uc.deps.UserRepo.GetByTeamID(ctx, teamID)
		if err != nil {
			return fmt.Errorf("TeamUseCase - BanTeam - UserRepo.GetByTeamID: %w", err)
		}
		ids := make([]uuid.UUID, len(members))
		for i, m := range members {
			ids[i] = m.ID
		}
		memberIDs = ids
		if err := uc.deps.TeamRepo.Ban(ctx, teamID, reason); err != nil {
			return fmt.Errorf("TeamUseCase - BanTeam - TeamRepo.Ban: %w", err)
		}
		if err := uc.adjustSolveCountsForTeam(ctx, teamID, uc.deps.ChallengeRepo.DecrementSolveCount); err != nil {
			return fmt.Errorf("TeamUseCase - BanTeam - adjustSolveCountsForTeam: %w", err)
		}
		auditLog := &entity.TeamAuditLog{
			TeamID:  teamID,
			UserID:  actorID,
			Action:  entity.TeamActionBanned,
			Details: map[string]any{"reason": reason},
		}
		if err := uc.deps.TeamRepo.CreateAuditLog(ctx, auditLog); err != nil {
			return fmt.Errorf("TeamUseCase - BanTeam - TeamRepo.CreateAuditLog: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("TeamUseCase - BanTeam - TM.Run: %w", err)
	}
	for _, id := range memberIDs {
		uc.invalidateUserCache(ctx, id)
	}
	uc.invalidateTeamCache(ctx, teamID)
	uc.invalidateScoreboardCacheForTeam(ctx, teamID)
	return nil
}

func (uc *TeamUseCase) UnbanTeam(ctx context.Context, teamID, actorID uuid.UUID) error {
	var memberIDs []uuid.UUID
	err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		if err := uc.deps.TeamRepo.Lock(ctx, teamID); err != nil {
			return fmt.Errorf("TeamUseCase - UnbanTeam - TeamRepo.Lock: %w", err)
		}
		team, err := uc.deps.TeamRepo.GetByID(ctx, teamID)
		if err != nil {
			return fmt.Errorf("TeamUseCase - UnbanTeam - TeamRepo.GetByID: %w", err)
		}
		if !team.IsBanned {
			return nil
		}
		members, err := uc.deps.UserRepo.GetByTeamID(ctx, teamID)
		if err != nil {
			return fmt.Errorf("TeamUseCase - UnbanTeam - UserRepo.GetByTeamID: %w", err)
		}
		ids := make([]uuid.UUID, len(members))
		for i, m := range members {
			ids[i] = m.ID
		}
		memberIDs = ids
		if err := uc.deps.TeamRepo.Unban(ctx, teamID); err != nil {
			return fmt.Errorf("TeamUseCase - UnbanTeam - TeamRepo.Unban: %w", err)
		}
		if err := uc.adjustSolveCountsForTeam(ctx, teamID, uc.deps.ChallengeRepo.IncrementSolveCount); err != nil {
			return fmt.Errorf("TeamUseCase - UnbanTeam - adjustSolveCountsForTeam: %w", err)
		}
		auditLog := &entity.TeamAuditLog{
			TeamID: teamID,
			UserID: actorID,
			Action: entity.TeamActionUnbanned,
		}
		if err := uc.deps.TeamRepo.CreateAuditLog(ctx, auditLog); err != nil {
			return fmt.Errorf("TeamUseCase - UnbanTeam - TeamRepo.CreateAuditLog: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("TeamUseCase - UnbanTeam - TM.Run: %w", err)
	}
	for _, id := range memberIDs {
		uc.invalidateUserCache(ctx, id)
	}
	uc.invalidateTeamCache(ctx, teamID)
	uc.invalidateScoreboardCacheForTeam(ctx, teamID)
	return nil
}

func (uc *TeamUseCase) SetHidden(ctx context.Context, teamID uuid.UUID, hidden bool) error {
	err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		_, err := uc.deps.TeamRepo.GetByID(ctx, teamID)
		if err != nil {
			return fmt.Errorf("TeamUseCase - SetHidden - TeamRepo.GetByID: %w", err)
		}
		if err := uc.deps.TeamRepo.SetHidden(ctx, teamID, hidden); err != nil {
			return fmt.Errorf("TeamUseCase - SetHidden - TeamRepo.SetHidden: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("TeamUseCase - SetHidden - TM.Run: %w", err)
	}
	uc.invalidateScoreboardCacheForTeam(ctx, teamID)
	return nil
}

func (uc *TeamUseCase) SetBracket(ctx context.Context, teamID uuid.UUID, bracketID *uuid.UUID) error {
	err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		_, err := uc.deps.TeamRepo.GetByID(ctx, teamID)
		if err != nil {
			return fmt.Errorf("TeamUseCase - SetBracket - TeamRepo.GetByID: %w", err)
		}
		if err := uc.deps.TeamRepo.SetBracket(ctx, teamID, bracketID); err != nil {
			return fmt.Errorf("TeamUseCase - SetBracket - TeamRepo.SetBracket: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("TeamUseCase - SetBracket - TM.Run: %w", err)
	}
	uc.invalidateScoreboardCacheForTeam(ctx, teamID)
	return nil
}

func (uc *TeamUseCase) invalidateScoreboardCache(ctx context.Context) {
	if uc.deps.ScoreboardCache != nil {
		uc.deps.ScoreboardCache.InvalidateAll(ctx)
	}
}

func (uc *TeamUseCase) invalidateScoreboardCacheForTeam(ctx context.Context, teamID uuid.UUID) {
	if uc.deps.ScoreboardCache != nil {
		uc.deps.ScoreboardCache.InvalidateForTeam(ctx, teamID)
	}
}

func (uc *TeamUseCase) invalidateUserCache(ctx context.Context, userID uuid.UUID) {
	if uc.deps.UserCache != nil {
		uc.deps.UserCache.InvalidateUser(ctx, userID)
	}
}

func (uc *TeamUseCase) invalidateTeamCache(ctx context.Context, teamID uuid.UUID) {
	if uc.deps.TeamCache != nil {
		_ = uc.deps.TeamCache.Del(ctx, cache.KeyTeam(teamID.String())) //nolint:errcheck // best-effort invalidation
	}
}

func (uc *TeamUseCase) ListTeams(ctx context.Context, search *string, page, perPage int) (*usecase.Paginated[*entity.Team], error) {
	result, err := usecase.FetchPage(ctx, page, perPage,
		func(ctx context.Context, limit, offset int) ([]*entity.Team, error) {
			return uc.deps.TeamRepo.Search(ctx, search, limit, offset)
		},
		func(ctx context.Context) (int64, error) {
			return uc.deps.TeamRepo.CountSearch(ctx, search)
		},
	)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - ListTeams: %w", err)
	}
	return result, nil
}

func (uc *TeamUseCase) GetTeamSolves(ctx context.Context, teamID uuid.UUID) ([]*entity.SolveWithDetails, error) {
	team, err := uc.deps.TeamRepo.GetByID(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - GetTeamSolves - TeamRepo.GetByID: %w", err)
	}
	if team.IsBanned {
		return nil, httperr.ErrTeamBanned
	}
	solves, err := uc.deps.SolveRepo.GetByTeamIDWithDetails(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - GetTeamSolves - SolveRepo.GetByTeamIDWithDetails: %w", err)
	}
	return solves, nil
}

func (uc *TeamUseCase) GetTeamFails(ctx context.Context, teamID uuid.UUID, page, perPage int) (*usecase.Paginated[*entity.SubmissionWithDetails], error) {
	team, err := uc.deps.TeamRepo.GetByID(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - GetTeamFails - TeamRepo.GetByID: %w", err)
	}
	if team.IsBanned {
		return nil, httperr.ErrTeamBanned
	}
	result, err := usecase.FetchPage(ctx, page, perPage,
		func(ctx context.Context, limit, offset int) ([]*entity.SubmissionWithDetails, error) {
			return uc.deps.SubmissionRepo.GetFailsByTeam(ctx, teamID, limit, offset)
		},
		func(ctx context.Context) (int64, error) {
			return uc.deps.SubmissionRepo.CountFailsByTeam(ctx, teamID)
		},
	)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - GetTeamFails: %w", err)
	}
	return result, nil
}

func (uc *TeamUseCase) GetTeamAwards(ctx context.Context, teamID uuid.UUID) ([]*entity.Award, error) {
	team, err := uc.deps.TeamRepo.GetByID(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - GetTeamAwards - TeamRepo.GetByID: %w", err)
	}
	if team.IsBanned {
		return nil, httperr.ErrTeamBanned
	}
	if uc.deps.AwardRepo == nil {
		return []*entity.Award{}, nil
	}
	awards, err := uc.deps.AwardRepo.GetByTeamID(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - GetTeamAwards - AwardRepo.GetByTeamID: %w", err)
	}
	return awards, nil
}

func (uc *TeamUseCase) AdminListTeams(ctx context.Context, search *string, page, perPage int) (*usecase.Paginated[*entity.Team], error) {
	result, err := usecase.FetchPage(ctx, page, perPage,
		func(ctx context.Context, limit, offset int) ([]*entity.Team, error) {
			return uc.deps.TeamRepo.SearchAdmin(ctx, search, limit, offset)
		},
		func(ctx context.Context) (int64, error) {
			return uc.deps.TeamRepo.CountSearchAdmin(ctx, search)
		},
	)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - AdminListTeams: %w", err)
	}
	return result, nil
}

//nolint:gocognit,gocyclo // optional-field patching with captain/bracket validation
func (uc *TeamUseCase) AdminUpdate(ctx context.Context, teamID uuid.UUID, name *string, captainID, bracketID *uuid.UUID, isHidden *bool) (*entity.Team, error) {
	if err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		if captainID != nil {
			if err := uc.deps.UserRepo.Lock(ctx, *captainID); err != nil {
				return fmt.Errorf("TeamUseCase - AdminUpdate - UserRepo.Lock: %w", err)
			}
			candidate, err := uc.deps.UserRepo.GetByID(ctx, *captainID)
			if err != nil {
				return fmt.Errorf("TeamUseCase - AdminUpdate - UserRepo.GetByID: %w", err)
			}
			if candidate.TeamID == nil || *candidate.TeamID != teamID {
				return httperr.ErrNewCaptainNotInTeam
			}
			if candidate.IsBanned {
				return httperr.ErrUserBanned
			}
		}
		if err := uc.deps.TeamRepo.Lock(ctx, teamID); err != nil {
			return fmt.Errorf("TeamUseCase - AdminUpdate - TeamRepo.Lock: %w", err)
		}
		if name != nil {
			currentTeam, err := uc.deps.TeamRepo.GetByID(ctx, teamID)
			if err != nil {
				return fmt.Errorf("TeamUseCase - AdminUpdate - TeamRepo.GetByID: %w", err)
			}
			if currentTeam.Name != *name {
				if err := uc.validateTeamNameAvailable(ctx, *name); err != nil {
					return fmt.Errorf("TeamUseCase - AdminUpdate - validateTeamNameAvailable: %w", err)
				}
			}
		}
		if err := uc.deps.TeamRepo.UpdateAdmin(ctx, teamID, name, captainID, bracketID, isHidden); err != nil {
			return fmt.Errorf("TeamUseCase - AdminUpdate - TeamRepo.UpdateAdmin: %w", err)
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("TeamUseCase - AdminUpdate - TM.Run: %w", err)
	}
	team, err := uc.deps.TeamRepo.GetByID(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - AdminUpdate - TeamRepo.GetByID: %w", err)
	}
	uc.invalidateScoreboardCache(ctx)
	return team, nil
}

//nolint:gocognit
func (uc *TeamUseCase) AdminDelete(ctx context.Context, teamID uuid.UUID) error {
	var memberIDs []uuid.UUID
	err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		if err := uc.deps.TeamRepo.Lock(ctx, teamID); err != nil {
			return fmt.Errorf("TeamUseCase - AdminDelete - TeamRepo.Lock: %w", err)
		}
		members, err := uc.deps.UserRepo.GetByTeamID(ctx, teamID)
		if err != nil {
			return fmt.Errorf("TeamUseCase - AdminDelete - UserRepo.GetByTeamID: %w", err)
		}
		if err := uc.adjustSolveCountsForTeam(ctx, teamID, uc.deps.ChallengeRepo.DecrementSolveCount); err != nil {
			return fmt.Errorf("TeamUseCase - AdminDelete - adjustSolveCountsForTeam: %w", err)
		}
		if err := uc.deps.SolveRepo.DeleteByTeamID(ctx, teamID); err != nil {
			return fmt.Errorf("TeamUseCase - AdminDelete - SolveRepo.DeleteByTeamID: %w", err)
		}
		ids := make([]uuid.UUID, len(members))
		for i, m := range members {
			ids[i] = m.ID
			if err := uc.deps.UserRepo.UpdateTeamID(ctx, m.ID, nil); err != nil {
				return fmt.Errorf("TeamUseCase - AdminDelete - UserRepo.UpdateTeamID: %w", err)
			}
		}
		memberIDs = ids
		if err := uc.deps.TeamRepo.Delete(ctx, teamID); err != nil {
			return fmt.Errorf("TeamUseCase - AdminDelete - TeamRepo.Delete: %w", err)
		}
		auditLog := &entity.TeamAuditLog{
			TeamID:  teamID,
			UserID:  uuid.Nil,
			Action:  entity.TeamActionDeleted,
			Details: map[string]any{"reason": "deleted_by_admin"},
		}
		if err := uc.deps.TeamRepo.CreateAuditLog(ctx, auditLog); err != nil {
			return fmt.Errorf("TeamUseCase - AdminDelete - TeamRepo.CreateAuditLog: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("TeamUseCase - AdminDelete - TM.Run: %w", err)
	}
	for _, id := range memberIDs {
		uc.invalidateUserCache(ctx, id)
	}
	uc.invalidateScoreboardCache(ctx)
	return nil
}

func (uc *TeamUseCase) AdminGetMembers(ctx context.Context, teamID uuid.UUID) ([]*entity.User, error) {
	if _, err := uc.deps.TeamRepo.GetByID(ctx, teamID); err != nil {
		return nil, fmt.Errorf("TeamUseCase - AdminGetMembers - TeamRepo.GetByID: %w", err)
	}
	members, err := uc.deps.UserRepo.GetByTeamID(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - AdminGetMembers - UserRepo.GetByTeamID: %w", err)
	}
	return members, nil
}

func (uc *TeamUseCase) AdminAddMember(ctx context.Context, teamID, userID uuid.UUID) error {
	if err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		return uc.adminAddMemberTx(ctx, teamID, userID)
	}); err != nil {
		return fmt.Errorf("TeamUseCase - AdminAddMember - TM.Run: %w", err)
	}
	uc.invalidateUserCache(ctx, userID)
	uc.invalidateScoreboardCache(ctx)
	return nil
}

//nolint:gocyclo // size-limit + bracket + competition-mode checks combine here
func (uc *TeamUseCase) adminAddMemberTx(ctx context.Context, teamID, userID uuid.UUID) error {
	if err := uc.deps.UserRepo.Lock(ctx, userID); err != nil {
		return fmt.Errorf("TeamUseCase - AdminAddMember - UserRepo.Lock: %w", err)
	}
	if err := uc.deps.TeamRepo.Lock(ctx, teamID); err != nil {
		return fmt.Errorf("TeamUseCase - AdminAddMember - TeamRepo.Lock: %w", err)
	}
	team, err := uc.deps.TeamRepo.GetByID(ctx, teamID)
	if err != nil {
		return fmt.Errorf("TeamUseCase - AdminAddMember - TeamRepo.GetByID: %w", err)
	}
	if team.IsSolo {
		return httperr.ErrCannotAddToSoloTeam
	}
	if team.IsBanned {
		return httperr.ErrTeamBanned
	}
	user, err := uc.deps.UserRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("TeamUseCase - AdminAddMember - UserRepo.GetByID: %w", err)
	}
	if user.IsBanned {
		return httperr.ErrUserBanned
	}
	if user.TeamID != nil {
		return httperr.ErrTeamConflict
	}
	members, err := uc.deps.UserRepo.GetByTeamID(ctx, teamID)
	if err != nil {
		return fmt.Errorf("TeamUseCase - AdminAddMember - UserRepo.GetByTeamID: %w", err)
	}
	comp, err := uc.deps.CompRepo.Get(ctx)
	if err != nil {
		return fmt.Errorf("TeamUseCase - AdminAddMember - CompetitionRepo.Get: %w", err)
	}
	if !comp.Mode.AllowsTeams() {
		return httperr.ErrTeamsNotAllowed
	}
	maxSize := comp.MaxTeamSize
	if maxSize <= 0 {
		maxSize = uc.deps.DefaultMaxTeamSize
	}
	if len(members) >= maxSize {
		return httperr.ErrTeamFull
	}
	if err := uc.deps.UserRepo.UpdateTeamID(ctx, userID, &teamID); err != nil {
		return fmt.Errorf("TeamUseCase - AdminAddMember - UserRepo.UpdateTeamID: %w", err)
	}
	auditLog := &entity.TeamAuditLog{
		TeamID:  teamID,
		UserID:  userID,
		Action:  entity.TeamActionJoined,
		Details: map[string]any{"reason": "added_by_admin"},
	}
	if err := uc.deps.TeamRepo.CreateAuditLog(ctx, auditLog); err != nil {
		return fmt.Errorf("TeamUseCase - AdminAddMember - TeamRepo.CreateAuditLog: %w", err)
	}
	return nil
}

// AdminRemoveMember removes a user from a team by admin action.
// Unlike the regular Leave/Kick flow, this intentionally bypasses MinTeamSize enforcement:
// admins may need to restructure teams regardless of size constraints.
//
//nolint:gocognit
func (uc *TeamUseCase) AdminRemoveMember(ctx context.Context, teamID, userID uuid.UUID) error {
	if err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		if err := uc.deps.UserRepo.Lock(ctx, userID); err != nil {
			return fmt.Errorf("TeamUseCase - AdminRemoveMember - UserRepo.Lock: %w", err)
		}
		if err := uc.deps.TeamRepo.Lock(ctx, teamID); err != nil {
			return fmt.Errorf("TeamUseCase - AdminRemoveMember - TeamRepo.Lock: %w", err)
		}
		user, err := uc.deps.UserRepo.GetByID(ctx, userID)
		if err != nil {
			return fmt.Errorf("TeamUseCase - AdminRemoveMember - UserRepo.GetByID: %w", err)
		}
		if user.TeamID == nil || *user.TeamID != teamID {
			return httperr.ErrTeamMemberNotFound
		}
		team, err := uc.deps.TeamRepo.GetByID(ctx, teamID)
		if err != nil {
			return fmt.Errorf("TeamUseCase - AdminRemoveMember - TeamRepo.GetByID: %w", err)
		}
		if team.CaptainID == userID {
			return httperr.ErrCaptainCannotLeave
		}
		if err := uc.deps.UserRepo.UpdateTeamID(ctx, userID, nil); err != nil {
			return fmt.Errorf("TeamUseCase - AdminRemoveMember - UserRepo.UpdateTeamID: %w", err)
		}
		auditLog := &entity.TeamAuditLog{
			TeamID:  teamID,
			UserID:  userID,
			Action:  entity.TeamActionMemberKicked,
			Details: map[string]any{"reason": "removed_by_admin"},
		}
		if err := uc.deps.TeamRepo.CreateAuditLog(ctx, auditLog); err != nil {
			return fmt.Errorf("TeamUseCase - AdminRemoveMember - TeamRepo.CreateAuditLog: %w", err)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("TeamUseCase - AdminRemoveMember - TM.Run: %w", err)
	}
	uc.invalidateUserCache(ctx, userID)
	uc.invalidateScoreboardCache(ctx)
	return nil
}

func (uc *TeamUseCase) UpdateMyTeam(ctx context.Context, captainID uuid.UUID, name string) (*entity.Team, error) {
	var team *entity.Team
	err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		var err error
		team, err = uc.updateMyTeamTx(ctx, captainID, name)
		if err != nil {
			return fmt.Errorf("TeamUseCase - UpdateMyTeam - updateMyTeamTx: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - UpdateMyTeam - TM.Run: %w", err)
	}
	if team != nil {
		uc.invalidateTeamCache(ctx, team.ID)
		uc.invalidateScoreboardCache(ctx)
	}
	return team, nil
}

func (uc *TeamUseCase) updateMyTeamTx(ctx context.Context, captainID uuid.UUID, name string) (*entity.Team, error) {
	if err := uc.deps.UserRepo.Lock(ctx, captainID); err != nil {
		return nil, fmt.Errorf("TeamUseCase - updateMyTeamTx - UserRepo.Lock: %w", err)
	}
	user, err := uc.deps.UserRepo.GetByID(ctx, captainID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - updateMyTeamTx - UserRepo.GetByID: %w", err)
	}
	if user.TeamID == nil {
		return nil, httperr.ErrTeamNotFound
	}
	if err := uc.deps.TeamRepo.Lock(ctx, *user.TeamID); err != nil {
		return nil, fmt.Errorf("TeamUseCase - updateMyTeamTx - TeamRepo.Lock: %w", err)
	}
	team, err := uc.deps.TeamRepo.GetByID(ctx, *user.TeamID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - updateMyTeamTx - TeamRepo.GetByID: %w", err)
	}
	if team.IsBanned {
		return nil, httperr.ErrTeamBanned
	}
	if team.CaptainID != captainID {
		return nil, httperr.ErrNotCaptain
	}
	if team.Name != name {
		if err := uc.validateTeamNameAvailable(ctx, name); err != nil {
			return nil, fmt.Errorf("TeamUseCase - updateMyTeamTx - validateTeamNameAvailable: %w", err)
		}
	}
	if err := uc.deps.TeamRepo.UpdateName(ctx, team.ID, name); err != nil {
		return nil, fmt.Errorf("TeamUseCase - updateMyTeamTx - TeamRepo.UpdateName: %w", err)
	}
	team.Name = name
	return team, nil
}

func (uc *TeamUseCase) GetInviteToken(ctx context.Context, captainID uuid.UUID) (*entity.Team, error) {
	user, err := uc.deps.UserRepo.GetByID(ctx, captainID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - GetInviteToken - UserRepo.GetByID: %w", err)
	}
	if user.TeamID == nil {
		return nil, httperr.ErrTeamNotFound
	}
	team, err := uc.deps.TeamRepo.GetByID(ctx, *user.TeamID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - GetInviteToken - TeamRepo.GetByID: %w", err)
	}
	if team.IsSolo {
		return nil, httperr.ErrTeamNotFound
	}
	if team.CaptainID != captainID {
		return nil, httperr.ErrNotCaptain
	}
	if team.IsBanned {
		return nil, httperr.ErrTeamBanned
	}
	return team, nil
}
