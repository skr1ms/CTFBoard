package team

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/scoring"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
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

// TryCreate attempts to create a team for captainID. If the user is not yet in any
// team the creation proceeds immediately and the result contains the new team. If the
// user already belongs to a solo or auto-created team, the function returns a result
// with RequiresConfirm=true and a summary of the data that would be lost, giving the
// caller the opportunity to present a confirmation step before calling ConfirmCreate
// Hard errors (banned user, competition state, etc.) are still returned as errors.
func (uc *TeamUseCase) TryCreate(ctx context.Context, name string, captainID uuid.UUID, isSolo bool) (*usecase.TeamCreateResult, error) {
	comp, err := uc.deps.Guard.RequireTeamSwitchAndTeamsMode(ctx)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - TryCreate - Guard: %w", err)
	}

	if isSolo && !comp.Mode.AllowsSolo() {
		return nil, apperr.ErrSoloModeNotAllowed
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

// tryCreateWhenInTeam handles the case where the user is already a member of a
// team when TryCreate is called. It locks the user row inside a transaction,
// re-fetches the membership state to close TOCTOU races, then inspects the old
// team. If the old team is a multi-member or regular (non-solo, non-auto) team
// the function returns ErrUserAlreadyInTeam immediately. For an eligible
// solo/auto-created single-member team it collects the affected data summary
// (points, solve count, awards, hint unlocks) and returns RequiresConfirm=true
// so the caller can present a confirmation step before proceeding to ConfirmCreate.
// No data is deleted here - actual cleanup happens in createTx -> handleSoloTeamCleanup.
func (uc *TeamUseCase) tryCreateWhenInTeam(ctx context.Context, captainID uuid.UUID) (*usecase.TeamCreateResult, error) {
	var result *usecase.TeamCreateResult

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
			return apperr.ErrUserBanned
		}

		if freshUser.TeamID == nil {
			return apperr.ErrTeamNotFound
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
			return apperr.ErrUserAlreadyInTeam
		}

		points, err := uc.deps.SolveRepo.GetTeamScore(ctx, *freshUser.TeamID)
		if err != nil {
			return fmt.Errorf("TeamUseCase - tryCreateWhenInTeam - SolveRepo.GetTeamScore: %w", err)
		}

		solveCount := 0

		if uc.deps.SolveRepo != nil {
			if solves, err := uc.deps.SolveRepo.GetByTeamIDWithDetails(ctx, *freshUser.TeamID); err == nil {
				solveCount = len(solves)
			}
		}

		awardsTotal := 0

		if uc.deps.AwardRepo != nil {
			if total, err := uc.deps.AwardRepo.GetTeamTotalAwards(ctx, *freshUser.TeamID); err == nil {
				awardsTotal = total
			}
		}

		hintUnlockCount := 0

		if uc.deps.HintRepo != nil {
			if n, err := uc.deps.HintRepo.CountByTeamID(ctx, *freshUser.TeamID); err == nil {
				hintUnlockCount = n
			}
		}

		result = &usecase.TeamCreateResult{
			RequiresConfirm:    true,
			ConfirmationReason: usecase.ConfirmReasonSoloTeamReset,
			AffectedData:       &usecase.TeamCreateAffectedData{Points: points, SolveCount: solveCount, AwardsTotal: awardsTotal, HintUnlockCount: hintUnlockCount},
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - TryCreate - TM.Run: %w", err)
	}

	return result, nil
}

func (uc *TeamUseCase) ConfirmCreate(ctx context.Context, name string, captainID uuid.UUID, isSolo bool) (*domain.Team, error) {
	return uc.Create(ctx, name, captainID, isSolo, true)
}

// CreateSoloTeam creates a solo wrapper team for userID. Delegates to createSoloTeamTx
// inside a transaction; invalidates the user cache on success.
func (uc *TeamUseCase) CreateSoloTeam(ctx context.Context, userID uuid.UUID, confirmReset bool) (*domain.Team, error) {
	var team *domain.Team

	err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		var errCreate error

		team, errCreate = uc.createSoloTeamTx(ctx, userID, confirmReset, false, false)
		if errCreate != nil {
			return fmt.Errorf("TeamUseCase - CreateSoloTeam - createSoloTeamTx: %w", errCreate)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - CreateSoloTeam - TM.Run: %w", err)
	}

	cacheutil.InvalidateUser(ctx, uc.deps.UserCache, userID)

	return team, nil
}

