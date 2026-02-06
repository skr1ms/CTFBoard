package competition

import (
	"context"

	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/repo"
)

type Guard struct {
	repo repo.CompetitionRepository
}

func NewGuard(repo repo.CompetitionRepository) *Guard {
	return &Guard{repo: repo}
}

func (g *Guard) Get(ctx context.Context) (*entity.Competition, error) {
	return g.repo.Get(ctx)
}

func (g *Guard) RequireTeamSwitch(ctx context.Context) (*entity.Competition, error) {
	comp, err := g.repo.Get(ctx)
	if err != nil {
		return nil, err
	}
	if !comp.AllowTeamSwitch {
		return nil, entityError.ErrRosterFrozen
	}
	return comp, nil
}

func (g *Guard) RequireTeamSwitchAndTeamsMode(ctx context.Context) (*entity.Competition, error) {
	comp, err := g.RequireTeamSwitch(ctx)
	if err != nil {
		return nil, err
	}
	mode := entity.CompetitionMode(comp.Mode)
	if !mode.AllowsTeams() {
		return nil, entityError.ErrSoloModeNotAllowed
	}
	return comp, nil
}

func (g *Guard) RequireTeamSwitchAndSoloMode(ctx context.Context) (*entity.Competition, error) {
	comp, err := g.RequireTeamSwitch(ctx)
	if err != nil {
		return nil, err
	}
	mode := entity.CompetitionMode(comp.Mode)
	if !mode.AllowsSolo() {
		return nil, entityError.ErrSoloModeNotAllowed
	}
	return comp, nil
}
