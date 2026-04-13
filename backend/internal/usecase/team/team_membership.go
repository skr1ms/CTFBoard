package team

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/cacheutil"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/guard"
)

// Join validates the competition guard, delegates the full join flow to joinTx inside a
// transaction, then invalidates the user cache and full scoreboard cache on success.
func (uc *TeamUseCase) Join(ctx context.Context, inviteToken, userID uuid.UUID, confirmReset bool) (*domain.Team, error) {
	_, err := uc.deps.Guard.RequireTeamSwitch(ctx)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - Join - Guard.RequireTeamSwitch: %w", err)
	}

	var team *domain.Team

	err = uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		var errJoin error

		team, errJoin = uc.joinTx(ctx, inviteToken, userID, confirmReset)
		if errJoin != nil {
			return fmt.Errorf("TeamUseCase - Join - joinTx: %w", errJoin)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - Join - TM.Run: %w", err)
	}

	cacheutil.InvalidateUser(ctx, uc.deps.UserCache, userID)
	cacheutil.InvalidateScoreboardForTeam(ctx, uc.deps.ScoreboardCache, team.ID)

	return team, nil
}

// joinTx processes a join-via-invite-token request inside a transaction. It delegates
// initial validation and pre-lock reads to joinTxPrepare, then acquires advisory locks
// in a deadlock-safe order: when the user is already in a team, both the old and new
// team rows are locked using orderTeamLockIDs (lexicographic on UUID); otherwise only
// the target team is locked. After re-fetching the team under the lock it checks for a
// ban, verifies the team is not a solo team, confirms the invite token has not expired
// or been rotated, and enforces MaxTeamSize. If the user holds a solo or auto-created
// team it is cleaned up via handleSoloTeamCleanup (requires confirmReset=true).
func (uc *TeamUseCase) joinTx(ctx context.Context, inviteToken, userID uuid.UUID, confirmReset bool) (*domain.Team, error) {
	team, user, err := uc.joinTxPrepare(ctx, inviteToken, userID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - joinTx - joinTxPrepare: %w", err)
	}

	if user.TeamID != nil {
		firstID, secondID := orderTeamLockIDs(*user.TeamID, &team.ID)
		err := uc.deps.TeamRepo.Lock(ctx, firstID)
		if err != nil {
			return nil, fmt.Errorf("TeamUseCase - joinTx - TeamRepo.Lock(first): %w", err)
		}

		if secondID != uuid.Nil {
			err := uc.deps.TeamRepo.Lock(ctx, secondID)
			if err != nil {
				return nil, fmt.Errorf("TeamUseCase - joinTx - TeamRepo.Lock(second): %w", err)
			}
		}
	} else {
		err := uc.deps.TeamRepo.Lock(ctx, team.ID)
		if err != nil {
			return nil, fmt.Errorf("TeamUseCase - joinTx - TeamRepo.Lock: %w", err)
		}
	}

	team, err = uc.deps.TeamRepo.GetByID(ctx, team.ID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - joinTx - TeamRepo.GetByID: %w", err)
	}

	if team.IsBanned {
		return nil, apperr.ErrTeamBanned
	}

	if team.IsSolo {
		return nil, apperr.ErrTeamNotFound
	}

	if team.InviteTokenExpiresAt != nil && time.Now().After(*team.InviteTokenExpiresAt) {
		return nil, apperr.ErrInviteExpired
	}

	if team.InviteToken != inviteToken {
		return nil, apperr.ErrInviteExpired
	}

	comp, err := uc.deps.CompRepo.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - joinTx - CompetitionRepo.Get: %w", err)
	}

	maxSize := resolveMaxTeamSize(comp, uc.deps.DefaultMaxTeamSize)

	members, err := uc.deps.UserRepo.GetByTeamID(ctx, team.ID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - joinTx - UserRepo.GetByTeamID: %w", err)
	}

	if len(members) >= maxSize {
		return nil, apperr.ErrTeamFull
	}

	if user.TeamID != nil {
		err := uc.handleSoloTeamCleanup(ctx, user, userID, confirmReset, &team.ID)
		if err != nil {
			return nil, fmt.Errorf("TeamUseCase - joinTx - handleSoloTeamCleanup: %w", err)
		}
	}

	if err := uc.deps.UserRepo.UpdateTeamID(ctx, userID, &team.ID); err != nil {
		return nil, fmt.Errorf("TeamUseCase - joinTx - UserRepo.UpdateTeamID: %w", err)
	}

	auditLog := &domain.TeamAuditLog{TeamID: team.ID, UserID: &userID, Action: domain.TeamActionJoined}
	if err := uc.deps.TeamRepo.CreateAuditLog(ctx, auditLog); err != nil {
		return nil, fmt.Errorf("TeamUseCase - joinTx - TeamRepo.CreateAuditLog: %w", err)
	}

	return team, nil
}

