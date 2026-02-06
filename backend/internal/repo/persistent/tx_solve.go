package persistent

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/repo/persistent/sqlc"
)

type TxSolveRepo struct {
	base *TxBase
}

func (r *TxSolveRepo) CreateSolveTx(ctx context.Context, tx pgx.Tx, s *entity.Solve) error {
	s.ID = uuid.New()
	s.SolvedAt = time.Now()
	err := r.base.q.WithTx(tx).CreateSolve(ctx, sqlc.CreateSolveParams{
		ID:          s.ID,
		UserID:      s.UserID,
		TeamID:      s.TeamID,
		ChallengeID: s.ChallengeID,
		SolvedAt:    &s.SolvedAt,
	})
	if err != nil {
		if isPgUniqueViolation(err) {
			return entityError.ErrAlreadySolved
		}
		return fmt.Errorf("TxSolveRepo - CreateSolveTx: %w", err)
	}
	return nil
}

func (r *TxSolveRepo) DeleteSolvesByTeamIDTx(ctx context.Context, tx pgx.Tx, teamID uuid.UUID) error {
	if err := r.base.q.WithTx(tx).DeleteSolvesByTeamID(ctx, teamID); err != nil {
		return fmt.Errorf("TxSolveRepo - DeleteSolvesByTeamIDTx: %w", err)
	}
	return nil
}

func (r *TxSolveRepo) GetSolveByTeamAndChallengeTx(ctx context.Context, tx pgx.Tx, teamID, challengeID uuid.UUID) (*entity.Solve, error) {
	s, err := r.base.q.WithTx(tx).GetSolveByTeamAndChallengeForUpdate(ctx, sqlc.GetSolveByTeamAndChallengeForUpdateParams{
		TeamID:      teamID,
		ChallengeID: challengeID,
	})
	if err != nil {
		if isNoRows(err) {
			return nil, entityError.ErrSolveNotFound
		}
		return nil, fmt.Errorf("TxSolveRepo - GetSolveByTeamAndChallengeTx: %w", err)
	}
	return toEntitySolve(s), nil
}

func (r *TxSolveRepo) GetTeamScoreTx(ctx context.Context, tx pgx.Tx, teamID uuid.UUID) (int, error) {
	total, err := r.base.q.WithTx(tx).GetTeamScore(ctx, teamID)
	if err != nil {
		return 0, fmt.Errorf("TxSolveRepo - GetTeamScoreTx: %w", err)
	}
	return int(total), nil
}