func (uc *TeamUseCase) requireSoloModeOnly(ctx context.Context) error {
	comp, err := uc.deps.CompRepo.Get(ctx)
	if err != nil {
		return fmt.Errorf("TeamUseCase - requireSoloModeOnly - CompetitionRepo.Get: %w", err)
	}

	if !comp.Mode.AllowsSolo() {
		return apperr.ErrSoloModeNotAllowed
	}

	return nil
}

// CreateSoloTeamForNewUser creates a solo team for a freshly registered user.
// Unlike CreateSoloTeam it skips the team-switch guard (the user has no team yet)
// and passes skipTeamSwitchCheck=true to createSoloTeamTx.
func (uc *TeamUseCase) CreateSoloTeamForNewUser(ctx context.Context, userID uuid.UUID) (*domain.Team, error) {
	if err := uc.requireSoloModeOnly(ctx); err != nil {
		return nil, fmt.Errorf("TeamUseCase - CreateSoloTeamForNewUser - requireSoloModeOnly: %w", err)
	}

	var team *domain.Team

	err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		var errCreate error

		team, errCreate = uc.createSoloTeamTx(ctx, userID, false, true, true)
		if errCreate != nil {
			return fmt.Errorf("TeamUseCase - CreateSoloTeamForNewUser - createSoloTeamTx: %w", errCreate)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - CreateSoloTeamForNewUser - TM.Run: %w", err)
	}

	cacheutil.InvalidateUser(ctx, uc.deps.UserCache, userID)

	return team, nil
}

// createSoloTeamTx creates a solo wrapper team for userID inside an existing transaction
// Name resolution prefers the username, falls back to "<username> (Solo)", then to a
// UUID-suffixed variant. If the final insert still hits a unique-violation it retries up
// to maxSoloNameRetries times - each iteration generates a fresh placeholder token for
// the suffix. Any existing solo or auto-created team is cleaned up via
// handleSoloTeamCleanup before the new team is created. After a successful insert the
// invite token is replaced with the team's own ID (making it a stable, never-expiring
// token), and the user is assigned to the team.
func (uc *TeamUseCase) createSoloTeamTx(ctx context.Context, userID uuid.UUID, confirmReset, isAutoCreated, skipTeamSwitchCheck bool) (*domain.Team, error) {
	if skipTeamSwitchCheck {
		err := uc.requireSoloModeOnly(ctx)
		if err != nil {
			return nil, fmt.Errorf("TeamUseCase - createSoloTeamTx - requireSoloModeOnly: %w", err)
		}
	} else {
		if _, err := uc.deps.Guard.RequireTeamSwitchAndSoloMode(ctx); err != nil {
			return nil, fmt.Errorf("TeamUseCase - createSoloTeamTx - Guard.RequireTeamSwitchAndSoloMode: %w", err)
		}
	}

	if _, err := uc.deps.CompRepo.Get(ctx); err != nil {
		return nil, fmt.Errorf("TeamUseCase - createSoloTeamTx - CompetitionRepo.Get: %w", err)
	}

	if err := uc.checkMaxTeams(ctx); err != nil {
		return nil, fmt.Errorf("TeamUseCase - createSoloTeamTx - checkMaxTeams: %w", err)
	}

	if err := uc.deps.UserRepo.Lock(ctx, userID); err != nil {
		return nil, fmt.Errorf("TeamUseCase - createSoloTeamTx - UserRepo.Lock: %w", err)
	}

	user, err := uc.deps.UserRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - createSoloTeamTx - UserRepo.GetByID: %w", err)
	}

	if user.IsBanned {
		return nil, apperr.ErrUserBanned
	}

	if user.WasInBannedTeam {
		return nil, apperr.ErrUserWasInBannedTeam
	}

	if user.TeamID != nil {
		err := uc.handleSoloTeamCleanup(ctx, user, userID, confirmReset, nil)
		if err != nil {
			return nil, fmt.Errorf("TeamUseCase - createSoloTeamTx - handleSoloTeamCleanup: %w", err)
		}
	}

	const maxSoloNameRetries = 15

	var team *domain.Team

	for attempt := range maxSoloNameRetries {
		placeholderToken := uuid.New()
		expiresAt := time.Now().Add(defaultInviteTokenTTL)

		team = &domain.Team{
			Name:                 user.Username,
			InviteToken:          placeholderToken,
			CaptainID:            userID,
			IsSolo:               true,
			IsAutoCreated:        isAutoCreated,
			InviteTokenExpiresAt: &expiresAt,
		}
		if _, err := uc.deps.TeamRepo.GetByName(ctx, team.Name); err == nil {
			fallback := fmt.Sprintf("%s (Solo)", user.Username)
			if _, err := uc.deps.TeamRepo.GetByName(ctx, fallback); err == nil {
				fallback = fmt.Sprintf("%s-%s", user.Username, placeholderToken.String())
			}

			team.Name = fallback
		}

		err := uc.deps.TeamRepo.Create(ctx, team)
		if err == nil {
			break
		}

		if !errors.Is(err, apperr.ErrTeamAlreadyExists) || attempt == maxSoloNameRetries-1 {
			return nil, fmt.Errorf("TeamUseCase - createSoloTeamTx - TeamRepo.Create: %w", err)
		}
	}

	if err := uc.deps.TeamRepo.UpdateInviteToken(ctx, team.ID, team.ID, nil); err != nil {
		return nil, fmt.Errorf("TeamUseCase - createSoloTeamTx - TeamRepo.UpdateInviteToken: %w", err)
	}

	team.InviteToken = team.ID

	team.InviteTokenExpiresAt = nil
	if err := uc.deps.UserRepo.UpdateTeamID(ctx, userID, &team.ID); err != nil {
		return nil, fmt.Errorf("TeamUseCase - createSoloTeamTx - UserRepo.UpdateTeamID: %w", err)
	}

	auditLog := &domain.TeamAuditLog{
		TeamID: team.ID, UserID: &userID, Action: domain.TeamActionCreated,
		Details: map[string]any{"mode": "solo"},
	}
	if err := uc.deps.TeamRepo.CreateAuditLog(ctx, auditLog); err != nil {
		return nil, fmt.Errorf("TeamUseCase - createSoloTeamTx - TeamRepo.CreateAuditLog: %w", err)
	}

	return team, nil
}

