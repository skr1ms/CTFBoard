package team

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

func (uc *TeamUseCase) AdminListTeams(ctx context.Context, search *string, page, perPage int) (*usecase.Paginated[*domain.Team], error) {
	var result *usecase.Paginated[*domain.Team]

	err := uc.deps.TM.ReadOnly(ctx, func(roCtx context.Context) error {
		var err error

		result, err = usecase.FetchPage(roCtx, page, perPage,
			func(ctx context.Context, limit, offset int) ([]*domain.Team, error) {
				return uc.deps.TeamRepo.SearchAdmin(ctx, search, limit, offset)
			},
			func(ctx context.Context) (int64, error) {
				return uc.deps.TeamRepo.CountSearchAdmin(ctx, search)
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
				return httperr.ErrNewCaptainNotInTeam
			}

			if candidate.IsBanned {
				return httperr.ErrUserBanned
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

	uc.invalidateScoreboardCache(ctx)

	return team, nil
}

func (uc *TeamUseCase) AdminDelete(ctx context.Context, teamID uuid.UUID) error {
	var memberIDs []uuid.UUID

	err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		members, err := uc.deps.UserRepo.GetByTeamID(ctx, teamID)
		if err != nil {
			return fmt.Errorf("TeamUseCase - AdminDelete - UserRepo.GetByTeamID: %w", err)
		}

		slices.SortFunc(members, func(a, b *domain.User) int {
			return strings.Compare(a.ID.String(), b.ID.String())
		})

		for _, m := range members {
			err := uc.deps.UserRepo.Lock(ctx, m.ID)
			if err != nil {
				return fmt.Errorf("TeamUseCase - AdminDelete - UserRepo.Lock: %w", err)
			}
		}

		if err := uc.deps.TeamRepo.Lock(ctx, teamID); err != nil {
			return fmt.Errorf("TeamUseCase - AdminDelete - TeamRepo.Lock: %w", err)
		}

		membersAfter, err := uc.deps.UserRepo.GetByTeamID(ctx, teamID)
		if err != nil {
			return fmt.Errorf("TeamUseCase - AdminDelete - UserRepo.GetByTeamID (recheck): %w", err)
		}

		slices.SortFunc(membersAfter, func(a, b *domain.User) int {
			return strings.Compare(a.ID.String(), b.ID.String())
		})

		if len(membersAfter) != len(members) {
			return httperr.ErrTeamConflict
		}

		for i := range members {
			if members[i].ID != membersAfter[i].ID {
				return httperr.ErrTeamConflict
			}
		}

		memberIDs = make([]uuid.UUID, len(membersAfter))
		for i, m := range membersAfter {
			memberIDs[i] = m.ID
		}

		if err := uc.adjustSolveCountsForTeam(ctx, teamID, true); err != nil {
			return fmt.Errorf("TeamUseCase - AdminDelete - adjustSolveCountsForTeam: %w", err)
		}

		if err := uc.deps.SolveRepo.DeleteByTeamID(ctx, teamID); err != nil {
			return fmt.Errorf("TeamUseCase - AdminDelete - SolveRepo.DeleteByTeamID: %w", err)
		}

		if err := uc.deps.SubmissionRepo.DeleteByTeamID(ctx, teamID); err != nil {
			return fmt.Errorf("TeamUseCase - AdminDelete - SubmissionRepo.DeleteByTeamID: %w", err)
		}

		if err := uc.deps.AwardRepo.DeleteByTeamID(ctx, teamID); err != nil {
			return fmt.Errorf("TeamUseCase - AdminDelete - AwardRepo.DeleteByTeamID: %w", err)
		}

		if uc.deps.HintRepo != nil {
			err := uc.deps.HintRepo.DeleteUnlocksByTeamID(ctx, teamID)
			if err != nil {
				return fmt.Errorf("TeamUseCase - AdminDelete - HintRepo.DeleteUnlocksByTeamID: %w", err)
			}
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
	uc.invalidateChallengeListCache(ctx)

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

	uc.invalidateUserCache(ctx, userID)
	uc.invalidateScoreboardCache(ctx)

	return nil
}

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

	if user.WasInBannedTeam {
		return httperr.ErrUserWasInBannedTeam
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

	auditLog := &domain.TeamAuditLog{
		TeamID:  teamID,
		UserID:  &userID,
		Action:  domain.TeamActionJoined,
		Details: map[string]any{"reason": "added_by_admin"},
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

		auditLog := &domain.TeamAuditLog{
			TeamID:  teamID,
			UserID:  &userID,
			Action:  domain.TeamActionMemberKicked,
			Details: map[string]any{"reason": "removed_by_admin"},
		}
		if err := uc.deps.TeamRepo.CreateAuditLog(ctx, auditLog); err != nil {
			return fmt.Errorf("TeamUseCase - AdminRemoveMember - TeamRepo.CreateAuditLog: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("TeamUseCase - AdminRemoveMember - TM.Run: %w", err)
	}

	uc.invalidateUserCache(ctx, userID)
	uc.invalidateScoreboardCache(ctx)

	return nil
}
