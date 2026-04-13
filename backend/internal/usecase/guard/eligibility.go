package guard

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

// EnsureChallengeVisible returns ErrChallengeNotFound when challenge is in hidden state.
// Used to normalise the 14 identical inline checks across usecase packages.
func EnsureChallengeVisible(ch *domain.Challenge) error {
	if ch.State == domain.ChallengeStateHidden {
		return apperr.ErrChallengeNotFound
	}

	return nil
}

// TeamMemberCounter counts members in a team, used for MinTeamSize enforcement.
type TeamMemberCounter interface {
	CountTeamMembers(ctx context.Context, teamID uuid.UUID) (int, error)
}

// ValidateSubmissionEligibility checks whether a user/team is allowed to submit or unlock hints
// given the current competition settings. It enforces:
//   - user not banned
//   - team not banned
//   - competition mode (solo-only / team-only)
//   - minimum team size
//
// user and team must already be fetched by the caller. comp may be nil (open competition).
func ValidateSubmissionEligibility(ctx context.Context, user *domain.User, team *domain.Team, comp *domain.Competition, teamRepo TeamMemberCounter) error {
	if user != nil && user.IsBanned {
		return apperr.ErrUserBanned
	}

	if team == nil {
		return nil
	}

	if team.IsBanned {
		return apperr.ErrTeamBanned
	}

	if comp == nil {
		return nil
	}

	if !comp.Mode.AllowsSolo() && team.IsSolo {
		return apperr.ErrTeamModeRequired
	}

	if !comp.Mode.AllowsTeams() && !team.IsSolo {
		return apperr.ErrSoloModeRequired
	}

	if comp.MinTeamSize > 0 && !team.IsSolo && teamRepo != nil {
		count, err := teamRepo.CountTeamMembers(ctx, team.ID)
		if err != nil {
			return fmt.Errorf("ValidateSubmissionEligibility - CountTeamMembers: %w", err)
		}

		if count < comp.MinTeamSize {
			return apperr.ErrTeamBelowMinSize
		}
	}

	return nil
}

// ValidateTeamSwitchState checks whether roster changes (create/join/leave/kick/disband) are
// allowed given the current competition state and AllowTeamSwitch flag. Pure function - no I/O.
func ValidateTeamSwitchState(comp *domain.Competition) error {
	switch comp.GetStatus() {
	case domain.CompetitionStatusEnded:
		return apperr.ErrCompetitionEnded
	case domain.CompetitionStatusPaused:
		return apperr.ErrCompetitionPaused
	case domain.CompetitionStatusNotStarted, domain.CompetitionStatusActive, domain.CompetitionStatusFrozen:
		// allowed - proceed to AllowTeamSwitch check below
	}

	if !comp.AllowTeamSwitch {
		return apperr.ErrRosterFrozen
	}

	return nil
}
