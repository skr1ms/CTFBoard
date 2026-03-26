package persistent

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wahrwelt-kit/go-pgkit/pgutil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

type AwardRepo struct {
	BaseRepo
}

var _ repo.AwardRepository = (*AwardRepo)(nil)

func NewAwardRepo(pool *pgxpool.Pool) *AwardRepo {
	return &AwardRepo{BaseRepo: BaseRepo{pool: pool}}
}

func toDomainAwardFromRow(id, teamID uuid.UUID, value int32, description string, createdBy *uuid.UUID, createdAt pgtype.Timestamptz) *domain.Award {
	return &domain.Award{
		ID:          id,
		TeamID:      teamID,
		Value:       int(value),
		Description: description,
		CreatedBy:   createdBy,
		CreatedAt:   pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(createdAt)),
	}
}

func toDomainAwardFromBackup(a sqlc.Award) *domain.Award {
	return &domain.Award{
		ID:           a.ID,
		TeamID:       a.TeamID,
		Value:        int(a.Value),
		Description:  a.Description,
		CreatedBy:    a.CreatedBy,
		CreatedAt:    pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(a.CreatedAt)),
		BannedTeamID: a.BannedTeamID,
	}
}

func (r *AwardRepo) GetByTeamID(ctx context.Context, teamID uuid.UUID) ([]*domain.Award, error) {
	rows, err := r.Q(ctx).GetAwardsByTeamID(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("AwardRepo - GetByTeamID: %w", err)
	}

	out := make([]*domain.Award, 0, len(rows))
	for _, a := range rows {
		out = append(out, toDomainAwardFromRow(a.ID, a.TeamID, a.Value, a.Description, a.CreatedBy, a.CreatedAt))
	}

	return out, nil
}

func (r *AwardRepo) GetTeamTotalAwards(ctx context.Context, teamID uuid.UUID) (int, error) {
	total, err := r.Q(ctx).GetTeamTotalAwards(ctx, teamID)
	if err != nil {
		return 0, fmt.Errorf("AwardRepo - GetTeamTotalAwards: %w", err)
	}

	return int(total), nil
}

func (r *AwardRepo) GetAll(ctx context.Context) ([]*domain.Award, error) {
	rows, err := r.Q(ctx).GetAllAwards(ctx)
	if err != nil {
		return nil, fmt.Errorf("AwardRepo - GetAll: %w", err)
	}

	out := make([]*domain.Award, 0, len(rows))
	for _, a := range rows {
		out = append(out, toDomainAwardFromRow(a.ID, a.TeamID, a.Value, a.Description, a.CreatedBy, a.CreatedAt))
	}

	return out, nil
}

func (r *AwardRepo) GetAllForBackup(ctx context.Context) ([]*domain.Award, error) {
	rows, err := r.Q(ctx).GetAwardsForBackup(ctx)
	if err != nil {
		return nil, fmt.Errorf("AwardRepo - GetAllForBackup: %w", err)
	}

	out := make([]*domain.Award, 0, len(rows))
	for _, a := range rows {
		out = append(out, toDomainAwardFromBackup(a))
	}

	return out, nil
}

func (r *AwardRepo) GetByID(ctx context.Context, ID uuid.UUID) (*domain.Award, error) {
	a, err := r.Q(ctx).GetAwardByID(ctx, ID)
	if err != nil {
		if pgutil.IsNoRows(err) {
			return nil, httperr.ErrAwardNotFound
		}

		return nil, fmt.Errorf("AwardRepo - GetByID: %w", err)
	}

	return toDomainAwardFromRow(a.ID, a.TeamID, a.Value, a.Description, a.CreatedBy, a.CreatedAt), nil
}

func (r *AwardRepo) Delete(ctx context.Context, ID uuid.UUID) error {
	err := r.Q(ctx).DeleteAward(ctx, ID)
	if err != nil {
		return fmt.Errorf("AwardRepo - Delete: %w", err)
	}

	return nil
}

func (r *AwardRepo) DeleteByTeamID(ctx context.Context, teamID uuid.UUID) error {
	err := r.Q(ctx).DeleteAwardsByTeamID(ctx, teamID)
	if err != nil {
		return fmt.Errorf("AwardRepo - DeleteByTeamID: %w", err)
	}

	return nil
}

func (r *AwardRepo) SoftBanByTeamID(ctx context.Context, teamID uuid.UUID) error {
	err := r.Q(ctx).SoftBanAwardsByTeamID(ctx, teamID)
	if err != nil {
		return fmt.Errorf("AwardRepo - SoftBanByTeamID: %w", err)
	}

	return nil
}

func (r *AwardRepo) RestoreByBannedTeamID(ctx context.Context, teamID uuid.UUID) error {
	err := r.Q(ctx).RestoreAwardsByBannedTeamID(ctx, &teamID)
	if err != nil {
		return fmt.Errorf("AwardRepo - RestoreByBannedTeamID: %w", err)
	}

	return nil
}

func (r *AwardRepo) Create(ctx context.Context, a *domain.Award) error {
	a.ID = uuid.New()
	a.CreatedAt = time.Now()

	value, err := intToInt32Safe(a.Value)
	if err != nil {
		return fmt.Errorf("AwardRepo - Create: %w", err)
	}

	err = r.Q(ctx).CreateAward(ctx, sqlc.CreateAwardParams{
		ID:          a.ID,
		TeamID:      a.TeamID,
		Value:       value,
		Description: a.Description,
		CreatedBy:   a.CreatedBy,
		CreatedAt:   pgutil.TimeToTimestamptz(&a.CreatedAt),
	})
	if err != nil {
		return fmt.Errorf("AwardRepo - Create: %w", err)
	}

	return nil
}
