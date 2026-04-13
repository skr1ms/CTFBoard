// Package team implements team use cases. Lock order for concurrent safety
// always lock user(s) first in canonical order (by UUID string), then team(s)
// in canonical order (orderTeamLockIDs). Never lock team before user when both are needed
package team

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-cachekit"
	"github.com/wahrwelt-kit/go-logkit"
	"golang.org/x/sync/errgroup"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/cache"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/scoring"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/cacheutil"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/guard"
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
	CompParamUC        usecase.CompetitionParamUseCase
	TM                 repo.TransactionManager
	Guard              usecase.CompetitionGuard
	ScoreboardCache    cache.ScoreboardCacheInvalidator
	ChallengeListCache cache.ChallengeListCacheInvalidator
	UserCache          cache.UserCacheInvalidator
	TeamCache          *cachekit.Cache
	HintRepo           repo.HintRepository
	RatingRepo         repo.RatingRepository
	FieldValueRepo     repo.FieldValueRepository
	JWTRevoker         JWTRevoker
	DefaultMaxTeamSize int
	Logger             logkit.Logger
}

const defaultMaxTeamSize = 10

var _ usecase.TeamUseCase = (*TeamUseCase)(nil)

func NewTeamUseCase(deps TeamDeps) *TeamUseCase {
	if deps.DefaultMaxTeamSize <= 0 {
		deps.DefaultMaxTeamSize = defaultMaxTeamSize
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

// GetMyTeam returns the caller's team along with its members, active solve count, and captain flag.
// Uses errgroup for concurrent fan-out: team, members, solves, and awards are fetched in parallel.
func (uc *TeamUseCase) GetMyTeam(ctx context.Context, userID uuid.UUID) (*domain.Team, []*domain.User, int, bool, error) {
	user, err := uc.deps.UserRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, nil, 0, false, fmt.Errorf("TeamUseCase - GetMyTeam - UserRepo.GetByID: %w", err)
	}

	if user.TeamID == nil {
		return nil, nil, 0, false, apperr.ErrUserNotInTeam
	}

	teamID := *user.TeamID

	var (
		team    *domain.Team
		members []*domain.User
	)

	g, gCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		var errTeam error

		team, errTeam = uc.deps.TeamRepo.GetByID(gCtx, teamID)
		if errTeam != nil {
			return fmt.Errorf("TeamUseCase - GetMyTeam - TeamRepo.GetByID: %w", errTeam)
		}

		return nil
	})
	g.Go(func() error {
		var errMembers error

		members, errMembers = uc.deps.UserRepo.GetByTeamID(gCtx, teamID)
		if errMembers != nil {
			return fmt.Errorf("TeamUseCase - GetMyTeam - UserRepo.GetByTeamID: %w", errMembers)
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

// SetHidden locks the team row, updates the hidden flag, then invalidates the
// scoreboard cache for that team only. The lock prevents a concurrent visibility
// flip from racing with score recalculations.
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

	cacheutil.InvalidateScoreboardForTeam(ctx, uc.deps.ScoreboardCache, teamID)

	return nil
}

// SetBracket locks the team row, assigns the bracket (nil removes it), then invalidates
// the scoreboard cache. Scoreboard entries are bracket-partitioned so any bracket change
// requires cache eviction.
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

	cacheutil.InvalidateScoreboardForTeam(ctx, uc.deps.ScoreboardCache, teamID)

	return nil
}

// ListTeams runs Search + CountSearch inside a read-only snapshot transaction so that
// the page slice and total count are consistent with each other.
func (uc *TeamUseCase) ListTeams(ctx context.Context, search *string, page, perPage int) (*usecase.Paginated[*domain.Team], error) {
	var result *usecase.Paginated[*domain.Team]

	err := uc.deps.TM.ReadOnly(ctx, func(roCtx context.Context) error {
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
	})
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - ListTeams: %w", err)
	}

	return result, nil
}

// GetTeamSolves returns the team's solves, excluding any that occurred after the
// competition freeze time. Banned teams always return ErrTeamBanned.
func (uc *TeamUseCase) GetTeamSolves(ctx context.Context, teamID uuid.UUID) ([]*domain.SolveWithDetails, error) {
	team, err := uc.deps.TeamRepo.GetByID(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - GetTeamSolves - TeamRepo.GetByID: %w", err)
	}

	if team.IsBanned {
		return nil, apperr.ErrTeamBanned
	}

	solves, err := uc.deps.SolveRepo.GetByTeamIDWithDetails(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - GetTeamSolves - SolveRepo.GetByTeamIDWithDetails: %w", err)
	}

	return scoring.FilterSolveDetailsByFreezeFromRepo(ctx, uc.deps.CompRepo, solves)
}

