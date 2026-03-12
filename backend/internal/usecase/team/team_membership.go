package team

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

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
		firstID, secondID := orderTeamLockIDs(*user.TeamID, &team.ID)
		if err := uc.deps.TeamRepo.Lock(ctx, firstID); err != nil {
			return nil, fmt.Errorf("TeamUseCase - joinTx - TeamRepo.Lock(first): %w", err)
		}
		if secondID != uuid.Nil {
			if err := uc.deps.TeamRepo.Lock(ctx, secondID); err != nil {
				return nil, fmt.Errorf("TeamUseCase - joinTx - TeamRepo.Lock(second): %w", err)
			}
		}
	} else {
		if err := uc.deps.TeamRepo.Lock(ctx, team.ID); err != nil {
			return nil, fmt.Errorf("TeamUseCase - joinTx - TeamRepo.Lock: %w", err)
		}
	}
	team, err = uc.deps.TeamRepo.GetByID(ctx, team.ID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - joinTx - TeamRepo.GetByID: %w", err)
	}
	if team.IsBanned {
		return nil, httperr.ErrTeamBanned
	}
	if team.IsSolo {
		return nil, httperr.ErrTeamNotFound
	}
	if team.InviteTokenExpiresAt != nil && time.Now().After(*team.InviteTokenExpiresAt) {
		return nil, httperr.ErrInviteExpired
	}
	if team.InviteToken != inviteToken {
		return nil, httperr.ErrInviteExpired
	}
	comp, err := uc.deps.CompRepo.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - joinTx - CompetitionRepo.Get: %w", err)
	}
	maxSize := comp.MaxTeamSize
	if maxSize <= 0 {
		maxSize = uc.deps.DefaultMaxTeamSize
	}
	members, err := uc.deps.UserRepo.GetByTeamID(ctx, team.ID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - joinTx - UserRepo.GetByTeamID: %w", err)
	}
	if len(members) >= maxSize {
		return nil, httperr.ErrTeamFull
	}
	if user.TeamID != nil {
		if err := uc.handleSoloTeamCleanup(ctx, user, userID, confirmReset, &team.ID); err != nil {
			return nil, fmt.Errorf("TeamUseCase - joinTx - handleSoloTeamCleanup: %w", err)
		}
	}
	if err := uc.deps.UserRepo.UpdateTeamID(ctx, userID, &team.ID); err != nil {
		return nil, fmt.Errorf("TeamUseCase - joinTx - UserRepo.UpdateTeamID: %w", err)
	}
	auditLog := &entity.TeamAuditLog{TeamID: team.ID, UserID: &userID, Action: entity.TeamActionJoined}
	if err := uc.deps.TeamRepo.CreateAuditLog(ctx, auditLog); err != nil {
		return nil, fmt.Errorf("TeamUseCase - joinTx - TeamRepo.CreateAuditLog: %w", err)
	}
	return team, nil
}

