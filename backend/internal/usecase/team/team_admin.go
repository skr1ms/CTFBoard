package team

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/cacheutil"
)

func (uc *TeamUseCase) AdminListTeams(ctx context.Context, search *string, banStatus usecase.AdminTeamBanStatus, visibility usecase.AdminTeamVisibility, page, perPage int) (*usecase.Paginated[*domain.Team], error) {
	var result *usecase.Paginated[*domain.Team]

	filter := repo.TeamAdminSearchFilter{
		Search:     search,
		BanStatus:  toRepoTeamBanStatus(banStatus),
		Visibility: toRepoTeamVisibility(visibility),
	}

	err := uc.deps.TM.ReadOnly(ctx, func(roCtx context.Context) error {
		var err error

		result, err = usecase.FetchPage(roCtx, page, perPage,
			func(ctx context.Context, limit, offset int) ([]*domain.Team, error) {
				return uc.deps.TeamRepo.SearchAdmin(ctx, filter, limit, offset)
			},
			func(ctx context.Context) (int64, error) {
				return uc.deps.TeamRepo.CountSearchAdmin(ctx, filter)
			},
		)
		if err != nil {
			return fmt.Errorf("TeamUseCase - AdminListTeams: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - AdminListTeams: %w", err)
	}

	return result, nil
}

func toRepoTeamBanStatus(status usecase.AdminTeamBanStatus) repo.TeamAdminBanStatus {
	switch status {
	case usecase.AdminTeamBanStatusAll:
		return repo.TeamAdminBanStatusAll
	case usecase.AdminTeamBanStatusNotBanned:
		return repo.TeamAdminBanStatusNotBanned
	case usecase.AdminTeamBanStatusBanned:
		return repo.TeamAdminBanStatusBanned
	default:
		return repo.TeamAdminBanStatusAll
	}
}

func toRepoTeamVisibility(visibility usecase.AdminTeamVisibility) repo.TeamAdminVisibility {
	switch visibility {
	case usecase.AdminTeamVisibilityAll:
		return repo.TeamAdminVisibilityAll
	case usecase.AdminTeamVisibilityVisible:
		return repo.TeamAdminVisibilityVisible
	case usecase.AdminTeamVisibilityHidden:
		return repo.TeamAdminVisibilityHidden
	default:
		return repo.TeamAdminVisibilityAll
	}
}

// AdminUpdate modifies team metadata without competition-state restrictions. When a new
// captain is requested the candidate is locked and validated (must be in the team and
// not banned) before the team row is locked, preventing captain/team lock inversion.
func (uc *TeamUseCase) AdminUpdate(ctx context.Context, teamID uuid.UUID, name *string, captainID, bracketID *uuid.UUID, isHidden *bool) (*domain.Team, error) {
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
				return apperr.ErrNewCaptainNotInTeam
			}

			if candidate.IsBanned {
				return apperr.ErrUserBanned
			}
		}

		if err := uc.deps.TeamRepo.Lock(ctx, teamID); err != nil {
			return fmt.Errorf("TeamUseCase - AdminUpdate - TeamRepo.Lock: %w", err)
		}

		currentTeam, err := uc.deps.TeamRepo.GetByID(ctx, teamID)
		if err != nil {
			return fmt.Errorf("TeamUseCase - AdminUpdate - TeamRepo.GetByID: %w", err)
		}

		if name != nil && currentTeam.Name != *name {
			err := uc.validateTeamNameAvailable(ctx, *name)
			if err != nil {
				return fmt.Errorf("TeamUseCase - AdminUpdate - validateTeamNameAvailable: %w", err)
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

	cacheutil.InvalidateScoreboardForTeam(ctx, uc.deps.ScoreboardCache, teamID)

	return team, nil
}

// AdminDelete force-deletes a team without competition-state restrictions. To guard
// against race conditions the function reads the member list, sorts it, locks each
// member row in lexicographic UUID order, then locks the team row, and reads the
// member list a second time. If the two snapshots differ in length or membership the
// transaction returns ErrTeamConflict and the caller retries. Once a consistent
// snapshot is confirmed, solve counts are recalculated before the cascade deletion so
// that challenge point totals are correct even if the transaction rolls back partway
// Solves, submissions, awards, and hint unlocks are then hard-deleted, all member
// team_id fields are cleared in a batch, the team record is deleted, and an audit log
// entry is written.
func (uc *TeamUseCase) AdminDelete(ctx context.Context, teamID uuid.UUID) error {
	var memberIDs []uuid.UUID

	err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		members, err := uc.lockTeamWithMembers(ctx, teamID)
		if err != nil {
			return fmt.Errorf("TeamUseCase - AdminDelete - lockTeamWithMembers: %w", err)
		}

		memberIDs = make([]uuid.UUID, len(members))
		for i, m := range members {
			memberIDs[i] = m.ID
		}

		challengeIDs, err := uc.getChallengeIDsForTeam(ctx, teamID)
		if err != nil {
			return fmt.Errorf("TeamUseCase - AdminDelete - getChallengeIDsForTeam: %w", err)
		}

		if err := uc.cascadeDelete(ctx, teamID); err != nil {
			return fmt.Errorf("TeamUseCase - AdminDelete - cascadeDelete: %w", err)
		}

		if err := uc.adjustSolveCountsForChallenges(ctx, challengeIDs); err != nil {
			return fmt.Errorf("TeamUseCase - AdminDelete - adjustSolveCountsForChallenges: %w", err)
		}

		if err := uc.deps.UserRepo.UpdateTeamIDBatch(ctx, memberIDs, nil); err != nil {
			return fmt.Errorf("TeamUseCase - AdminDelete - UserRepo.UpdateTeamIDBatch: %w", err)
		}

		if err := uc.deps.TeamRepo.Delete(ctx, teamID); err != nil {
			return fmt.Errorf("TeamUseCase - AdminDelete - TeamRepo.Delete: %w", err)
		}

		auditLog := &domain.TeamAuditLog{
			TeamID:  teamID,
			UserID:  nil,
			Action:  domain.TeamActionDeleted,
			Details: map[string]any{domain.TeamAuditDetailReason: "deleted_by_admin"},
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
		cacheutil.InvalidateUser(ctx, uc.deps.UserCache, id)
	}

	cacheutil.InvalidateScoreboard(ctx, uc.deps.ScoreboardCache)
	cacheutil.InvalidateChallengeList(ctx, uc.deps.ChallengeListCache)

	return nil
}

func (uc *TeamUseCase) AdminGetMembers(ctx context.Context, teamID uuid.UUID) ([]*domain.User, error) {
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
	err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		return uc.adminAddMemberTx(ctx, teamID, userID)
	})
	if err != nil {
		return fmt.Errorf("TeamUseCase - AdminAddMember - TM.Run: %w", err)
	}

	cacheutil.InvalidateUser(ctx, uc.deps.UserCache, userID)
	cacheutil.InvalidateScoreboardForTeam(ctx, uc.deps.ScoreboardCache, teamID)

	return nil
}