// GetTeamFails returns paginated failed submissions. Banned teams return ErrTeamBanned;
// freeze-time filtering is not applied to fails (they are not public scoring data).
func (uc *TeamUseCase) GetTeamFails(ctx context.Context, teamID uuid.UUID, page, perPage int) (*usecase.Paginated[*domain.SubmissionWithDetails], error) {
	team, err := uc.deps.TeamRepo.GetByID(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - GetTeamFails - TeamRepo.GetByID: %w", err)
	}

	if team.IsBanned {
		return nil, apperr.ErrTeamBanned
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

// GetTeamAwards returns the team's awards. Returns an empty slice (not an error) when
// AwardRepo is not wired, so callers can treat awards as an optional feature.
func (uc *TeamUseCase) GetTeamAwards(ctx context.Context, teamID uuid.UUID) ([]*domain.Award, error) {
	team, err := uc.deps.TeamRepo.GetByID(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - GetTeamAwards - TeamRepo.GetByID: %w", err)
	}

	if team.IsBanned {
		return nil, apperr.ErrTeamBanned
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

// UpdateMyTeam validates the competition-level team-switch guard, runs updateMyTeamTx
// in a transaction, then evicts both the team-specific cache entry and the full
// scoreboard cache (name change affects public display).
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
		cacheutil.InvalidateTeam(ctx, uc.deps.TeamCache, uc.deps.Logger, team.ID)
		cacheutil.InvalidateScoreboardForTeam(ctx, uc.deps.ScoreboardCache, team.ID)
	}

	return team, nil
}

// updateMyTeamTx performs the team name update inside a transaction: locks the team row,
// validates name uniqueness, recomputes the slug, and persists the change.
func (uc *TeamUseCase) updateMyTeamTx(ctx context.Context, captainID uuid.UUID, name string) (*domain.Team, error) {
	comp, err := uc.deps.CompRepo.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - updateMyTeamTx - CompetitionRepo.Get: %w", err)
	}

	if err := guard.ValidateTeamSwitchState(comp); err != nil {
		return nil, fmt.Errorf("TeamUseCase - updateMyTeamTx - ValidateTeamSwitchState: %w", err)
	}

	if err := uc.deps.UserRepo.Lock(ctx, captainID); err != nil {
		return nil, fmt.Errorf("TeamUseCase - updateMyTeamTx - UserRepo.Lock: %w", err)
	}

	user, err := uc.deps.UserRepo.GetByID(ctx, captainID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - updateMyTeamTx - UserRepo.GetByID: %w", err)
	}

	if user.TeamID == nil {
		return nil, apperr.ErrTeamNotFound
	}

	if user.IsBanned {
		return nil, apperr.ErrUserBanned
	}

	if err := uc.deps.TeamRepo.Lock(ctx, *user.TeamID); err != nil {
		return nil, fmt.Errorf("TeamUseCase - updateMyTeamTx - TeamRepo.Lock: %w", err)
	}

	team, err := uc.deps.TeamRepo.GetByID(ctx, *user.TeamID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - updateMyTeamTx - TeamRepo.GetByID: %w", err)
	}

	if team.IsBanned {
		return nil, apperr.ErrTeamBanned
	}

	if team.CaptainID != captainID {
		return nil, apperr.ErrNotCaptain
	}

	if team.Name != name {
		err := uc.validateTeamNameAvailable(ctx, name)
		if err != nil {
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
		return nil, apperr.ErrUserBanned
	}

	if user.TeamID == nil {
		return nil, apperr.ErrTeamNotFound
	}

	team, err := uc.deps.TeamRepo.GetByID(ctx, *user.TeamID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - GetInviteToken - TeamRepo.GetByID: %w", err)
	}

	if team.CaptainID != captainID {
		return nil, apperr.ErrNotCaptain
	}

	if team.IsBanned {
		return nil, apperr.ErrTeamBanned
	}

	return team, nil
}

const defaultInviteTokenTTL = 7 * 24 * time.Hour

// RegenerateInviteToken rotates the team invite token inside a transaction and returns the updated team.
func (uc *TeamUseCase) RegenerateInviteToken(ctx context.Context, captainID uuid.UUID) (*domain.Team, error) {
	var team *domain.Team

	err := uc.deps.TM.Run(ctx, func(txCtx context.Context) error {
		comp, err := uc.deps.CompRepo.Get(txCtx)
		if err != nil {
			return fmt.Errorf("TeamUseCase - RegenerateInviteToken - CompetitionRepo.Get: %w", err)
		}

		if err := guard.ValidateTeamSwitchState(comp); err != nil {
			return fmt.Errorf("TeamUseCase - RegenerateInviteToken - ValidateTeamSwitchState: %w", err)
		}

		if err := uc.deps.UserRepo.Lock(txCtx, captainID); err != nil {
			return fmt.Errorf("TeamUseCase - RegenerateInviteToken - UserRepo.Lock: %w", err)
		}

		user, err := uc.deps.UserRepo.GetByID(txCtx, captainID)
		if err != nil {
			return fmt.Errorf("TeamUseCase - RegenerateInviteToken - UserRepo.GetByID: %w", err)
		}

		if user.IsBanned {
			return apperr.ErrUserBanned
		}

		if user.TeamID == nil {
			return apperr.ErrTeamNotFound
		}

		if err := uc.deps.TeamRepo.Lock(txCtx, *user.TeamID); err != nil {
			return fmt.Errorf("TeamUseCase - RegenerateInviteToken - TeamRepo.Lock: %w", err)
		}

		var errTeam error

		team, errTeam = uc.deps.TeamRepo.GetByID(txCtx, *user.TeamID)
		if errTeam != nil {
			return fmt.Errorf("TeamUseCase - RegenerateInviteToken - TeamRepo.GetByID: %w", errTeam)
		}

		if team.CaptainID != captainID {
			return apperr.ErrNotCaptain
		}

		if team.IsSolo {
			return apperr.ErrTeamNotFound
		}

		if team.IsBanned {
			return apperr.ErrTeamBanned
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
