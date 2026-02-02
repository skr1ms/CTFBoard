package persistent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/repo/persistent/sqlc"
)

type AuditLogRepo struct {
	db *pgxpool.Pool
	q  *sqlc.Queries
}

func NewAuditLogRepo(db *pgxpool.Pool) *AuditLogRepo {
	return &AuditLogRepo{db: db, q: sqlc.New(db)}
}

func (r *AuditLogRepo) Create(ctx context.Context, l *entity.AuditLog) error {
	details, err := json.Marshal(l.Details)
	if err != nil {
		return fmt.Errorf("AuditLogRepo - Create Marshal: %w", err)
	}
	row, err := r.q.CreateAuditLog(ctx, sqlc.CreateAuditLogParams{
		UserID:     l.UserID,
		Action:     string(l.Action),
		EntityType: string(l.EntityType),
		EntityID:   strPtrOrNil(l.EntityID),
		Ip:         strPtrOrNil(l.IP),
		Details:    details,
	})
	if err != nil {
		return fmt.Errorf("AuditLogRepo - Create: %w", err)
	}
	l.ID = row.ID
	l.CreatedAt = ptrTimeToTime(row.CreatedAt)
	return nil
}
