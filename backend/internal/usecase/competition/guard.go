// Package competition provides competition state and guards for team/solo operations
//
// CompetitionGuard uses cached competition (not the current transaction snapshot)
// TOCTOU: AllowTeamSwitch/mode could change between Guard check and write within cache TTL
// Risk is closed: usecases re-check competition state inside transactions before writing
// competition.Update forbids changing mode/start_time when competition is active
package competition

import (
	"context"
	"fmt"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/guard"
)

// competitionSource fetches competition data. Satisfied by *CompetitionUseCase
// (which uses a local+Redis cache) rather than going directly to the DB
// Trade-off: when Guard is used inside a transaction (e.g. team Create/Join),
// the competition state is read from cache, not from the current tx. AllowTeamSwitch
// could theoretically change between the Guard check and the write; the window is
// at most the cache TTL (local 5s, Redis 15s). Competition.Update forbids
// changing mode/start_time when active, so the risk is acceptable.
type competitionSource interface {
	Get(ctx context.Context) (*domain.Competition, error)
}

type Guard struct {
	src competitionSource
}

var _ usecase.CompetitionGuard = (*Guard)(nil)

func NewGuard(src competitionSource) *Guard {
	return &Guard{src: src}
}

func (g *Guard) Get(ctx context.Context) (*domain.Competition, error) {
	comp, err := g.src.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("CompetitionGuard - Get - src.Get: %w", err)
	}

	return comp, nil
}

// RequireTeamSwitch returns the competition if team roster changes are allowed
// When AllowTeamSwitch is false (roster frozen), Create/Join/Leave/Disband/Kick/TransferCaptain
// are blocked. In teams_only mode with AllowTeamSwitch=false, new users cannot create or
// join a team; admin can add them via AdminAddMember which bypasses this guard.
func (g *Guard) RequireTeamSwitch(ctx context.Context) (*domain.Competition, error) {
	comp, err := g.src.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("CompetitionGuard - RequireTeamSwitch - src.Get: %w", err)
	}

	if err := guard.ValidateTeamSwitchState(comp); err != nil {
		return nil, err
	}

	return comp, nil
}

func (g *Guard) RequireTeamSwitchAndTeamsMode(ctx context.Context) (*domain.Competition, error) {
	comp, err := g.RequireTeamSwitch(ctx)
	if err != nil {
		return nil, fmt.Errorf("CompetitionGuard - RequireTeamSwitchAndTeamsMode - RequireTeamSwitch: %w", err)
	}

	if !comp.Mode.AllowsTeams() {
		return nil, apperr.ErrTeamsNotAllowed
	}

	return comp, nil
}

func (g *Guard) RequireTeamSwitchAndSoloMode(ctx context.Context) (*domain.Competition, error) {
	comp, err := g.RequireTeamSwitch(ctx)
	if err != nil {
		return nil, fmt.Errorf("CompetitionGuard - RequireTeamSwitchAndSoloMode - RequireTeamSwitch: %w", err)
	}

	if !comp.Mode.AllowsSolo() {
		return nil, apperr.ErrSoloModeNotAllowed
	}

	return comp, nil
}
