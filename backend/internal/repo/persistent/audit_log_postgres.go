package persistent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuditLogRepo struct {
	pool *pgxpool.Pool
}

var _ repo.AuditLogRepository = (*AuditLogRepo)(nil)

func NewAuditLogRepo(pool *pgxpool.Pool) *AuditLogRepo {
	return &AuditLogRepo{pool: pool}
}

func (r *AuditLogRepo) q(ctx context.Context) *sqlc.Queries {
	return sqlc.New(ExtractDB(ctx, r.pool))
}

func (r *AuditLogRepo) Create(ctx context.Context, l *entity.AuditLog) error {
	details, err := json.Marshal(l.Details)
	if err != nil {
		return fmt.Errorf("AuditLogRepo - Create - Marshal: %w", err)
	}
	row, err := r.q(ctx).CreateAuditLog(ctx, sqlc.CreateAuditLogParams{
		UserID:     l.UserID,
		Action:     string(l.Action),
		EntityType: string(l.EntityType),
		EntityID:   strPtrOrNil(l.EntityID),
		IP:         strPtrOrNil(l.IP),
		Details:    details,
	})
	if err != nil {
		return fmt.Errorf("AuditLogRepo - Create: %w", err)
	}
	l.ID = row.ID
	l.CreatedAt = ptrTimeToTime(row.CreatedAt)
	return nil
}
