package persistent

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

type CompetitionRepo struct {
	pool *pgxpool.Pool
}

var _ repo.CompetitionRepository = (*CompetitionRepo)(nil)

func NewCompetitionRepo(pool *pgxpool.Pool) *CompetitionRepo {
	return &CompetitionRepo{pool: pool}
}

func (r *CompetitionRepo) q(ctx context.Context) *sqlc.Queries {
	return sqlc.New(ExtractDB(ctx, r.pool))
}

func toEntityCompetition(c sqlc.Competition) *entity.Competition {
	return &entity.Competition{
		ID:                           int(c.ID),
		Name:                         c.Name,
		StartTime:                    timestamptzToTime(c.StartTime),
		EndTime:                      timestamptzToTime(c.EndTime),
		FreezeTime:                   timestamptzToTime(c.FreezeTime),
		IsPaused:                     boolPtrToBool(c.IsPaused),
		PausedAt:                     timestamptzToTime(c.PausedAt),
		IsPublic:                     boolPtrToBool(c.IsPublic),
		FlagRegex:                    c.FlagRegex,
		Mode:                         entity.CompetitionMode(ptrStrToStr(c.Mode)),
		AllowTeamSwitch:              boolPtrToBool(c.AllowTeamSwitch),
		MinTeamSize:                  int32PtrToInt(c.MinTeamSize),
		MaxTeamSize:                  int32PtrToInt(c.MaxTeamSize),
		KeepScoreboardFrozenAfterEnd: c.KeepScoreboardFrozenAfterEnd,
		CreatedAt:                    ptrTimeToTime(timestamptzToTime(c.CreatedAt)),
		UpdatedAt:                    ptrTimeToTime(timestamptzToTime(c.UpdatedAt)),
	}
}

func (r *CompetitionRepo) Get(ctx context.Context) (*entity.Competition, error) {
	c, err := r.q(ctx).GetCompetition(ctx)
	if err != nil {
		if isNoRows(err) {
			return nil, httperr.ErrCompetitionNotFound
		}
		return nil, fmt.Errorf("CompetitionRepo - Get: %w", err)
	}
	return toEntityCompetition(c), nil
}

func (r *CompetitionRepo) GetForUpdate(ctx context.Context) (*entity.Competition, error) {
	c, err := r.q(ctx).GetCompetitionForUpdate(ctx)
	if err != nil {
		if isNoRows(err) {
			return nil, httperr.ErrCompetitionNotFound
		}
		return nil, fmt.Errorf("CompetitionRepo - GetForUpdate: %w", err)
	}
	return toEntityCompetition(c), nil
}

func (r *CompetitionRepo) Update(ctx context.Context, c *entity.Competition) error {
	minTeamSize, err := intToInt32Safe(c.MinTeamSize)
	if err != nil {
		return fmt.Errorf("CompetitionRepo - Update - MinTeamSize: %w", err)
	}
	maxTeamSize, err := intToInt32Safe(c.MaxTeamSize)
	if err != nil {
		return fmt.Errorf("CompetitionRepo - Update - MaxTeamSize: %w", err)
	}
	err = r.q(ctx).UpdateCompetition(ctx, sqlc.UpdateCompetitionParams{
		Name:                         c.Name,
		StartTime:                    timeToTimestamptz(c.StartTime),
		EndTime:                      timeToTimestamptz(c.EndTime),
		FreezeTime:                   timeToTimestamptz(c.FreezeTime),
		IsPaused:                     &c.IsPaused,
		PausedAt:                     timeToTimestamptz(c.PausedAt),
		IsPublic:                     &c.IsPublic,
		FlagRegex:                    c.FlagRegex,
		Mode:                         func() *string { s := string(c.Mode); return &s }(),
		AllowTeamSwitch:              &c.AllowTeamSwitch,
		MinTeamSize:                  &minTeamSize,
		MaxTeamSize:                  &maxTeamSize,
		KeepScoreboardFrozenAfterEnd: c.KeepScoreboardFrozenAfterEnd,
	})
	if err != nil {
		return fmt.Errorf("CompetitionRepo - Update: %w", err)
	}
	return nil
}
