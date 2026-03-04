package competition

import (
	"context"
	"fmt"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

// competitionSource fetches competition data. Satisfied by *CompetitionUseCase
// (which uses a local+Redis cache) rather than going directly to the DB.
type competitionSource interface {
	Get(ctx context.Context) (*entity.Competition, error)
}

type Guard struct {
	src competitionSource
}

var _ usecase.CompetitionGuard = (*Guard)(nil)

func NewGuard(src competitionSource) *Guard {
	return &Guard{src: src}
}

func (g *Guard) Get(ctx context.Context) (*entity.Competition, error) {
	comp, err := g.src.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("CompetitionGuard - Get - src.Get: %w", err)
	}
	return comp, nil
}

func (g *Guard) RequireTeamSwitch(ctx context.Context) (*entity.Competition, error) {
	comp, err := g.src.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("CompetitionGuard - RequireTeamSwitch - src.Get: %w", err)
	}
	if comp.GetStatus() == entity.CompetitionStatusEnded {
		return nil, httperr.ErrCompetitionEnded
	}
	if !comp.AllowTeamSwitch {
		return nil, httperr.ErrRosterFrozen
	}
	return comp, nil
}

func (g *Guard) RequireTeamSwitchAndTeamsMode(ctx context.Context) (*entity.Competition, error) {
	comp, err := g.RequireTeamSwitch(ctx)
	if err != nil {
		return nil, fmt.Errorf("CompetitionGuard - RequireTeamSwitchAndTeamsMode - RequireTeamSwitch: %w", err)
	}
	if !comp.Mode.AllowsTeams() {
		return nil, httperr.ErrTeamsNotAllowed
	}
	return comp, nil
}

func (g *Guard) RequireTeamSwitchAndSoloMode(ctx context.Context) (*entity.Competition, error) {
	comp, err := g.RequireTeamSwitch(ctx)
	if err != nil {
		return nil, fmt.Errorf("CompetitionGuard - RequireTeamSwitchAndSoloMode - RequireTeamSwitch: %w", err)
	}
	if !comp.Mode.AllowsSolo() {
		return nil, httperr.ErrSoloModeNotAllowed
	}
	return comp, nil
}
