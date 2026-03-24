// Package team implements team use cases. Lock order for concurrent safety:
// always lock user(s) first in canonical order (by UUID string), then team(s)
// in canonical order (orderTeamLockIDs). Never lock team before user when both are needed.
package team

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-logkit"
	"golang.org/x/sync/errgroup"

	"github.com/wahrwelt-kit/go-cachekit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"

	"github.com/TakuyaYagam1/AstroCTFb/pkg/cache"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

type TeamUseCase struct {
	deps TeamDeps
}

type JWTRevoker interface {
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
}

type TeamDeps struct {
	TeamRepo           repo.TeamRepository
	UserRepo           repo.UserRepository
	SolveRepo          repo.SolveRepository
	SubmissionRepo     repo.SubmissionRepository
	AwardRepo          repo.AwardRepository
	CompRepo           repo.CompetitionRepository
	SettingsGetter     usecase.SettingsGetter
	ChallengeRepo      repo.ChallengeRepository
	TM                 repo.TransactionManager
	Guard              usecase.CompetitionGuard
	ScoreboardCache    cache.ScoreboardCacheInvalidator
	ChallengeListCache cache.ChallengeListCacheInvalidator
	UserCache          cache.UserCacheInvalidator
	TeamCache          *cachekit.Cache
	HintRepo           repo.HintRepository
	FieldValueRepo     repo.FieldValueRepository
	JWTRevoker         JWTRevoker
	DefaultMaxTeamSize int
	Logger             logkit.Logger
}

var _ usecase.TeamUseCase = (*TeamUseCase)(nil)

func NewTeamUseCase(deps TeamDeps) *TeamUseCase {
	if deps.DefaultMaxTeamSize <= 0 {
		deps.DefaultMaxTeamSize = 10
	}
	if deps.Logger == nil {
		deps.Logger = logkit.Noop()
	}
	return &TeamUseCase{deps: deps}
}

func (uc *TeamUseCase) GetByID(ctx context.Context, ID uuid.UUID) (*domain.Team, error) {
	team, err := uc.deps.TeamRepo.GetByID(ctx, ID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - GetByID - TeamRepo.GetByID: %w", err)
	}
	return team, nil
}

func (uc *TeamUseCase) GetMyTeam(ctx context.Context, userID uuid.UUID) (*domain.Team, []*domain.User, int, bool, error) {
	user, err := uc.deps.UserRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, nil, 0, false, fmt.Errorf("TeamUseCase - GetMyTeam - UserRepo.GetByID: %w", err)
	}
	if user.TeamID == nil {
		return nil, nil, 0, false, httperr.ErrUserNotInTeam
	}
	teamID := *user.TeamID

	var team *domain.Team
	var members []*domain.User
	g, gCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		var err2 error
		team, err2 = uc.deps.TeamRepo.GetByID(gCtx, teamID)
		if err2 != nil {
			return fmt.Errorf("TeamUseCase - GetMyTeam - TeamRepo.GetByID: %w", err2)
		}
		return nil
	})
	g.Go(func() error {
		var err2 error
		members, err2 = uc.deps.UserRepo.GetByTeamID(gCtx, teamID)
		if err2 != nil {
			return fmt.Errorf("TeamUseCase - GetMyTeam - UserRepo.GetByTeamID: %w", err2)
		}
		return nil
	})
	if err := g.Wait(); err != nil {
		return nil, nil, 0, false, fmt.Errorf("TeamUseCase - GetMyTeam - errgroup.Wait: %w", err)
	}
	minTeamSize := 0
	meetsMinSize := true
	if uc.deps.CompRepo != nil {
		comp, err := uc.deps.CompRepo.Get(ctx)
		if err == nil && comp != nil && comp.MinTeamSize > 0 {
			minTeamSize = comp.MinTeamSize
			meetsMinSize = len(members) >= comp.MinTeamSize
		}
	}
	return team, members, minTeamSize, meetsMinSize, nil
}

func (uc *TeamUseCase) GetTeamMembers(ctx context.Context, teamID uuid.UUID) ([]*domain.User, error) {
	users, err := uc.deps.UserRepo.GetByTeamID(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - GetTeamMembers - UserRepo.GetByTeamID: %w", err)
	}
	return users, nil
}