// joinTxPrepare loads and validates the team and user before acquiring locks inside the join transaction.
// Checks invite token validity, competition mode, team capacity, and ban status.
func (uc *TeamUseCase) joinTxPrepare(ctx context.Context, inviteToken, userID uuid.UUID) (*domain.Team, *domain.User, error) {
	comp, err := uc.deps.CompRepo.Get(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("TeamUseCase - joinTx - CompetitionRepo.Get: %w", err)
	}

	if err := guard.ValidateTeamSwitchState(comp); err != nil {
		return nil, nil, err
	}

	if !comp.Mode.AllowsTeams() {
		return nil, nil, apperr.ErrTeamsNotAllowed
	}

	if err := uc.deps.UserRepo.Lock(ctx, userID); err != nil {
		return nil, nil, fmt.Errorf("TeamUseCase - joinTx - UserRepo.Lock: %w", err)
	}

	team, err := uc.deps.TeamRepo.GetByInviteToken(ctx, inviteToken)
	if err != nil {
		return nil, nil, fmt.Errorf("TeamUseCase - joinTx - TeamRepo.GetByInviteToken: %w", err)
	}

	if team.IsBanned {
		return nil, nil, apperr.ErrTeamBanned
	}

	if team.IsSolo {
		return nil, nil, apperr.ErrTeamNotFound
	}

	members, err := uc.deps.UserRepo.GetByTeamID(ctx, team.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("TeamUseCase - joinTx - UserRepo.GetByTeamID: %w", err)
	}

	maxSize := resolveMaxTeamSize(comp, uc.deps.DefaultMaxTeamSize)

	if len(members) >= maxSize {
		return nil, nil, apperr.ErrTeamFull
	}

	user, err := uc.deps.UserRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("TeamUseCase - joinTx - UserRepo.GetByID: %w", err)
	}

	if user.IsBanned {
		return nil, nil, apperr.ErrUserBanned
	}

	if user.WasInBannedTeam {
		return nil, nil, apperr.ErrUserWasInBannedTeam
	}

	return team, user, nil
}

// Leave validates the competition guard, runs leaveTx (prepare -> validate -> execute) in
// a transaction, then invalidates the user cache and full scoreboard cache.
func (uc *TeamUseCase) Leave(ctx context.Context, userID uuid.UUID) error {
	_, err := uc.deps.Guard.RequireTeamSwitch(ctx)
	if err != nil {
		return fmt.Errorf("TeamUseCase - Leave - Guard: %w", err)
	}

	var leftTeamID uuid.UUID

	if err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		var errLeave error

		leftTeamID, errLeave = uc.leaveTx(ctx, userID)
		if errLeave != nil {
			return fmt.Errorf("TeamUseCase - Leave - leaveTx: %w", errLeave)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("TeamUseCase - Leave - TM.Run: %w", err)
	}

	cacheutil.InvalidateUser(ctx, uc.deps.UserCache, userID)
	cacheutil.InvalidateScoreboardForTeam(ctx, uc.deps.ScoreboardCache, leftTeamID)

	return nil
}

// leaveTx orchestrates the leave flow inside a transaction:
// prepare (validate + lock) -> determine outcome (transfer captain or disband) -> execute.
func (uc *TeamUseCase) leaveTx(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	comp, err := uc.deps.CompRepo.Get(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("TeamUseCase - leaveTx - CompetitionRepo.Get: %w", err)
	}

	if err := guard.ValidateTeamSwitchState(comp); err != nil {
		return uuid.Nil, fmt.Errorf("TeamUseCase - leaveTx - ValidateTeamSwitchState: %w", err)
	}

	user, team, members, err := uc.leavePrepare(ctx, userID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("TeamUseCase - leaveTx - leavePrepare: %w", err)
	}

	if err := uc.leaveValidate(user, team, members, comp); err != nil {
		return uuid.Nil, fmt.Errorf("TeamUseCase - leaveTx - leaveValidate: %w", err)
	}

	return team.ID, uc.leaveExecute(ctx, userID, team)
}

