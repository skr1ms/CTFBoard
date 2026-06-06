package team

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/cacheutil"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/guard"
)

// Create creates a new team with captainID as captain. The actual work runs in a
// transaction via createTx; cache entries for the captain and the scoreboard are
// invalidated after a successful commit.
func (uc *TeamUseCase) Create(ctx context.Context, name string, captainID uuid.UUID, isSolo, confirmReset bool) (*domain.Team, error) {
	var team *domain.Team

	err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		var errCreate error

		team, errCreate = uc.createTx(ctx, name, captainID, isSolo, confirmReset)
		if errCreate != nil {
			return fmt.Errorf("TeamUseCase - Create - createTx: %w", errCreate)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - Create - TM.Run: %w", err)
	}

	cacheutil.InvalidateUser(ctx, uc.deps.UserCache, captainID)
	cacheutil.InvalidateScoreboardForTeam(ctx, uc.deps.ScoreboardCache, team.ID)

	return team, nil
}

// checkMaxTeams enforces the max-teams setting atomically using a PostgreSQL advisory lock
// so that concurrent team creations cannot both pass the count check simultaneously.
func (uc *TeamUseCase) checkMaxTeams(ctx context.Context) error {
	appSettings, err := uc.deps.SettingsGetter.Get(ctx)
	if err != nil {
		return fmt.Errorf("TeamUseCase - checkMaxTeams - SettingsGetter.Get: %w", err)
	}

	if appSettings.MaxTeams <= 0 {
		return nil
	}

	const maxTeamsLockKey int64 = 0x4354467465616D73
	if err := uc.deps.TeamRepo.AcquireAdvisoryLock(ctx, maxTeamsLockKey); err != nil {
		return fmt.Errorf("TeamUseCase - checkMaxTeams - AcquireAdvisoryLock: %w", err)
	}

	currentCount, err := uc.deps.TeamRepo.CountActiveTeams(ctx)
	if err != nil {
		return fmt.Errorf("TeamUseCase - checkMaxTeams - TeamRepo.CountActiveTeams: %w", err)
	}

	if currentCount >= appSettings.MaxTeams {
		return apperr.ErrMaxTeamsReached
	}

	return nil
}

// createTx runs the full team-creation sequence inside an existing transaction
// It checks that the competition is in a state that allows team operations and that
// the requested mode (teams/solo) is permitted, then acquires an advisory lock before
// counting active teams to enforce the MaxTeams limit without TOCTOU races. After
// locking the captain's user row it validates name uniqueness. If the captain is
// already a member of a solo or auto-created team, handleSoloTeamCleanup is called
// to wipe that team's data (requires confirmReset=true). Finally it persists the new
// team, assigns the captain, and writes an audit log entry.
func (uc *TeamUseCase) createTx(ctx context.Context, name string, captainID uuid.UUID, isSolo, confirmReset bool) (*domain.Team, error) {
	comp, err := uc.deps.CompRepo.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - createTx - CompetitionRepo.Get: %w", err)
	}

	if err := guard.ValidateTeamSwitchState(comp); err != nil {
		return nil, err
	}

	if !comp.Mode.AllowsTeams() {
		return nil, apperr.ErrTeamsNotAllowed
	}

	if isSolo && !comp.Mode.AllowsSolo() {
		return nil, apperr.ErrSoloModeNotAllowed
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
		return nil, apperr.ErrUserBanned
	}

	if user.WasInBannedTeam {
		return nil, apperr.ErrUserWasInBannedTeam
	}

	if user.TeamID != nil {
		err := uc.handleSoloTeamCleanup(ctx, user, captainID, confirmReset, nil)
		if err != nil {
			return nil, fmt.Errorf("TeamUseCase - createTx - handleSoloTeamCleanup: %w", err)
		}
	}

	expiresAt := time.Now().Add(defaultInviteTokenTTL)

	team := &domain.Team{
		Name:                 name,
		InviteToken:          uuid.New(),
		CaptainID:            captainID,
		IsSolo:               isSolo,
		InviteTokenExpiresAt: &expiresAt,
	}
	if err := uc.deps.TeamRepo.Create(ctx, team); err != nil {
		return nil, fmt.Errorf("TeamUseCase - createTx - TeamRepo.Create: %w", err)
	}

	if err := uc.deps.UserRepo.UpdateTeamID(ctx, captainID, &team.ID); err != nil {
		return nil, fmt.Errorf("TeamUseCase - createTx - UserRepo.UpdateTeamID: %w", err)
	}

	auditLog := &domain.TeamAuditLog{TeamID: team.ID, UserID: &captainID, Action: domain.TeamActionCreated}
	if err := uc.deps.TeamRepo.CreateAuditLog(ctx, auditLog); err != nil {
		return nil, fmt.Errorf("TeamUseCase - createTx - TeamRepo.CreateAuditLog: %w", err)
	}

	return team, nil
}

func (uc *TeamUseCase) validateTeamNameAvailable(ctx context.Context, name string) error {
	_, err := uc.deps.TeamRepo.GetByName(ctx, name)
	if err == nil {
		return apperr.ErrTeamAlreadyExists
	}

	if !errors.Is(err, apperr.ErrTeamNotFound) {
		return fmt.Errorf("TeamUseCase - validateTeamNameAvailable - TeamRepo.GetByName: %w", err)
	}

	return nil
}
