package persistent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/samber/lo"
	"github.com/wahrwelt-kit/go-pgkit/pgutil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
)

type AuditLogRepo struct {
	BaseRepo
}

var _ repo.AuditLogRepository = (*AuditLogRepo)(nil)

func NewAuditLogRepo(pool *pgxpool.Pool) *AuditLogRepo {
	return &AuditLogRepo{BaseRepo: BaseRepo{pool: pool}}
}

func (r *AuditLogRepo) Create(ctx context.Context, l *domain.AuditLog) error {
	details, err := json.Marshal(l.Details)
	if err != nil {
		return fmt.Errorf("AuditLogRepo - Create - Marshal: %w", err)
	}

	row, err := r.Q(ctx).CreateAuditLog(ctx, sqlc.CreateAuditLogParams{
		UserID:     l.UserID,
		Action:     string(l.Action),
		EntityType: string(l.EntityType),
		EntityID:   lo.EmptyableToPtr(l.EntityID),
		IP:         lo.EmptyableToPtr(l.IP),
		Details:    details,
	})
	if err != nil {
		return fmt.Errorf("AuditLogRepo - Create: %w", err)
	}

	l.ID = row.ID
	l.CreatedAt = pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(row.CreatedAt))

	return nil
}