func (uc *TeamUseCase) joinTxPrepare(ctx context.Context, inviteToken, userID uuid.UUID) (*entity.Team, *entity.User, error) {
	comp, err := uc.deps.CompRepo.Get(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("TeamUseCase - joinTx - CompetitionRepo.Get: %w", err)
	}
	if err := uc.requireTeamSwitch(comp); err != nil {
		return nil, nil, err
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
	if user.WasInBannedTeam {
		return nil, nil, httperr.ErrUserWasInBannedTeam
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
	comp, err := uc.deps.CompRepo.Get(ctx)
	if err != nil {
		return fmt.Errorf("TeamUseCase - leaveTx - CompetitionRepo.Get: %w", err)
	}
	if err := uc.requireTeamSwitch(comp); err != nil {
		return fmt.Errorf("TeamUseCase - leaveTx - requireTeamSwitch: %w", err)
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
	if user.IsBanned {
		return httperr.ErrUserBanned
	}
	if team.IsSolo && comp.Mode == entity.ModeSoloOnly {
		return httperr.ErrCannotLeaveSoloTeam
	}
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
	auditLog := &entity.TeamAuditLog{TeamID: team.ID, UserID: &userID, Action: entity.TeamActionLeft}
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
	comp, err := uc.deps.CompRepo.Get(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("TeamUseCase - transferCaptainTx - CompetitionRepo.Get: %w", err)
	}
	if err := uc.requireTeamSwitch(comp); err != nil {
		return uuid.Nil, fmt.Errorf("TeamUseCase - transferCaptainTx - requireTeamSwitch: %w", err)
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
		TeamID: team.ID, UserID: &captainID, Action: entity.TeamActionCaptainTransfer,
		Details: map[string]any{"from": captainID.String(), "to": newCaptainID.String()},
	}
	if err := uc.deps.TeamRepo.CreateAuditLog(ctx, auditLog); err != nil {
		return fmt.Errorf("TeamUseCase - transferCaptainExecute - TeamRepo.CreateAuditLog: %w", err)
	}
	return nil
}

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
		if err := uc.requireTeamSwitch(comp); err != nil {
			return fmt.Errorf("TeamUseCase - DisbandTeam - requireTeamSwitch: %w", err)
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
		uc.invalidateUserCache(ctx, id)
	}
	uc.invalidateScoreboardCache(ctx)
	uc.invalidateChallengeListCache(ctx)
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

func (uc *TeamUseCase) disbandValidate(user *entity.User, team *entity.Team, captainID uuid.UUID, comp *entity.Competition) error {
	if team.CaptainID != captainID {
		return httperr.ErrNotCaptain
	}
	if team.IsSolo && comp.Mode == entity.ModeSoloOnly {
		return httperr.ErrCannotDisbandSoloTeam
	}
	if user.IsBanned {
		return httperr.ErrUserBanned
	}
	if team.IsBanned {
		return httperr.ErrTeamBanned
	}
	return nil
}

func (uc *TeamUseCase) disbandExecute(ctx context.Context, team *entity.Team, members []*entity.User, captainID uuid.UUID) error {
	challengeIDs, err := uc.getChallengeIDsForTeam(ctx, team.ID)
	if err != nil {
		return fmt.Errorf("TeamUseCase - disbandExecute - getChallengeIDsForTeam: %w", err)
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
	if uc.deps.HintRepo != nil {
		if err := uc.deps.HintRepo.DeleteUnlocksByTeamID(ctx, team.ID); err != nil {
			return fmt.Errorf("TeamUseCase - disbandExecute - HintRepo.DeleteUnlocksByTeamID: %w", err)
		}
	}
	if err := uc.adjustSolveCountsForChallenges(ctx, challengeIDs, true); err != nil {
		return fmt.Errorf("TeamUseCase - disbandExecute - adjustSolveCountsForChallenges: %w", err)
	}
	auditLog := &entity.TeamAuditLog{
		TeamID: team.ID, UserID: &captainID, Action: entity.TeamActionDeleted,
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
		if err := uc.deps.FieldValueRepo.DeleteByEntityID(ctx, team.ID); err != nil {
			return fmt.Errorf("TeamUseCase - disbandExecute - FieldValueRepo.DeleteByEntityID: %w", err)
		}
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
	comp, err := uc.deps.CompRepo.Get(ctx)
	if err != nil {
		return fmt.Errorf("TeamUseCase - KickMember - kickMemberTx CompRepo.Get: %w", err)
	}
	if err := uc.requireTeamSwitch(comp); err != nil {
		return fmt.Errorf("TeamUseCase - KickMember - kickMemberTx requireTeamSwitch: %w", err)
	}
	captain, team, targetUser, err := uc.kickMemberPrepare(ctx, captainID, targetUserID)
	if err != nil {
		return err
	}
	members, err := uc.deps.UserRepo.GetByTeamID(ctx, team.ID)
	if err != nil {
		return fmt.Errorf("TeamUseCase - KickMember - kickMemberTx UserRepo.GetByTeamID: %w", err)
	}
	if err := uc.kickMemberValidate(captain, team, targetUser, captainID, targetUserID, len(members), comp.MinTeamSize); err != nil {
		return err
	}
	return uc.kickMemberExecute(ctx, team.ID, captainID, targetUserID)
}

func (uc *TeamUseCase) kickMemberPrepare(ctx context.Context, captainID, targetUserID uuid.UUID) (*entity.User, *entity.Team, *entity.User, error) {
	firstID, secondID := captainID, targetUserID
	if captainID.String() > targetUserID.String() {
		firstID, secondID = targetUserID, captainID
	}
	if err := uc.deps.UserRepo.Lock(ctx, firstID); err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - KickMember - kickMemberPrepare UserRepo.Lock: %w", err)
	}
	if err := uc.deps.UserRepo.Lock(ctx, secondID); err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - KickMember - kickMemberPrepare UserRepo.Lock: %w", err)
	}
	captain, err := uc.deps.UserRepo.GetByID(ctx, captainID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - KickMember - kickMemberPrepare UserRepo.GetByID: %w", err)
	}
	if captain.TeamID == nil {
		return nil, nil, nil, httperr.ErrTeamNotFound
	}
	if err := uc.deps.TeamRepo.Lock(ctx, *captain.TeamID); err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - KickMember - kickMemberPrepare TeamRepo.Lock: %w", err)
	}
	team, err := uc.deps.TeamRepo.GetByID(ctx, *captain.TeamID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - KickMember - kickMemberPrepare TeamRepo.GetByID: %w", err)
	}
	targetUser, err := uc.deps.UserRepo.GetByID(ctx, targetUserID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - KickMember - kickMemberPrepare UserRepo.GetByID: %w", err)
	}
	return captain, team, targetUser, nil
}

func (uc *TeamUseCase) kickMemberValidate(captain *entity.User, team *entity.Team, targetUser *entity.User, captainID, _ uuid.UUID, memberCount, minTeamSize int) error {
	if captain.IsBanned {
		return httperr.ErrUserBanned
	}
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
		TeamID: teamID, UserID: &captainID, Action: entity.TeamActionMemberKicked,
		Details: map[string]any{"target_user_id": targetUserID.String()},
	}
	if err := uc.deps.TeamRepo.CreateAuditLog(ctx, auditLog); err != nil {
		return fmt.Errorf("TeamUseCase - kickMemberExecute - TeamRepo.CreateAuditLog: %w", err)
	}
	return nil
}
