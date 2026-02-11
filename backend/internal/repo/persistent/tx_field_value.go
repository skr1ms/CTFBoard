package persistent

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/repo"
	"github.com/skr1ms/CTFBoard/internal/repo/persistent/sqlc"
)

type TxFieldValueRepo struct {
	base *TxBase
}

func (r *TxFieldValueRepo) SetFieldValuesTx(ctx context.Context, tx repo.Transaction, entityID uuid.UUID, values map[string]string) error {
	if len(values) == 0 {
		return nil
	}
	pgxTx := mustPgxTx(tx)
	q := r.base.q.WithTx(pgxTx)
	if err := q.DeleteFieldValuesByEntityID(ctx, entityID); err != nil {
		return fmt.Errorf("TxFieldValueRepo - SetFieldValuesTx Delete: %w", err)
	}
	for fieldIDStr, value := range values {
		fieldID, err := uuid.Parse(fieldIDStr)
		if err != nil {
			return fmt.Errorf("TxFieldValueRepo - SetFieldValuesTx invalid field_id %s: %w", fieldIDStr, err)
		}
		now := time.Now()
		if err := q.UpsertFieldValue(ctx, sqlc.UpsertFieldValueParams{
			ID:        uuid.New(),
			FieldID:   fieldID,
			EntityID:  entityID,
			Value:     value,
			CreatedAt: &now,
		}); err != nil {
			return fmt.Errorf("TxFieldValueRepo - SetFieldValuesTx Upsert: %w", err)
		}
	}
	return nil
}
