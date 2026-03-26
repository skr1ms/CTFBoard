package team

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/scoring"
)

func (uc *TeamUseCase) Create(ctx context.Context, name string, captainID uuid.UUID, isSolo, confirmReset bool) (*domain.Team, error) {
	var team *domain.Team

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

	const maxTeamsLockKey int64 = 0x4354467465616D73
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

func (uc *TeamUseCase) createTx(ctx context.Context, name string, captainID uuid.UUID, isSolo, confirmReset bool) (*domain.Team, error) {
	comp, err := uc.deps.CompRepo.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - createTx - CompetitionRepo.Get: %w", err)
	}

	if err := uc.requireTeamSwitchAndTeamsMode(comp); err != nil {
		return nil, err
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

	if user.WasInBannedTeam {
		return nil, httperr.ErrUserWasInBannedTeam
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

func (uc *TeamUseCase) requireTeamSwitch(comp *domain.Competition) error {
	switch comp.GetStatus() {
	case domain.CompetitionStatusNotStarted, domain.CompetitionStatusActive, domain.CompetitionStatusFrozen:
	case domain.CompetitionStatusEnded:
		return httperr.ErrCompetitionEnded
	case domain.CompetitionStatusPaused:
		return httperr.ErrCompetitionPaused
	}

	if !comp.AllowTeamSwitch {
		return httperr.ErrRosterFrozen
	}

	return nil
}

func (uc *TeamUseCase) requireTeamSwitchAndTeamsMode(comp *domain.Competition) error {
	err := uc.requireTeamSwitch(comp)
	if err != nil {
		return err
	}

	if !comp.Mode.AllowsTeams() {
		return httperr.ErrTeamsNotAllowed
	}

	return nil
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

func (uc *TeamUseCase) CreateSoloTeam(ctx context.Context, userID uuid.UUID, confirmReset bool) (*domain.Team, error) {
	_, err := uc.deps.Guard.RequireTeamSwitchAndSoloMode(ctx)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - CreateSoloTeam - Guard: %w", err)
	}

	var team *domain.Team

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

func (uc *TeamUseCase) CreateSoloTeamForNewUser(ctx context.Context, userID uuid.UUID) (*domain.Team, error) {
	if err := uc.requireSoloModeOnly(ctx); err != nil {
		return nil, fmt.Errorf("TeamUseCase - CreateSoloTeamForNewUser - requireSoloModeOnly: %w", err)
	}

	var team *domain.Team

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
		return nil, httperr.ErrUserBanned
	}

	if user.WasInBannedTeam {
		return nil, httperr.ErrUserWasInBannedTeam
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

		var pgErr *pgconn.PgError
		if (!errors.As(err, &pgErr) || pgErr.Code != "23505") || attempt == maxSoloNameRetries-1 {
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

func (uc *TeamUseCase) getChallengeIDsForTeam(ctx context.Context, teamID uuid.UUID) ([]uuid.UUID, error) {
	if uc.deps.SolveRepo == nil {
		return nil, nil
	}

	solves, err := uc.deps.SolveRepo.GetByTeamIDWithDetails(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - getChallengeIDsForTeam - SolveRepo.GetByTeamIDWithDetails: %w", err)
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

	return challengeIDs, nil
}

func (uc *TeamUseCase) adjustSolveCountsForChallenges(ctx context.Context, challengeIDs []uuid.UUID, decrement bool) error {
	if uc.deps.ChallengeRepo == nil || len(challengeIDs) == 0 {
		return nil
	}

	if decrement {
		err := uc.deps.ChallengeRepo.BatchDecrementSolveCount(ctx, challengeIDs)
		if err != nil {
			return fmt.Errorf("TeamUseCase - adjustSolveCountsForChallenges - BatchDecrementSolveCount: %w", err)
		}
	} else {
		err := uc.deps.ChallengeRepo.BatchIncrementSolveCount(ctx, challengeIDs)
		if err != nil {
			return fmt.Errorf("TeamUseCase - adjustSolveCountsForChallenges - BatchIncrementSolveCount: %w", err)
		}
	}

	challengesMap, err := uc.deps.ChallengeRepo.GetByIDs(ctx, challengeIDs)
	if err != nil {
		return fmt.Errorf("TeamUseCase - adjustSolveCountsForChallenges - ChallengeRepo.GetByIDs: %w", err)
	}

	ids, points := scoring.RecalculatePoints(challengesMap)
	if len(ids) > 0 {
		err := uc.deps.ChallengeRepo.BatchUpdatePoints(ctx, ids, points)
		if err != nil {
			return fmt.Errorf("TeamUseCase - adjustSolveCountsForChallenges - BatchUpdatePoints: %w", err)
		}
	}

	if uc.deps.SolveRepo != nil {
		var dynamicIDs []uuid.UUID

		for _, id := range challengeIDs {
			if c := challengesMap[id]; c != nil && c.InitialValue > 0 && c.Decay > 0 {
				dynamicIDs = append(dynamicIDs, id)
			}
		}

		if len(dynamicIDs) > 0 {
			rows, err := uc.deps.SolveRepo.GetSolvesForPointsRecalc(ctx, dynamicIDs)
			if err != nil {
				return fmt.Errorf("TeamUseCase - adjustSolveCountsForChallenges - GetSolvesForPointsRecalc: %w", err)
			}

			recalcRows := make([]*scoring.SolveRowForPointsRecalc, 0, len(rows))
			for _, r := range rows {
				recalcRows = append(recalcRows, &scoring.SolveRowForPointsRecalc{
					ID: r.ID, ChallengeID: r.ChallengeID,
					InitialValue: r.InitialValue, MinValue: r.MinValue, Decay: r.Decay,
				})
			}

			solveIDs, newPoints := scoring.RecalculatePointsAtSolveRows(recalcRows)
			if len(solveIDs) > 0 {
				err := uc.deps.SolveRepo.BatchUpdateSolvePoints(ctx, solveIDs, newPoints)
				if err != nil {
					return fmt.Errorf("TeamUseCase - adjustSolveCountsForChallenges - BatchUpdateSolvePoints: %w", err)
				}
			}
		}
	}

	return nil
}

func (uc *TeamUseCase) adjustSolveCountsForTeam(ctx context.Context, teamID uuid.UUID, decrement bool) error {
	challengeIDs, err := uc.getChallengeIDsForTeam(ctx, teamID)
	if err != nil {
		return err
	}

	return uc.adjustSolveCountsForChallenges(ctx, challengeIDs, decrement)
}

func orderTeamLockIDs(oldTeamID uuid.UUID, newTeamID *uuid.UUID) (first, second uuid.UUID) {
	if newTeamID == nil {
		return oldTeamID, uuid.Nil
	}

	if oldTeamID.String() < newTeamID.String() {
		return oldTeamID, *newTeamID
	}

	return *newTeamID, oldTeamID
}

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
			return httperr.ErrTeamBanned
		}

		return httperr.ErrUserAlreadyInTeam
	}

	if oldTeam.IsBanned {
		return httperr.ErrTeamBanned
	}

	if !confirmReset {
		return httperr.ErrConfirmationRequired
	}

	oldTeamID := *user.TeamID

	challengeIDs, err := uc.getChallengeIDsForTeam(ctx, oldTeamID)
	if err != nil {
		return fmt.Errorf("TeamUseCase - handleSoloTeamCleanup - getChallengeIDsForTeam: %w", err)
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

	if uc.deps.HintRepo != nil {
		err := uc.deps.HintRepo.DeleteUnlocksByTeamID(ctx, oldTeamID)
		if err != nil {
			return fmt.Errorf("TeamUseCase - handleSoloTeamCleanup - HintRepo.DeleteUnlocksByTeamID: %w", err)
		}
	}

	if err := uc.adjustSolveCountsForChallenges(ctx, challengeIDs, true); err != nil {
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
