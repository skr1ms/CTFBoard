package persistent

import (
	"context"
	"fmt"
	"time"

	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/repo"
	"github.com/skr1ms/CTFBoard/internal/repo/persistent/sqlc"
)

type TxCompetitionRepo struct {
	base *TxBase
}

func (r *TxCompetitionRepo) GetCompetitionTx(ctx context.Context, tx repo.Transaction) (*entity.Competition, error) {
	pgxTx := mustPgxTx(tx)
	c, err := r.base.q.WithTx(pgxTx).GetCompetition(ctx)
	if err != nil {
		if isNoRows(err) {
			return nil, entityError.ErrCompetitionNotFound
		}
		return nil, fmt.Errorf("TxCompetitionRepo - GetCompetitionTx: %w", err)
	}
	return toEntityCompetition(c), nil
}

func (r *TxCompetitionRepo) UpdateCompetitionTx(ctx context.Context, tx repo.Transaction, c *entity.Competition) error {
	pgxTx := mustPgxTx(tx)
	minTeamSize, err := intToInt32Safe(c.MinTeamSize)
	if err != nil {
		return fmt.Errorf("TxCompetitionRepo - UpdateCompetitionTx MinTeamSize: %w", err)
	}
	maxTeamSize, err := intToInt32Safe(c.MaxTeamSize)
	if err != nil {
		return fmt.Errorf("TxCompetitionRepo - UpdateCompetitionTx MaxTeamSize: %w", err)
	}
	updatedAt := time.Now()
	err = r.base.q.WithTx(pgxTx).UpdateCompetition(ctx, sqlc.UpdateCompetitionParams{
		Name:            c.Name,
		StartTime:       c.StartTime,
		EndTime:         c.EndTime,
		FreezeTime:      c.FreezeTime,
		IsPaused:        &c.IsPaused,
		IsPublic:        &c.IsPublic,
		FlagRegex:       c.FlagRegex,
		Mode:            &c.Mode,
		AllowTeamSwitch: &c.AllowTeamSwitch,
		MinTeamSize:     &minTeamSize,
		MaxTeamSize:     &maxTeamSize,
		UpdatedAt:       &updatedAt,
	})
	if err != nil {
		return fmt.Errorf("TxCompetitionRepo - UpdateCompetitionTx: %w", err)
	}
	return nil
}