// leavePrepare acquires advisory locks on the user row then the team row (in that order),
// fetches both entities and the current member list for use in leaveValidate.
func (uc *TeamUseCase) leavePrepare(ctx context.Context, userID uuid.UUID) (*domain.User, *domain.Team, []*domain.User, error) {
	if err := uc.deps.UserRepo.Lock(ctx, userID); err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - leavePrepare - UserRepo.Lock: %w", err)
	}

	user, err := uc.deps.UserRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - leavePrepare - UserRepo.GetByID: %w", err)
	}

	if user.TeamID == nil {
		return nil, nil, nil, apperr.ErrTeamNotFound
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

// leaveValidate checks all preconditions for a leave operation: user/team ban status,
// solo-only mode restriction, single-member team guard, captain restriction, and
// minimum team size constraint from competition settings.
func (uc *TeamUseCase) leaveValidate(user *domain.User, team *domain.Team, members []*domain.User, comp *domain.Competition) error {
	if user.IsBanned {
		return apperr.ErrUserBanned
	}

	if team.IsBanned {
		return apperr.ErrTeamBanned
	}

	if team.IsSolo && comp.Mode == domain.ModeSoloOnly {
		return apperr.ErrCannotLeaveSoloTeam
	}

	if len(members) == 1 {
		return apperr.ErrCannotLeaveAsOnlyMember
	}

	if team.CaptainID == user.ID {
		return apperr.ErrCaptainCannotLeave
	}

	minSize := comp.MinTeamSize
	if minSize > 0 && len(members)-1 < minSize {
		return apperr.ErrTeamBelowMinSize
	}

	return nil
}

// leaveExecute clears the user's team membership and writes a TeamActionLeft audit log
// entry. Cache invalidation is handled by the caller (Leave) after the transaction.
func (uc *TeamUseCase) leaveExecute(ctx context.Context, userID uuid.UUID, team *domain.Team) error {
	err := uc.deps.UserRepo.UpdateTeamID(ctx, userID, nil)
	if err != nil {
		return fmt.Errorf("TeamUseCase - leaveExecute - UserRepo.UpdateTeamID: %w", err)
	}

	auditLog := &domain.TeamAuditLog{TeamID: team.ID, UserID: &userID, Action: domain.TeamActionLeft}

	err = uc.deps.TeamRepo.CreateAuditLog(ctx, auditLog)
	if err != nil {
		return fmt.Errorf("TeamUseCase - leaveExecute - TeamRepo.CreateAuditLog: %w", err)
	}

	return nil
}

// TransferCaptain validates the guard, rejects a self-transfer immediately, then runs
// transferCaptainTx in a transaction. On success it invalidates the new captain's user
// cache, the team cache, and the full scoreboard cache.
func (uc *TeamUseCase) TransferCaptain(ctx context.Context, captainID, newCaptainID uuid.UUID) error {
	_, err := uc.deps.Guard.RequireTeamSwitch(ctx)
	if err != nil {
		return fmt.Errorf("TeamUseCase - TransferCaptain - Guard: %w", err)
	}

	if captainID == newCaptainID {
		return apperr.ErrCannotTransferToSelf
	}

	var teamID uuid.UUID

	if err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		var txErr error

		teamID, txErr = uc.transferCaptainTx(ctx, captainID, newCaptainID)

		return txErr
	}); err != nil {
		return fmt.Errorf("TeamUseCase - TransferCaptain - TM.Run: %w", err)
	}

	cacheutil.InvalidateUser(ctx, uc.deps.UserCache, captainID)
	cacheutil.InvalidateUser(ctx, uc.deps.UserCache, newCaptainID)
	cacheutil.InvalidateTeam(ctx, uc.deps.TeamCache, uc.deps.Logger, teamID)
	cacheutil.InvalidateScoreboardForTeam(ctx, uc.deps.ScoreboardCache, teamID)

	return nil
}

