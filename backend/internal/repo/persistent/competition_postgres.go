package persistent

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/repo/persistent/sqlc"
)

type CompetitionRepo struct {
	db *pgxpool.Pool
	q  *sqlc.Queries
}

func NewCompetitionRepo(db *pgxpool.Pool) *CompetitionRepo {
	return &CompetitionRepo{db: db, q: sqlc.New(db)}
}

func toEntityCompetition(c sqlc.Competition) *entity.Competition {
	return &entity.Competition{
		ID:              int(c.ID),
		Name:            c.Name,
		StartTime:       c.StartTime,
		EndTime:         c.EndTime,
		FreezeTime:      c.FreezeTime,
		IsPaused:        boolPtrToBool(c.IsPaused),
		IsPublic:        boolPtrToBool(c.IsPublic),
		FlagRegex:       c.FlagRegex,
		Mode:            ptrStrToStr(c.Mode),
		AllowTeamSwitch: boolPtrToBool(c.AllowTeamSwitch),
		MinTeamSize:     int32PtrToInt(c.MinTeamSize),
		MaxTeamSize:     int32PtrToInt(c.MaxTeamSize),
		CreatedAt:       ptrTimeToTime(c.CreatedAt),
		UpdatedAt:       ptrTimeToTime(c.UpdatedAt),
	}
}

func (r *CompetitionRepo) Get(ctx context.Context) (*entity.Competition, error) {
	c, err := r.q.GetCompetition(ctx)
	if err != nil {
		if isNoRows(err) {
			return nil, entityError.ErrCompetitionNotFound
		}
		return nil, fmt.Errorf("CompetitionRepo - Get: %w", err)
	}
	return toEntityCompetition(c), nil
}

func (r *CompetitionRepo) Update(ctx context.Context, c *entity.Competition) error {
	minTeamSize, err := intToInt32Safe(c.MinTeamSize)
	if err != nil {
		return fmt.Errorf("CompetitionRepo - Update MinTeamSize: %w", err)
	}
	maxTeamSize, err := intToInt32Safe(c.MaxTeamSize)
	if err != nil {
		return fmt.Errorf("CompetitionRepo - Update MaxTeamSize: %w", err)
	}
	updatedAt := time.Now()
	err = r.q.UpdateCompetition(ctx, sqlc.UpdateCompetitionParams{
		Name:            c.Name,
		StartTime:       c.StartTime,
		EndTime:         c.EndTime,
		FreezeTime:      c.FreezeTime,
		IsPaused:        &c.IsPaused,
		IsPublic:        &c.IsPublic,
		FlagRegex:       c.FlagRegex,
		Mode:            &c.Mode,
		AllowTeamSwitch: &c.AllowTeamSwitch,
		MinTeamSize:     &minTeamSize,
		MaxTeamSize:     &maxTeamSize,
		UpdatedAt:       &updatedAt,
	})
	if err != nil {
		return fmt.Errorf("CompetitionRepo - Update: %w", err)
	}
	return nil
}
