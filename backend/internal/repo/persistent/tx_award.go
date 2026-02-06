package persistent

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/repo/persistent/sqlc"
)

type TxAwardRepo struct {
	base *TxBase
}

func (r *TxAwardRepo) CreateAwardTx(ctx context.Context, tx pgx.Tx, a *entity.Award) error {
	a.ID = uuid.New()
	a.CreatedAt = time.Now()
	value, err := intToInt32Safe(a.Value)
	if err != nil {
		return fmt.Errorf("TxAwardRepo - CreateAwardTx: %w", err)
	}
	err = r.base.q.WithTx(tx).CreateAward(ctx, sqlc.CreateAwardParams{
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
