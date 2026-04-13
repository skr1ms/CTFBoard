package persistent

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wahrwelt-kit/go-pgkit/pgutil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
)

type CompetitionRepo struct {
	BaseRepo
}

var _ repo.CompetitionRepository = (*CompetitionRepo)(nil)

func NewCompetitionRepo(pool *pgxpool.Pool) *CompetitionRepo {
	return &CompetitionRepo{BaseRepo: BaseRepo{pool: pool}}
}

func toDomainCompetition(c sqlc.Competition) *domain.Competition {
	return &domain.Competition{
		ID:                           int(c.ID),
		Name:                         c.Name,
		StartTime:                    pgutil.TimestamptzToTime(c.StartTime),
		EndTime:                      pgutil.TimestamptzToTime(c.EndTime),
		FreezeTime:                   pgutil.TimestamptzToTime(c.FreezeTime),
		IsPaused:                     c.IsPaused,
		PausedAt:                     pgutil.TimestamptzToTime(c.PausedAt),
		IsPublic:                     c.IsPublic,
		FlagRegex:                    c.FlagRegex,
		Mode:                         domain.CompetitionMode(c.Mode),
		AllowTeamSwitch:              c.AllowTeamSwitch,
		MinTeamSize:                  int(c.MinTeamSize),
		MaxTeamSize:                  int(c.MaxTeamSize),
		KeepScoreboardFrozenAfterEnd: c.KeepScoreboardFrozenAfterEnd,
		CreatedAt:                    pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(c.CreatedAt)),
		UpdatedAt:                    pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(c.UpdatedAt)),
	}
}

func (r *CompetitionRepo) Get(ctx context.Context) (*domain.Competition, error) {
	c, err := GetOrNotFound(func() (sqlc.Competition, error) { return r.Q(ctx).GetCompetition(ctx) },
		apperr.ErrCompetitionNotFound, "CompetitionRepo - Get")
	if err != nil {
		return nil, err
	}

	return toDomainCompetition(c), nil
}

func (r *CompetitionRepo) GetForUpdate(ctx context.Context) (*domain.Competition, error) {
	c, err := GetOrNotFound(func() (sqlc.Competition, error) { return r.Q(ctx).GetCompetitionForUpdate(ctx) },
		apperr.ErrCompetitionNotFound, "CompetitionRepo - GetForUpdate")
	if err != nil {
		return nil, err
	}

	return toDomainCompetition(c), nil
}

func (r *CompetitionRepo) Update(ctx context.Context, c *domain.Competition) error {
	minTeamSize, err := intToInt32Safe(c.MinTeamSize)
	if err != nil {
		return fmt.Errorf("CompetitionRepo - Update - MinTeamSize: %w", err)
	}

	maxTeamSize, err := intToInt32Safe(c.MaxTeamSize)
	if err != nil {
		return fmt.Errorf("CompetitionRepo - Update - MaxTeamSize: %w", err)
	}

	now := time.Now()

	err = r.Q(ctx).UpdateCompetition(ctx, sqlc.UpdateCompetitionParams{
		Name:                         c.Name,
		StartTime:                    pgutil.TimeToTimestamptz(c.StartTime),
		EndTime:                      pgutil.TimeToTimestamptz(c.EndTime),
		FreezeTime:                   pgutil.TimeToTimestamptz(c.FreezeTime),
		IsPaused:                     c.IsPaused,
		PausedAt:                     pgutil.TimeToTimestamptz(c.PausedAt),
		IsPublic:                     c.IsPublic,
		FlagRegex:                    c.FlagRegex,
		Mode:                         string(c.Mode),
		AllowTeamSwitch:              c.AllowTeamSwitch,
		MinTeamSize:                  minTeamSize,
		MaxTeamSize:                  maxTeamSize,
		KeepScoreboardFrozenAfterEnd: c.KeepScoreboardFrozenAfterEnd,
		UpdatedAt:                    pgutil.TimeToTimestamptz(&now),
	})
	if err != nil {
		return fmt.Errorf("CompetitionRepo - Update: %w", err)
	}

	return nil
}