func (uc *TeamUseCase) SetHidden(ctx context.Context, teamID uuid.UUID, hidden bool) error {
	err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		if err := uc.deps.TeamRepo.Lock(ctx, teamID); err != nil {
			return fmt.Errorf("TeamUseCase - SetHidden - TeamRepo.Lock: %w", err)
		}
		_, err := uc.deps.TeamRepo.GetByID(ctx, teamID)
		if err != nil {
			return fmt.Errorf("TeamUseCase - SetHidden - TeamRepo.GetByID: %w", err)
		}
		if err := uc.deps.TeamRepo.SetHidden(ctx, teamID, hidden); err != nil {
			return fmt.Errorf("TeamUseCase - SetHidden - TeamRepo.SetHidden: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("TeamUseCase - SetHidden - TM.Run: %w", err)
	}
	uc.invalidateScoreboardCacheForTeam(ctx, teamID)
	return nil
}

func (uc *TeamUseCase) SetBracket(ctx context.Context, teamID uuid.UUID, bracketID *uuid.UUID) error {
	err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		if err := uc.deps.TeamRepo.Lock(ctx, teamID); err != nil {
			return fmt.Errorf("TeamUseCase - SetBracket - TeamRepo.Lock: %w", err)
		}
		_, err := uc.deps.TeamRepo.GetByID(ctx, teamID)
		if err != nil {
			return fmt.Errorf("TeamUseCase - SetBracket - TeamRepo.GetByID: %w", err)
		}
		if err := uc.deps.TeamRepo.SetBracket(ctx, teamID, bracketID); err != nil {
			return fmt.Errorf("TeamUseCase - SetBracket - TeamRepo.SetBracket: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("TeamUseCase - SetBracket - TM.Run: %w", err)
	}
	uc.invalidateScoreboardCacheForTeam(ctx, teamID)
	return nil
}

func (uc *TeamUseCase) invalidateScoreboardCache(ctx context.Context) {
	if uc.deps.ScoreboardCache != nil {
		uc.deps.ScoreboardCache.InvalidateAll(ctx)
	}
}

func (uc *TeamUseCase) invalidateScoreboardCacheForTeam(ctx context.Context, teamID uuid.UUID) {
	if uc.deps.ScoreboardCache != nil {
		uc.deps.ScoreboardCache.InvalidateForTeam(ctx, teamID)
	}
}

func (uc *TeamUseCase) invalidateChallengeListCache(ctx context.Context) {
	if uc.deps.ChallengeListCache != nil {
		uc.deps.ChallengeListCache.InvalidateAll(ctx)
	}
}

func (uc *TeamUseCase) invalidateUserCache(ctx context.Context, userID uuid.UUID) {
	if uc.deps.UserCache != nil {
		uc.deps.UserCache.InvalidateUser(ctx, userID)
	}
}

func (uc *TeamUseCase) invalidateTeamCache(ctx context.Context, teamID uuid.UUID) {
	if uc.deps.TeamCache != nil {
		if err := uc.deps.TeamCache.Del(ctx, cache.KeyTeam(teamID.String())); err != nil && uc.deps.Logger != nil {
			uc.deps.Logger.WithError(err).Warn("TeamUseCase - invalidateTeamCache - Del")
		}
	}
}

func (uc *TeamUseCase) ListTeams(ctx context.Context, search *string, page, perPage int) (*usecase.Paginated[*domain.Team], error) {
	var result *usecase.Paginated[*domain.Team]
	if err := uc.deps.TM.ReadOnly(ctx, func(roCtx context.Context) error {
		var err error
		result, err = usecase.FetchPage(roCtx, page, perPage,
			func(ctx context.Context, limit, offset int) ([]*domain.Team, error) {
				return uc.deps.TeamRepo.Search(ctx, search, limit, offset)
			},
			func(ctx context.Context) (int64, error) {
				return uc.deps.TeamRepo.CountSearch(ctx, search)
			},
		)
		if err != nil {
			return fmt.Errorf("TeamUseCase - ListTeams: %w", err)
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("TeamUseCase - ListTeams: %w", err)
	}
	return result, nil
}

func (uc *TeamUseCase) GetTeamSolves(ctx context.Context, teamID uuid.UUID) ([]*domain.SolveWithDetails, error) {
	team, err := uc.deps.TeamRepo.GetByID(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - GetTeamSolves - TeamRepo.GetByID: %w", err)
	}
	if team.IsBanned {
		return nil, httperr.ErrTeamBanned
	}
	solves, err := uc.deps.SolveRepo.GetByTeamIDWithDetails(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - GetTeamSolves - SolveRepo.GetByTeamIDWithDetails: %w", err)
	}
	return solves, nil
}

func (uc *TeamUseCase) GetTeamFails(ctx context.Context, teamID uuid.UUID, page, perPage int) (*usecase.Paginated[*domain.SubmissionWithDetails], error) {
	team, err := uc.deps.TeamRepo.GetByID(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - GetTeamFails - TeamRepo.GetByID: %w", err)
	}
	if team.IsBanned {
		return nil, httperr.ErrTeamBanned
	}
	result, err := usecase.FetchPage(ctx, page, perPage,
		func(ctx context.Context, limit, offset int) ([]*domain.SubmissionWithDetails, error) {
			return uc.deps.SubmissionRepo.GetFailsByTeam(ctx, teamID, limit, offset)
		},
		func(ctx context.Context) (int64, error) {
			return uc.deps.SubmissionRepo.CountFailsByTeam(ctx, teamID)
		},
	)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - GetTeamFails: %w", err)
	}
	return result, nil
}

func (uc *TeamUseCase) GetTeamAwards(ctx context.Context, teamID uuid.UUID) ([]*domain.Award, error) {
	team, err := uc.deps.TeamRepo.GetByID(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - GetTeamAwards - TeamRepo.GetByID: %w", err)
	}
	if team.IsBanned {
		return nil, httperr.ErrTeamBanned
	}
	if uc.deps.AwardRepo == nil {
		return []*domain.Award{}, nil
	}
	awards, err := uc.deps.AwardRepo.GetByTeamID(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - GetTeamAwards - AwardRepo.GetByTeamID: %w", err)
	}
	return awards, nil
}

func (uc *TeamUseCase) UpdateMyTeam(ctx context.Context, captainID uuid.UUID, name string) (*domain.Team, error) {
	if _, err := uc.deps.Guard.RequireTeamSwitch(ctx); err != nil {
		return nil, fmt.Errorf("TeamUseCase - UpdateMyTeam - Guard.RequireTeamSwitch: %w", err)
	}
	var team *domain.Team
	err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		var err error
		team, err = uc.updateMyTeamTx(ctx, captainID, name)
		if err != nil {
			return fmt.Errorf("TeamUseCase - UpdateMyTeam - updateMyTeamTx: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - UpdateMyTeam - TM.Run: %w", err)
	}
	if team != nil {
		uc.invalidateTeamCache(ctx, team.ID)
		uc.invalidateScoreboardCache(ctx)
	}
	return team, nil
}

func (uc *TeamUseCase) updateMyTeamTx(ctx context.Context, captainID uuid.UUID, name string) (*domain.Team, error) {
	comp, err := uc.deps.CompRepo.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - updateMyTeamTx - CompetitionRepo.Get: %w", err)
	}
	if err := uc.requireTeamSwitch(comp); err != nil {
		return nil, fmt.Errorf("TeamUseCase - updateMyTeamTx - requireTeamSwitch: %w", err)
	}
	if err := uc.deps.UserRepo.Lock(ctx, captainID); err != nil {
		return nil, fmt.Errorf("TeamUseCase - updateMyTeamTx - UserRepo.Lock: %w", err)
	}
	user, err := uc.deps.UserRepo.GetByID(ctx, captainID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - updateMyTeamTx - UserRepo.GetByID: %w", err)
	}
	if user.TeamID == nil {
		return nil, httperr.ErrTeamNotFound
	}
	if user.IsBanned {
		return nil, httperr.ErrUserBanned
	}
	if err := uc.deps.TeamRepo.Lock(ctx, *user.TeamID); err != nil {
		return nil, fmt.Errorf("TeamUseCase - updateMyTeamTx - TeamRepo.Lock: %w", err)
	}
	team, err := uc.deps.TeamRepo.GetByID(ctx, *user.TeamID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - updateMyTeamTx - TeamRepo.GetByID: %w", err)
	}
	if team.IsBanned {
		return nil, httperr.ErrTeamBanned
	}
	if team.CaptainID != captainID {
		return nil, httperr.ErrNotCaptain
	}
	if team.Name != name {
		if err := uc.validateTeamNameAvailable(ctx, name); err != nil {
			return nil, fmt.Errorf("TeamUseCase - updateMyTeamTx - validateTeamNameAvailable: %w", err)
		}
	}
	if err := uc.deps.TeamRepo.UpdateName(ctx, team.ID, name); err != nil {
		return nil, fmt.Errorf("TeamUseCase - updateMyTeamTx - TeamRepo.UpdateName: %w", err)
	}
	team.Name = name
	return team, nil
}

func (uc *TeamUseCase) GetInviteToken(ctx context.Context, captainID uuid.UUID) (*domain.Team, error) {
	if _, err := uc.deps.Guard.RequireTeamSwitch(ctx); err != nil {
		return nil, fmt.Errorf("TeamUseCase - GetInviteToken - Guard.RequireTeamSwitch: %w", err)
	}
	user, err := uc.deps.UserRepo.GetByID(ctx, captainID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - GetInviteToken - UserRepo.GetByID: %w", err)
	}
	if user.IsBanned {
		return nil, httperr.ErrUserBanned
	}
	if user.TeamID == nil {
		return nil, httperr.ErrTeamNotFound
	}
	team, err := uc.deps.TeamRepo.GetByID(ctx, *user.TeamID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - GetInviteToken - TeamRepo.GetByID: %w", err)
	}
	if team.CaptainID != captainID {
		return nil, httperr.ErrNotCaptain
	}
	if team.IsBanned {
		return nil, httperr.ErrTeamBanned
	}
	return team, nil
}

const defaultInviteTokenTTL = 7 * 24 * time.Hour

func (uc *TeamUseCase) RegenerateInviteToken(ctx context.Context, captainID uuid.UUID) (*domain.Team, error) {
	var team *domain.Team
	err := uc.deps.TM.Run(ctx, func(txCtx context.Context) error {
		comp, err := uc.deps.CompRepo.Get(txCtx)
		if err != nil {
			return fmt.Errorf("TeamUseCase - RegenerateInviteToken - CompetitionRepo.Get: %w", err)
		}
		if err := uc.requireTeamSwitch(comp); err != nil {
			return fmt.Errorf("TeamUseCase - RegenerateInviteToken - requireTeamSwitch: %w", err)
		}
		if err := uc.deps.UserRepo.Lock(txCtx, captainID); err != nil {
			return fmt.Errorf("TeamUseCase - RegenerateInviteToken - UserRepo.Lock: %w", err)
		}
		user, err := uc.deps.UserRepo.GetByID(txCtx, captainID)
		if err != nil {
			return fmt.Errorf("TeamUseCase - RegenerateInviteToken - UserRepo.GetByID: %w", err)
		}
		if user.IsBanned {
			return httperr.ErrUserBanned
		}
		if user.TeamID == nil {
			return httperr.ErrTeamNotFound
		}
		if err := uc.deps.TeamRepo.Lock(txCtx, *user.TeamID); err != nil {
			return fmt.Errorf("TeamUseCase - RegenerateInviteToken - TeamRepo.Lock: %w", err)
		}
		var err2 error
		team, err2 = uc.deps.TeamRepo.GetByID(txCtx, *user.TeamID)
		if err2 != nil {
			return fmt.Errorf("TeamUseCase - RegenerateInviteToken - TeamRepo.GetByID: %w", err2)
		}
		if team.CaptainID != captainID {
			return httperr.ErrNotCaptain
		}
		if team.IsSolo {
			return httperr.ErrTeamNotFound
		}
		if team.IsBanned {
			return httperr.ErrTeamBanned
		}
		newToken := uuid.New()
		expiresAt := time.Now().Add(defaultInviteTokenTTL)
		if err := uc.deps.TeamRepo.UpdateInviteToken(txCtx, team.ID, newToken, &expiresAt); err != nil {
			return fmt.Errorf("TeamUseCase - RegenerateInviteToken - TeamRepo.UpdateInviteToken: %w", err)
		}
		team.InviteToken = newToken
		team.InviteTokenExpiresAt = &expiresAt
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - RegenerateInviteToken - TM.Run: %w", err)
	}
	return team, nil
}
