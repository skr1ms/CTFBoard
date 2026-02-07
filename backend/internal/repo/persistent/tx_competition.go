package persistent

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/repo"
	"github.com/skr1ms/CTFBoard/internal/repo/persistent/sqlc"
)

type TxChallengeRepo struct {
	base *TxBase
}

func (r *TxChallengeRepo) GetChallengeByIDTx(ctx context.Context, tx repo.Transaction, id uuid.UUID) (*entity.Challenge, error) {
	pgxTx := mustPgxTx(tx)
	row, err := r.base.q.WithTx(pgxTx).GetChallengeByIDForUpdate(ctx, id)
	if err != nil {
		if isNoRows(err) {
			return nil, entityError.ErrChallengeNotFound
		}
		return nil, fmt.Errorf("TxChallengeRepo - GetChallengeByIDTx: %w", err)
	}
	return toEntityChallengeFromRow(row.ID, row.Title, row.Description, row.Category, row.Points, row.InitialValue, row.MinValue, row.Decay, row.SolveCount, row.FlagHash, row.IsHidden, row.IsRegex, row.IsCaseInsensitive, row.FlagRegex, row.FlagFormatRegex), nil
}

func (r *TxChallengeRepo) IncrementChallengeSolveCountTx(ctx context.Context, tx repo.Transaction, id uuid.UUID) (int, error) {
	pgxTx := mustPgxTx(tx)
	n, err := r.base.q.WithTx(pgxTx).IncrementChallengeSolveCount(ctx, id)
	if err != nil {
		return 0, fmt.Errorf("TxChallengeRepo - IncrementChallengeSolveCountTx: %w", err)
	}
	return int(n), nil
}

func (r *TxChallengeRepo) UpdateChallengePointsTx(ctx context.Context, tx repo.Transaction, id uuid.UUID, points int) error {
	pgxTx := mustPgxTx(tx)
	pts, err := intToInt32Safe(points)
	if err != nil {
		return fmt.Errorf("TxChallengeRepo - UpdateChallengePointsTx: %w", err)
	}
	_, err = r.base.q.WithTx(pgxTx).UpdateChallengePoints(ctx, sqlc.UpdateChallengePointsParams{ID: id, Points: &pts})
	if err != nil {
		if isNoRows(err) {
			return entityError.ErrChallengeNotFound
		}
		return fmt.Errorf("TxChallengeRepo - UpdateChallengePointsTx: %w", err)
	}
	return nil
}

func (r *TxChallengeRepo) DeleteChallengeTx(ctx context.Context, tx repo.Transaction, challengeID uuid.UUID) error {
	pgxTx := mustPgxTx(tx)
	_, err := r.base.q.WithTx(pgxTx).DeleteChallenge(ctx, challengeID)
	if err != nil {
		if isNoRows(err) {
			return entityError.ErrChallengeNotFound
		}
		return fmt.Errorf("TxChallengeRepo - DeleteChallengeTx: %w", err)
	}
	return nil
}

type TxSolveRepo struct {
	base *TxBase
}

func (r *TxSolveRepo) CreateSolveTx(ctx context.Context, tx repo.Transaction, s *entity.Solve) error {
	pgxTx := mustPgxTx(tx)
	s.ID = uuid.New()
	s.SolvedAt = time.Now()
	err := r.base.q.WithTx(pgxTx).CreateSolve(ctx, sqlc.CreateSolveParams{
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

func (r *TxSolveRepo) DeleteSolvesByTeamIDTx(ctx context.Context, tx repo.Transaction, teamID uuid.UUID) error {
	pgxTx := mustPgxTx(tx)
	if err := r.base.q.WithTx(pgxTx).DeleteSolvesByTeamID(ctx, teamID); err != nil {
		return fmt.Errorf("TxSolveRepo - DeleteSolvesByTeamIDTx: %w", err)
	}
	return nil
}

func (r *TxSolveRepo) GetSolveByTeamAndChallengeTx(ctx context.Context, tx repo.Transaction, teamID, challengeID uuid.UUID) (*entity.Solve, error) {
	pgxTx := mustPgxTx(tx)
	s, err := r.base.q.WithTx(pgxTx).GetSolveByTeamAndChallengeForUpdate(ctx, sqlc.GetSolveByTeamAndChallengeForUpdateParams{
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

func (r *TxSolveRepo) GetTeamScoreTx(ctx context.Context, tx repo.Transaction, teamID uuid.UUID) (int, error) {
	pgxTx := mustPgxTx(tx)
	total, err := r.base.q.WithTx(pgxTx).GetTeamScore(ctx, teamID)
	if err != nil {
		return 0, fmt.Errorf("TxSolveRepo - GetTeamScoreTx: %w", err)
	}
	return int(total), nil
}

type TxHintRepo struct {
	base *TxBase
}

func (r *TxHintRepo) CreateHintUnlockTx(ctx context.Context, tx repo.Transaction, teamID, hintID uuid.UUID) error {
	pgxTx := mustPgxTx(tx)
	err := r.base.q.WithTx(pgxTx).CreateHintUnlock(ctx, sqlc.CreateHintUnlockParams{
		ID:     uuid.New(),
		HintID: hintID,
		TeamID: teamID,
	})
	if err != nil {
		return fmt.Errorf("TxHintRepo - CreateHintUnlockTx: %w", err)
	}
	return nil
}

func (r *TxHintRepo) GetHintUnlockByTeamAndHintTx(ctx context.Context, tx repo.Transaction, teamID, hintID uuid.UUID) (*entity.HintUnlock, error) {
	pgxTx := mustPgxTx(tx)
	u, err := r.base.q.WithTx(pgxTx).GetHintUnlockByTeamAndHintForUpdate(ctx, sqlc.GetHintUnlockByTeamAndHintForUpdateParams{
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
