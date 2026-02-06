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

type TxChallengeRepo struct {
	base *TxBase
}

func (r *TxChallengeRepo) GetChallengeByIDTx(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*entity.Challenge, error) {
	row, err := r.base.q.WithTx(tx).GetChallengeByIDForUpdate(ctx, id)
	if err != nil {
		if isNoRows(err) {
			return nil, entityError.ErrChallengeNotFound
		}
		return nil, fmt.Errorf("TxChallengeRepo - GetChallengeByIDTx: %w", err)
	}
	return toEntityChallengeFromRow(row.ID, row.Title, row.Description, row.Category, row.Points, row.InitialValue, row.MinValue, row.Decay, row.SolveCount, row.FlagHash, row.IsHidden, row.IsRegex, row.IsCaseInsensitive, row.FlagRegex, row.FlagFormatRegex), nil
}

func (r *TxChallengeRepo) IncrementChallengeSolveCountTx(ctx context.Context, tx pgx.Tx, id uuid.UUID) (int, error) {
	n, err := r.base.q.WithTx(tx).IncrementChallengeSolveCount(ctx, id)
	if err != nil {
		return 0, fmt.Errorf("TxChallengeRepo - IncrementChallengeSolveCountTx: %w", err)
	}
	return int(n), nil
}

func (r *TxChallengeRepo) UpdateChallengePointsTx(ctx context.Context, tx pgx.Tx, id uuid.UUID, points int) error {
	pts, err := intToInt32Safe(points)
	if err != nil {
		return fmt.Errorf("TxChallengeRepo - UpdateChallengePointsTx: %w", err)
	}
	_, err = r.base.q.WithTx(tx).UpdateChallengePoints(ctx, sqlc.UpdateChallengePointsParams{ID: id, Points: &pts})
	if err != nil {
		if isNoRows(err) {
			return entityError.ErrChallengeNotFound
		}
		return fmt.Errorf("TxChallengeRepo - UpdateChallengePointsTx: %w", err)
	}
	return nil
}

func (r *TxChallengeRepo) DeleteChallengeTx(ctx context.Context, tx pgx.Tx, challengeID uuid.UUID) error {
	_, err := r.base.q.WithTx(tx).DeleteChallenge(ctx, challengeID)
	if err != nil {
		if isNoRows(err) {
			return entityError.ErrChallengeNotFound
		}
		return fmt.Errorf("TxChallengeRepo - DeleteChallengeTx: %w", err)
	}
	return nil
}
