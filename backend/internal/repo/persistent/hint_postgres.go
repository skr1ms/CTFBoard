package persistent

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/repo/persistent/sqlc"
)

type HintRepo struct {
	db *pgxpool.Pool
	q  *sqlc.Queries
}

func NewHintRepo(db *pgxpool.Pool) *HintRepo {
	return &HintRepo{db: db, q: sqlc.New(db)}
}

func toEntityHint(h sqlc.Hint) *entity.Hint {
	return &entity.Hint{
		ID:          h.ID,
		ChallengeID: h.ChallengeID,
		Content:     h.Content,
		Cost:        int(h.Cost),
		OrderIndex:  int(h.OrderIndex),
	}
}

func (r *HintRepo) Create(ctx context.Context, h *entity.Hint) error {
	h.ID = uuid.New()
	cost, err := intToInt32Safe(h.Cost)
	if err != nil {
		return fmt.Errorf("HintRepo - Create Cost: %w", err)
	}
	orderIndex, err := intToInt32Safe(h.OrderIndex)
	if err != nil {
		return fmt.Errorf("HintRepo - Create OrderIndex: %w", err)
	}
	err = r.q.CreateHint(ctx, sqlc.CreateHintParams{
		ID:          h.ID,
		ChallengeID: h.ChallengeID,
		Content:     h.Content,
		Cost:        cost,
		OrderIndex:  orderIndex,
	})
	if err != nil {
		return fmt.Errorf("HintRepo - Create: %w", err)
	}
	return nil
}

func (r *HintRepo) GetByID(ctx context.Context, id uuid.UUID) (*entity.Hint, error) {
	h, err := r.q.GetHintByID(ctx, id)
	if err != nil {
		if isNoRows(err) {
			return nil, entityError.ErrHintNotFound
		}
		return nil, fmt.Errorf("HintRepo - GetByID: %w", err)
	}
	return toEntityHint(h), nil
}

func (r *HintRepo) GetByChallengeID(ctx context.Context, challengeID uuid.UUID) ([]*entity.Hint, error) {
	rows, err := r.q.GetHintsByChallengeID(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("HintRepo - GetByChallengeID: %w", err)
	}
	out := make([]*entity.Hint, 0, len(rows))
	for _, h := range rows {
		out = append(out, toEntityHint(h))
	}
	return out, nil
}

func (r *HintRepo) Update(ctx context.Context, h *entity.Hint) error {
	cost, err := intToInt32Safe(h.Cost)
	if err != nil {
		return fmt.Errorf("HintRepo - Update Cost: %w", err)
	}
	orderIndex, err := intToInt32Safe(h.OrderIndex)
	if err != nil {
		return fmt.Errorf("HintRepo - Update OrderIndex: %w", err)
	}
	err = r.q.UpdateHint(ctx, sqlc.UpdateHintParams{
		ID:         h.ID,
		Content:    h.Content,
		Cost:       cost,
		OrderIndex: orderIndex,
	})
	if err != nil {
		return fmt.Errorf("HintRepo - Update: %w", err)
	}
	return nil
}

func (r *HintRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.q.DeleteHint(ctx, id); err != nil {
		return fmt.Errorf("HintRepo - Delete: %w", err)
	}
	return nil
}

type HintUnlockRepo struct {
	db *pgxpool.Pool
	q  *sqlc.Queries
}

func NewHintUnlockRepo(db *pgxpool.Pool) *HintUnlockRepo {
	return &HintUnlockRepo{db: db, q: sqlc.New(db)}
}

func toEntityHintUnlock(u sqlc.HintUnlock) *entity.HintUnlock {
	return &entity.HintUnlock{
		ID:         u.ID,
		HintID:     u.HintID,
		TeamID:     u.TeamID,
		UnlockedAt: ptrTimeToTime(u.UnlockedAt),
	}
}

func (r *HintUnlockRepo) GetByTeamAndHint(ctx context.Context, teamID, hintID uuid.UUID) (*entity.HintUnlock, error) {
	u, err := r.q.GetHintUnlockByTeamAndHint(ctx, sqlc.GetHintUnlockByTeamAndHintParams{
		TeamID: teamID,
		HintID: hintID,
	})
	if err != nil {
		if isNoRows(err) {
			return nil, entityError.ErrHintNotFound
		}
		return nil, fmt.Errorf("HintUnlockRepo - GetByTeamAndHint: %w", err)
	}
	return toEntityHintUnlock(u), nil
}

func (r *HintUnlockRepo) GetUnlockedHintIDs(ctx context.Context, teamID, challengeID uuid.UUID) ([]uuid.UUID, error) {
	return r.q.GetUnlockedHintIDs(ctx, sqlc.GetUnlockedHintIDsParams{
		TeamID:      teamID,
		ChallengeID: challengeID,
	})
}

func (r *HintUnlockRepo) Create(ctx context.Context, u *entity.HintUnlock) error {
	u.ID = uuid.New()
	return r.q.CreateHintUnlock(ctx, sqlc.CreateHintUnlockParams{
		ID:     u.ID,
		HintID: u.HintID,
		TeamID: u.TeamID,
	})
}