// getChallengeIDsForTeam returns the deduplicated set of challenge IDs the team has
// solved, preserving first-seen order. Returns nil when SolveRepo is not wired.
func (uc *TeamUseCase) getChallengeIDsForTeam(ctx context.Context, teamID uuid.UUID) ([]uuid.UUID, error) {
	if uc.deps.SolveRepo == nil {
		return nil, nil
	}

	solves, err := uc.deps.SolveRepo.GetByTeamIDWithDetails(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - getChallengeIDsForTeam - SolveRepo.GetByTeamIDWithDetails: %w", err)
	}

	rawIDs := make([]uuid.UUID, 0, len(solves))

	for _, s := range solves {
		rawIDs = append(rawIDs, s.ChallengeID)
	}

	return domain.UniqueUUIDs(rawIDs), nil
}

func (uc *TeamUseCase) adjustSolveCountsForChallenges(ctx context.Context, challengeIDs []uuid.UUID) error {
	if uc.deps.ChallengeRepo == nil || len(challengeIDs) == 0 {
		return nil
	}

	var (
		getSolves   func(context.Context, []uuid.UUID) ([]*scoring.SolveRowForPointsRecalc, error)
		batchUpdate func(context.Context, []uuid.UUID, []int) error
	)

	if uc.deps.SolveRepo != nil {
		getSolves = scoring.MapSolvesForRecalcFn(
			uc.deps.SolveRepo.GetSolvesForPointsRecalc,
			scoring.DefaultSolveMapper,
			"TeamUseCase - adjustSolveCountsForChallenges",
		)
		batchUpdate = uc.deps.SolveRepo.BatchUpdateSolvePoints
	}

	return scoring.AdjustDynamicScores(
		ctx, challengeIDs, uc.deps.ChallengeRepo,
		getSolves, batchUpdate,
		scoring.GetDecayFn(ctx, uc.deps.CompParamUC),
	)
}