// adminAddMemberTx locks the user then the team (user-before-team order), validates
// team capacity, competition mode, and user eligibility, then assigns the user.
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
		return apperr.ErrCannotAddToSoloTeam
	}

	if team.IsBanned {
		return apperr.ErrTeamBanned
	}

	user, err := uc.deps.UserRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("TeamUseCase - AdminAddMember - UserRepo.GetByID: %w", err)
	}

	if user.IsBanned {
		return apperr.ErrUserBanned
	}

	if user.WasInBannedTeam {
		return apperr.ErrUserWasInBannedTeam
	}

	if user.TeamID != nil {
		return apperr.ErrTeamConflict
	}

	members, err := uc.deps.UserRepo.GetByTeamID(ctx, teamID)
	if err != nil {
		return fmt.Errorf("TeamUseCase - AdminAddMember - UserRepo.GetByTeamID: %w", err)
	}

	comp, err := uc.deps.CompRepo.GetForUpdate(ctx)
	if err != nil {
		return fmt.Errorf("TeamUseCase - AdminAddMember - CompetitionRepo.GetForUpdate: %w", err)
	}

	if !comp.Mode.AllowsTeams() {
		return apperr.ErrTeamsNotAllowed
	}

	maxSize := resolveMaxTeamSize(comp, uc.deps.DefaultMaxTeamSize)

	if len(members) >= maxSize {
		return apperr.ErrTeamFull
	}

	if err := uc.deps.UserRepo.UpdateTeamID(ctx, userID, &teamID); err != nil {
		return fmt.Errorf("TeamUseCase - AdminAddMember - UserRepo.UpdateTeamID: %w", err)
	}

	auditLog := &domain.TeamAuditLog{
		TeamID:  teamID,
		UserID:  &userID,
		Action:  domain.TeamActionJoined,
		Details: map[string]any{domain.TeamAuditDetailReason: "added_by_admin"},
	}
	if err := uc.deps.TeamRepo.CreateAuditLog(ctx, auditLog); err != nil {
		return fmt.Errorf("TeamUseCase - AdminAddMember - TeamRepo.CreateAuditLog: %w", err)
	}

	return nil
}

// AdminRemoveMember removes a user from a team. MinTeamSize is intentionally not
// enforced so that admins retain full control (e.g. to fix roster or prepare for disband).
func (uc *TeamUseCase) AdminRemoveMember(ctx context.Context, teamID, userID uuid.UUID) error {
	err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
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
			return apperr.ErrTeamMemberNotFound
		}

		team, err := uc.deps.TeamRepo.GetByID(ctx, teamID)
		if err != nil {
			return fmt.Errorf("TeamUseCase - AdminRemoveMember - TeamRepo.GetByID: %w", err)
		}

		if team.CaptainID == userID {
			return apperr.ErrCaptainCannotLeave
		}

		if err := uc.deps.UserRepo.UpdateTeamID(ctx, userID, nil); err != nil {
			return fmt.Errorf("TeamUseCase - AdminRemoveMember - UserRepo.UpdateTeamID: %w", err)
		}

		auditLog := &domain.TeamAuditLog{
			TeamID:  teamID,
			UserID:  &userID,
			Action:  domain.TeamActionMemberKicked,
			Details: map[string]any{domain.TeamAuditDetailReason: "removed_by_admin"},
		}
		if err := uc.deps.TeamRepo.CreateAuditLog(ctx, auditLog); err != nil {
			return fmt.Errorf("TeamUseCase - AdminRemoveMember - TeamRepo.CreateAuditLog: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("TeamUseCase - AdminRemoveMember - TM.Run: %w", err)
	}

	cacheutil.InvalidateUser(ctx, uc.deps.UserCache, userID)
	cacheutil.InvalidateScoreboardForTeam(ctx, uc.deps.ScoreboardCache, teamID)

	return nil
}
