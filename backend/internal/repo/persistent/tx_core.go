package persistent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/repo"
	"github.com/skr1ms/CTFBoard/internal/repo/persistent/sqlc"
)

type TxAuditRepo struct {
	base *TxBase
}

func (r *TxAuditRepo) CreateAuditLogTx(ctx context.Context, tx repo.Transaction, log *entity.AuditLog) error {
	pgxTx := mustPgxTx(tx)
	details, err := json.Marshal(log.Details)
	if err != nil {
		return fmt.Errorf("TxAuditRepo - CreateAuditLogTx Marshal: %w", err)
	}
	row, err := r.base.q.WithTx(pgxTx).CreateAuditLog(ctx, sqlc.CreateAuditLogParams{
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

type TxAwardRepo struct {
	base *TxBase
}

func (r *TxAwardRepo) CreateAwardTx(ctx context.Context, tx repo.Transaction, a *entity.Award) error {
	pgxTx := mustPgxTx(tx)
	a.ID = uuid.New()
	a.CreatedAt = time.Now()
	value, err := intToInt32Safe(a.Value)
	if err != nil {
		return fmt.Errorf("TxAwardRepo - CreateAwardTx: %w", err)
	}
	err = r.base.q.WithTx(pgxTx).CreateAward(ctx, sqlc.CreateAwardParams{
		ID:          a.ID,
		TeamID:      a.TeamID,
		Value:       value,
		Description: a.Description,
		CreatedBy:   a.CreatedBy,
		CreatedAt:   &a.CreatedAt,
	})
	if err != nil {
		return fmt.Errorf("TxAwardRepo - CreateAwardTx: %w", err)
	}
	return nil
}
