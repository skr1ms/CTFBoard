package team

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/scoring"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

func (uc *TeamUseCase) GetByID(ctx context.Context, ID uuid.UUID) (*domain.Team, error) {
	team, err := uc.deps.TeamRepo.GetByID(ctx, ID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - GetByID - TeamRepo.GetByID: %w", err)
	}

	return team, nil
}

func (uc *TeamUseCase) GetProfile(ctx context.Context, ID uuid.UUID) (*usecase.TeamProfile, error) {
	var (
		team        *domain.Team
		fields      []*domain.Field
		fieldValues []*domain.FieldValue
	)

	g, gCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		var err error

		team, err = uc.deps.TeamRepo.GetByID(gCtx, ID)
		if err != nil {
			return fmt.Errorf("TeamUseCase - GetProfile - TeamRepo.GetByID: %w", err)
		}

		return nil
	})

	if uc.deps.FieldRepo != nil && uc.deps.FieldValueRepo != nil {
		g.Go(func() error {
			var err error

			fields, err = uc.deps.FieldRepo.GetByEntityType(gCtx, domain.EntityTypeTeam)
			if err != nil {
				return fmt.Errorf("TeamUseCase - GetProfile - FieldRepo.GetByEntityType: %w", err)
			}

			return nil
		})
		g.Go(func() error {
			var err error

			fieldValues, err = uc.deps.FieldValueRepo.GetByEntityID(gCtx, ID)
			if err != nil {
				return fmt.Errorf("TeamUseCase - GetProfile - FieldValueRepo.GetByEntityID: %w", err)
			}

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("TeamUseCase - GetProfile - errgroup.Wait: %w", err)
	}

	return &usecase.TeamProfile{
		Team:         team,
		CustomFields: visibleTeamFieldValuesToMap(fields, fieldValues, publicTeamField),
	}, nil
}

// GetMyTeam returns the caller's team, members, eligibility metadata, and self-visible custom fields.
func (uc *TeamUseCase) GetMyTeam(ctx context.Context, userID uuid.UUID) (*usecase.TeamMe, error) {
	user, err := uc.deps.UserRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - GetMyTeam - UserRepo.GetByID: %w", err)
	}

	if user.TeamID == nil {
		return nil, apperr.ErrUserNotInTeam
	}

	teamID := *user.TeamID

	var (
		team        *domain.Team
		members     []*domain.User
		fields      []*domain.Field
		fieldValues []*domain.FieldValue
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

	if uc.deps.FieldRepo != nil && uc.deps.FieldValueRepo != nil {
		g.Go(func() error {
			var err error

			fields, err = uc.deps.FieldRepo.GetByEntityType(gCtx, domain.EntityTypeTeam)
			if err != nil {
				return fmt.Errorf("TeamUseCase - GetMyTeam - FieldRepo.GetByEntityType: %w", err)
			}

			return nil
		})
		g.Go(func() error {
			var err error

			fieldValues, err = uc.deps.FieldValueRepo.GetByEntityID(gCtx, teamID)
			if err != nil {
				return fmt.Errorf("TeamUseCase - GetMyTeam - FieldValueRepo.GetByEntityID: %w", err)
			}

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("TeamUseCase - GetMyTeam - errgroup.Wait: %w", err)
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

	return &usecase.TeamMe{
		Team:         team,
		Members:      members,
		MinTeamSize:  minTeamSize,
		MeetsMinSize: meetsMinSize,
		CustomFields: visibleTeamFieldValuesToMap(fields, fieldValues, selfTeamField),
	}, nil
}

func visibleTeamFieldValuesToMap(
	fields []*domain.Field,
	values []*domain.FieldValue,
	include func(*domain.Field) bool,
) usecase.CustomFieldValues {
	return usecase.CustomFieldStorageValuesToMap(fields, values, include)
}

func publicTeamField(field *domain.Field) bool {
	return field.Public
}

func selfTeamField(field *domain.Field) bool {
	return field.Public || field.Editable
}

func (uc *TeamUseCase) GetTeamMembers(ctx context.Context, teamID uuid.UUID) ([]*domain.User, error) {
	users, err := uc.deps.UserRepo.GetByTeamID(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - GetTeamMembers - UserRepo.GetByTeamID: %w", err)
	}

	return users, nil
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
