package persistent

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/repo/persistent/sqlc"
)

type TxHintRepo struct {
	base *TxBase
}

func (r *TxHintRepo) CreateHintUnlockTx(ctx context.Context, tx pgx.Tx, teamID, hintID uuid.UUID) error {
	err := r.base.q.WithTx(tx).CreateHintUnlock(ctx, sqlc.CreateHintUnlockParams{
		ID:     uuid.New(),
		HintID: hintID,
		TeamID: teamID,
	})
	if err != nil {
		return fmt.Errorf("TxHintRepo - CreateHintUnlockTx: %w", err)
	}
	return nil
}

func (r *TxHintRepo) GetHintUnlockByTeamAndHintTx(ctx context.Context, tx pgx.Tx, teamID, hintID uuid.UUID) (*entity.HintUnlock, error) {
	u, err := r.base.q.WithTx(tx).GetHintUnlockByTeamAndHintForUpdate(ctx, sqlc.GetHintUnlockByTeamAndHintForUpdateParams{
		TeamID: teamID,
		HintID: hintID,
	})
	if err != nil {
		if isNoRows(err) {
			return nil, entityError.ErrHintNotFound
		}
		return nil, fmt.Errorf("TxHintRepo - GetHintUnlockByTeamAndHintTx: %w", err)
	}
	return toEntityHintUnlock(u), nil
}