// transferCaptainTx executes the captaincy transfer inside a transaction after acquiring per-user locks.
func (uc *TeamUseCase) transferCaptainTx(ctx context.Context, captainID, newCaptainID uuid.UUID) (uuid.UUID, error) {
	comp, err := uc.deps.CompRepo.Get(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("TeamUseCase - transferCaptainTx - CompetitionRepo.Get: %w", err)
	}

	if err := guard.ValidateTeamSwitchState(comp); err != nil {
		return uuid.Nil, fmt.Errorf("TeamUseCase - transferCaptainTx - ValidateTeamSwitchState: %w", err)
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

// transferCaptainPrepare acquires per-user locks in lexicographic UUID order to prevent
// deadlocks, then locks the team row and loads all three entities required for the transfer.
func (uc *TeamUseCase) transferCaptainPrepare(ctx context.Context, captainID, newCaptainID uuid.UUID) (*domain.User, *domain.Team, *domain.User, error) {
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
		return nil, nil, nil, apperr.ErrTeamNotFound
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

func (uc *TeamUseCase) transferCaptainValidate(captain *domain.User, team *domain.Team, newCaptain *domain.User, captainID uuid.UUID) error {
	if team.CaptainID != captainID {
		return apperr.ErrNotCaptain
	}

	if captain.IsBanned {
		return apperr.ErrUserBanned
	}

	if team.IsBanned {
		return apperr.ErrTeamBanned
	}

	if newCaptain.TeamID == nil || *newCaptain.TeamID != team.ID {
		return apperr.ErrNewCaptainNotInTeam
	}

	if newCaptain.IsBanned {
		return apperr.ErrUserBanned
	}

	return nil
}

func (uc *TeamUseCase) transferCaptainExecute(ctx context.Context, captainID, newCaptainID uuid.UUID, team *domain.Team) error {
	err := uc.deps.TeamRepo.UpdateCaptain(ctx, team.ID, newCaptainID)
	if err != nil {
		return fmt.Errorf("TeamUseCase - transferCaptainExecute - TeamRepo.UpdateCaptain: %w", err)
	}

	auditLog := &domain.TeamAuditLog{
		TeamID: team.ID, UserID: &captainID, Action: domain.TeamActionCaptainTransfer,
		Details: map[string]any{"from": captainID.String(), "to": newCaptainID.String()},
	}

	err = uc.deps.TeamRepo.CreateAuditLog(ctx, auditLog)
	if err != nil {
		return fmt.Errorf("TeamUseCase - transferCaptainExecute - TeamRepo.CreateAuditLog: %w", err)
	}

	return nil
}

// DisbandTeam dissolves a team initiated by its captain. It checks competition-level
// team-switch guard, validates competition state restrictions, then runs a transaction
// that cascades deletion of solves, submissions, awards, and hint unlocks, recalculates
// dynamic scoring, and clears team membership for all members.
func (uc *TeamUseCase) DisbandTeam(ctx context.Context, captainID uuid.UUID) error {
	if _, err := uc.deps.Guard.RequireTeamSwitch(ctx); err != nil {
		return fmt.Errorf("TeamUseCase - DisbandTeam - Guard: %w", err)
	}

	var memberIDs []uuid.UUID

	err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		comp, err := uc.deps.CompRepo.Get(ctx)
		if err != nil {
			return fmt.Errorf("TeamUseCase - DisbandTeam - CompetitionRepo.Get: %w", err)
		}

		if err := guard.ValidateTeamSwitchState(comp); err != nil {
			return fmt.Errorf("TeamUseCase - DisbandTeam - ValidateTeamSwitchState: %w", err)
		}

		user, team, members, err := uc.disbandPrepare(ctx, captainID)
		if err != nil {
			return fmt.Errorf("TeamUseCase - DisbandTeam - disbandPrepare: %w", err)
		}

		if err := uc.disbandValidate(user, team, captainID, comp); err != nil {
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
		cacheutil.InvalidateUser(ctx, uc.deps.UserCache, id)
	}

	cacheutil.InvalidateScoreboard(ctx, uc.deps.ScoreboardCache)
	cacheutil.InvalidateChallengeList(ctx, uc.deps.ChallengeListCache)

	return nil
}

func (uc *TeamUseCase) disbandPrepare(ctx context.Context, captainID uuid.UUID) (*domain.User, *domain.Team, []*domain.User, error) {
	if err := uc.deps.UserRepo.Lock(ctx, captainID); err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - disbandPrepare - UserRepo.Lock: %w", err)
	}

	user, err := uc.deps.UserRepo.GetByID(ctx, captainID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - disbandPrepare - UserRepo.GetByID: %w", err)
	}

	if user.TeamID == nil {
		return nil, nil, nil, apperr.ErrTeamNotFound
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

func (uc *TeamUseCase) disbandValidate(user *domain.User, team *domain.Team, captainID uuid.UUID, comp *domain.Competition) error {
	if team.CaptainID != captainID {
		return apperr.ErrNotCaptain
	}

	if team.IsSolo && comp.Mode == domain.ModeSoloOnly {
		return apperr.ErrCannotDisbandSoloTeam
	}

	if user.IsBanned {
		return apperr.ErrUserBanned
	}

	if team.IsBanned {
		return apperr.ErrTeamBanned
	}

	return nil
}

// disbandExecute cascades a team deletion inside a transaction. It collects the
// challenge IDs affected by the team's solves so that solve counts and dynamic scoring
// can be recalculated after deletion. It then hard-deletes all solves, submissions,
// awards, and hint unlocks, calls adjustSolveCountsForChallenges, writes a disbanded
// audit log entry, clears the team_id field for all members in a single batch update,
// removes any custom field values, and finally soft-deletes the team record. Solo teams
// for the former members are not created here; callers are responsible for that if
// required by the competition mode.
func (uc *TeamUseCase) disbandExecute(ctx context.Context, team *domain.Team, members []*domain.User, captainID uuid.UUID) error {
	challengeIDs, err := uc.getChallengeIDsForTeam(ctx, team.ID)
	if err != nil {
		return fmt.Errorf("TeamUseCase - disbandExecute - getChallengeIDsForTeam: %w", err)
	}

	if err := uc.cascadeDelete(ctx, team.ID); err != nil {
		return fmt.Errorf("TeamUseCase - disbandExecute - cascadeDelete: %w", err)
	}

	if err := uc.adjustSolveCountsForChallenges(ctx, challengeIDs); err != nil {
		return fmt.Errorf("TeamUseCase - disbandExecute - adjustSolveCountsForChallenges: %w", err)
	}

	auditLog := &domain.TeamAuditLog{
		TeamID: team.ID, UserID: &captainID, Action: domain.TeamActionDeleted,
		Details: map[string]any{"reason": "disbanded_by_captain"},
	}
	if err := uc.deps.TeamRepo.CreateAuditLog(ctx, auditLog); err != nil {
		return fmt.Errorf("TeamUseCase - disbandExecute - TeamRepo.CreateAuditLog: %w", err)
	}

	memberIDs := make([]uuid.UUID, len(members))
	for i, m := range members {
		memberIDs[i] = m.ID
	}

	if err := uc.deps.UserRepo.UpdateTeamIDBatch(ctx, memberIDs, nil); err != nil {
		return fmt.Errorf("TeamUseCase - disbandExecute - UserRepo.UpdateTeamIDBatch: %w", err)
	}

	if uc.deps.FieldValueRepo != nil {
		err := uc.deps.FieldValueRepo.DeleteByEntityID(ctx, team.ID)
		if err != nil {
			return fmt.Errorf("TeamUseCase - disbandExecute - FieldValueRepo.DeleteByEntityID: %w", err)
		}
	}

	if err := uc.deps.TeamRepo.Delete(ctx, team.ID); err != nil {
		return fmt.Errorf("TeamUseCase - disbandExecute - TeamRepo.Delete: %w", err)
	}

	return nil
}

// KickMember validates the guard, rejects a self-kick immediately, then runs
// kickMemberTx in a transaction. On success it invalidates the target user's cache
// and the full scoreboard cache.
func (uc *TeamUseCase) KickMember(ctx context.Context, captainID, targetUserID uuid.UUID) error {
	_, err := uc.deps.Guard.RequireTeamSwitch(ctx)
	if err != nil {
		return fmt.Errorf("TeamUseCase - KickMember - Guard: %w", err)
	}

	if captainID == targetUserID {
		return apperr.ErrCannotKickSelf
	}

	var kickedFromTeamID uuid.UUID

	if err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		var errKick error

		kickedFromTeamID, errKick = uc.kickMemberTx(ctx, captainID, targetUserID)

		return errKick
	}); err != nil {
		return fmt.Errorf("TeamUseCase - KickMember - TM.Run: %w", err)
	}

	cacheutil.InvalidateUser(ctx, uc.deps.UserCache, targetUserID)
	cacheutil.InvalidateScoreboardForTeam(ctx, uc.deps.ScoreboardCache, kickedFromTeamID)

	return nil
}

// kickMemberTx runs the kick operation inside a transaction: checks competition guard,
// acquires locks via kickMemberPrepare, validates constraints, then executes the kick.
// Returns the team ID so the caller can perform targeted cache invalidation.
func (uc *TeamUseCase) kickMemberTx(ctx context.Context, captainID, targetUserID uuid.UUID) (uuid.UUID, error) {
	comp, err := uc.deps.CompRepo.Get(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("TeamUseCase - kickMemberTx - CompRepo.Get: %w", err)
	}

	if err := guard.ValidateTeamSwitchState(comp); err != nil {
		return uuid.Nil, fmt.Errorf("TeamUseCase - kickMemberTx - ValidateTeamSwitchState: %w", err)
	}

	captain, team, targetUser, err := uc.kickMemberPrepare(ctx, captainID, targetUserID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("TeamUseCase - kickMemberTx - kickMemberPrepare: %w", err)
	}

	members, err := uc.deps.UserRepo.GetByTeamID(ctx, team.ID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("TeamUseCase - kickMemberTx - UserRepo.GetByTeamID: %w", err)
	}

	if err := uc.kickMemberValidate(captain, team, targetUser, captainID, targetUserID, len(members), comp.MinTeamSize); err != nil {
		return uuid.Nil, err
	}

	return team.ID, uc.kickMemberExecute(ctx, team.ID, captainID, targetUserID)
}

// kickMemberPrepare acquires per-user locks in lexicographic UUID order (captain and
// target), then locks the team row, and loads all three entities needed for validation.
func (uc *TeamUseCase) kickMemberPrepare(ctx context.Context, captainID, targetUserID uuid.UUID) (*domain.User, *domain.Team, *domain.User, error) {
	firstID, secondID := captainID, targetUserID
	if captainID.String() > targetUserID.String() {
		firstID, secondID = targetUserID, captainID
	}

	if err := uc.deps.UserRepo.Lock(ctx, firstID); err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - kickMemberPrepare - UserRepo.Lock (first): %w", err)
	}
	if err := uc.deps.UserRepo.Lock(ctx, secondID); err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - kickMemberPrepare - UserRepo.Lock (second): %w", err)
	}

	captain, err := uc.deps.UserRepo.GetByID(ctx, captainID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - kickMemberPrepare - UserRepo.GetByID (captain): %w", err)
	}

	if captain.TeamID == nil {
		return nil, nil, nil, apperr.ErrTeamNotFound
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
		return nil, nil, nil, fmt.Errorf("TeamUseCase - kickMemberPrepare - UserRepo.GetByID (target): %w", err)
	}

	return captain, team, targetUser, nil
}

