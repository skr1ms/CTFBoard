package persistent

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skr1ms/CTFBoard/internal/repo"
)

type TxRepo struct {
	*TxBase
	*TxUserRepo
	*TxTeamRepo
	*TxChallengeRepo
	*TxSolveRepo
	*TxHintRepo
	*TxAwardRepo
	*TxAuditRepo
}

func NewTxRepo(pool *pgxpool.Pool) *TxRepo {
	base := NewTxBase(pool)
	return &TxRepo{
		TxBase:          base,
		TxUserRepo:      &TxUserRepo{base: base},
		TxTeamRepo:      &TxTeamRepo{base: base},
		TxChallengeRepo: &TxChallengeRepo{base: base},
		TxSolveRepo:     &TxSolveRepo{base: base},
		TxHintRepo:      &TxHintRepo{base: base},
		TxAwardRepo:     &TxAwardRepo{base: base},
		TxAuditRepo:     &TxAuditRepo{base: base},
	}
}

var _ repo.TxRepository = (*TxRepo)(nil)
