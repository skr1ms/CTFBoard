package persistent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/repo/persistent/sqlc"
)

type TxAuditRepo struct {
	base *TxBase
}

func (r *TxAuditRepo) CreateAuditLogTx(ctx context.Context, tx pgx.Tx, log *entity.AuditLog) error {
	details, err := json.Marshal(log.Details)
	if err != nil {
		return fmt.Errorf("TxAuditRepo - CreateAuditLogTx Marshal: %w", err)
	}
	row, err := r.base.q.WithTx(tx).CreateAuditLog(ctx, sqlc.CreateAuditLogParams{
		UserID:     log.UserID,
		Action:     string(log.Action),
		EntityType: string(log.EntityType),
		EntityID:   strPtrOrNil(log.EntityID),
		Ip:         strPtrOrNil(log.IP),
		Details:    details,
	})
	if err != nil {
		return fmt.Errorf("TxAuditRepo - CreateAuditLogTx: %w", err)
	}
	log.ID = row.ID
	log.CreatedAt = ptrTimeToTime(row.CreatedAt)
	return nil
}