func (uc *TeamUseCase) adjustSolveCountsForTeam(ctx context.Context, teamID uuid.UUID) error {
	challengeIDs, err := uc.getChallengeIDsForTeam(ctx, teamID)
	if err != nil {
		return err
	}

	return uc.adjustSolveCountsForChallenges(ctx, challengeIDs)
}

// orderTeamLockIDs returns the two team IDs in lexicographic string order so that
// callers always acquire advisory/row locks in a consistent sequence and avoid
// deadlocks. When newTeamID is nil the second return value is uuid.Nil.
func orderTeamLockIDs(oldTeamID uuid.UUID, newTeamID *uuid.UUID) (first, second uuid.UUID) {
	if newTeamID == nil {
		return oldTeamID, uuid.Nil
	}

	if oldTeamID.String() < newTeamID.String() {
		return oldTeamID, *newTeamID
	}

	return *newTeamID, oldTeamID
}

// handleSoloTeamCleanup removes the user's current solo or auto-created team inside
// the transaction. To prevent deadlocks when two operations touch both the old and new
// team rows, advisory locks are acquired in a deterministic order derived from the
// lexicographic comparison of both team UUIDs (orderTeamLockIDs). Once locked, the
// function verifies the team is still eligible for cleanup (single-member solo/auto
// team). If confirmReset is false it returns ErrConfirmationRequired without making any
// changes. Otherwise it deletes all solves, submissions, awards, and hint unlocks for
// the old team, calls adjustSolveCountsForChallenges to keep scoring consistent, detaches
// the user, writes an audit log, and hard-deletes the old team record.
func (uc *TeamUseCase) handleSoloTeamCleanup(ctx context.Context, user *domain.User, actorID uuid.UUID, confirmReset bool, newTeamID *uuid.UUID) error {
	if user.TeamID == nil {
		return nil
	}

	firstID, secondID := orderTeamLockIDs(*user.TeamID, newTeamID)
	if err := uc.deps.TeamRepo.Lock(ctx, firstID); err != nil {
		return fmt.Errorf("TeamUseCase - handleSoloTeamCleanup - TeamRepo.Lock: %w", err)
	}
	if secondID != uuid.Nil {
		err := uc.deps.TeamRepo.Lock(ctx, secondID)
		if err != nil {
			return fmt.Errorf("TeamUseCase - handleSoloTeamCleanup - TeamRepo.Lock(second): %w", err)
		}
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
		if oldTeam.IsBanned {
			return apperr.ErrTeamBanned
		}

		return apperr.ErrUserAlreadyInTeam
	}

	if oldTeam.IsBanned {
		return apperr.ErrTeamBanned
	}

	if !confirmReset {
		return apperr.ErrConfirmationRequired
	}

	oldTeamID := *user.TeamID

	challengeIDs, err := uc.getChallengeIDsForTeam(ctx, oldTeamID)
	if err != nil {
		return fmt.Errorf("TeamUseCase - handleSoloTeamCleanup - getChallengeIDsForTeam: %w", err)
	}

	if err := uc.cascadeDelete(ctx, oldTeamID); err != nil {
		return fmt.Errorf("TeamUseCase - handleSoloTeamCleanup - cascadeDelete: %w", err)
	}

	if err := uc.adjustSolveCountsForChallenges(ctx, challengeIDs); err != nil {
		return fmt.Errorf("TeamUseCase - handleSoloTeamCleanup - adjustSolveCountsForChallenges: %w", err)
	}

	if err := uc.deps.UserRepo.UpdateTeamID(ctx, actorID, nil); err != nil {
		return fmt.Errorf("TeamUseCase - handleSoloTeamCleanup - UserRepo.UpdateTeamID: %w", err)
	}

	auditLog := &domain.TeamAuditLog{
		TeamID: oldTeamID,
		UserID: &actorID,
		Action: domain.TeamActionDeleted,
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

func (uc *TeamUseCase) shouldCleanupSoloTeam(user *domain.User, members []*domain.User, oldTeam *domain.Team) bool {
	return len(members) == 1 && members[0].ID == user.ID && (oldTeam.IsSolo || oldTeam.IsAutoCreated)
}