// kickMemberValidate enforces kick preconditions: captain ban, captaincy ownership,
// anti-self-kick guard on captain row, team ban, target membership in team, and
// minimum team size constraint.
func (uc *TeamUseCase) kickMemberValidate(captain *domain.User, team *domain.Team, targetUser *domain.User, captainID, _ uuid.UUID, memberCount, minTeamSize int) error {
	if captain.IsBanned {
		return apperr.ErrUserBanned
	}

	if team.CaptainID != captainID {
		return apperr.ErrNotCaptain
	}

	if targetUser.ID == team.CaptainID {
		return apperr.ErrCannotKickCaptain
	}

	if team.IsBanned {
		return apperr.ErrTeamBanned
	}

	if targetUser.TeamID == nil || *targetUser.TeamID != team.ID {
		return apperr.ErrUserNotFound
	}

	if minTeamSize > 0 && memberCount-1 < minTeamSize {
		return apperr.ErrTeamBelowMinSize
	}

	return nil
}

// kickMemberExecute clears the target's team membership and writes a TeamActionMemberKicked
// audit log with the target_user_id recorded in Details for traceability.
func (uc *TeamUseCase) kickMemberExecute(ctx context.Context, teamID, captainID, targetUserID uuid.UUID) error {
	err := uc.deps.UserRepo.UpdateTeamID(ctx, targetUserID, nil)
	if err != nil {
		return fmt.Errorf("TeamUseCase - kickMemberExecute - UserRepo.UpdateTeamID: %w", err)
	}

	auditLog := &domain.TeamAuditLog{
		TeamID: teamID, UserID: &captainID, Action: domain.TeamActionMemberKicked,
		Details: map[string]any{"target_user_id": targetUserID.String()},
	}

	err = uc.deps.TeamRepo.CreateAuditLog(ctx, auditLog)
	if err != nil {
		return fmt.Errorf("TeamUseCase - kickMemberExecute - TeamRepo.CreateAuditLog: %w", err)
	}

	return nil
}
