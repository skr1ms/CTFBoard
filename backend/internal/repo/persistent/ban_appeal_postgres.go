package persistent

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wahrwelt-kit/go-pgkit/pgutil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
)

type BanAppealRepo struct {
	BaseRepo
}

var _ repo.BanAppealRepository = (*BanAppealRepo)(nil)

func NewBanAppealRepo(pool *pgxpool.Pool) *BanAppealRepo {
	return &BanAppealRepo{BaseRepo: BaseRepo{pool: pool}}
}

func (r *BanAppealRepo) Create(ctx context.Context, a *domain.BanAppeal) error {
	row, err := r.Q(ctx).CreateBanAppeal(ctx, sqlc.CreateBanAppealParams{
		UserID:  a.UserID,
		Message: a.Message,
	})
	if err != nil {
		return fmt.Errorf("BanAppealRepo - Create: %w", err)
	}

	a.ID = row.ID
	a.Decision = domain.AppealDecision(row.Decision)
	a.CreatedAt = pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(row.CreatedAt))

	return nil
}

func (r *BanAppealRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.BanAppeal, error) {
	return GetOrNotFound(func() (*domain.BanAppeal, error) {
		row, err := r.Q(ctx).GetBanAppealByID(ctx, id)
		if err != nil {
			return nil, err
		}

		return banAppealFromRow(row), nil
	}, apperr.ErrAppealNotFound, "BanAppealRepo - GetByID")
}

func (r *BanAppealRepo) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.BanAppeal, error) {
	rows, err := r.Q(ctx).GetBanAppealsByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("BanAppealRepo - GetByUserID: %w", err)
	}

	return banAppealsFromRows(rows), nil
}

func (r *BanAppealRepo) GetLatestByUserID(ctx context.Context, userID uuid.UUID) (*domain.BanAppeal, error) {
	row, err := r.Q(ctx).GetLatestBanAppealByUserID(ctx, userID)
	if err != nil {
		if pgutil.IsNoRows(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("BanAppealRepo - GetLatestByUserID: %w", err)
	}

	return banAppealFromRow(row), nil
}

func (r *BanAppealRepo) List(ctx context.Context, decision *domain.AppealDecision, limit, offset int) ([]*domain.BanAppeal, int64, error) {
	limit32, offset32, err := toLimitOffset(limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("BanAppealRepo - List: %w", err)
	}

	var (
		rows  []sqlc.BanAppeal
		total int64
	)

	if decision != nil {
		dec := string(*decision)

		rows, err = r.Q(ctx).ListBanAppealsByDecision(ctx, sqlc.ListBanAppealsByDecisionParams{
			Decision: dec,
			Limit:    limit32,
			Offset:   offset32,
		})
		if err != nil {
			return nil, 0, fmt.Errorf("BanAppealRepo - List - Query: %w", err)
		}

		total, err = r.Q(ctx).CountBanAppealsByDecision(ctx, dec)
	} else {
		rows, err = r.Q(ctx).ListBanAppeals(ctx, sqlc.ListBanAppealsParams{
			Limit:  limit32,
			Offset: offset32,
		})
		if err != nil {
			return nil, 0, fmt.Errorf("BanAppealRepo - List - Query: %w", err)
		}

		total, err = r.Q(ctx).CountBanAppeals(ctx)
	}

	if err != nil {
		return nil, 0, fmt.Errorf("BanAppealRepo - List - Count: %w", err)
	}

	return banAppealsFromRows(rows), total, nil
}

func (r *BanAppealRepo) Update(ctx context.Context, a *domain.BanAppeal) error {
	now := time.Now().UTC()
	a.ReviewedAt = &now

	n, err := r.Q(ctx).UpdateBanAppeal(ctx, sqlc.UpdateBanAppealParams{
		ID:            a.ID,
		Decision:      string(a.Decision),
		AdminResponse: a.AdminResponse,
		ReviewedAt:    pgutil.TimeToTimestamptz(&now),
	})
	if err != nil {
		return fmt.Errorf("BanAppealRepo - Update: %w", err)
	}

	if n == 0 {
		return apperr.ErrAppealNotFound
	}

	return nil
}

func banAppealFromRow(row sqlc.BanAppeal) *domain.BanAppeal {
	a := &domain.BanAppeal{
		ID:            row.ID,
		UserID:        row.UserID,
		Message:       row.Message,
		AdminResponse: row.AdminResponse,
		Decision:      domain.AppealDecision(row.Decision),
		CreatedAt:     pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(row.CreatedAt)),
		ReviewedAt:    pgutil.TimestamptzToTime(row.ReviewedAt),
	}

	return a
}

func banAppealsFromRows(rows []sqlc.BanAppeal) []*domain.BanAppeal {
	appeals := make([]*domain.BanAppeal, len(rows))
	for i, row := range rows {
		appeals[i] = banAppealFromRow(row)
	}

	return appeals
}
