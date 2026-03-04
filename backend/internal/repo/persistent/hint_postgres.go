package persistent

import (
	"context"
	"fmt"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type HintRepo struct {
	pool *pgxpool.Pool
}

var _ repo.HintRepository = (*HintRepo)(nil)

func NewHintRepo(pool *pgxpool.Pool) *HintRepo {
	return &HintRepo{pool: pool}
}

func (r *HintRepo) q(ctx context.Context) *sqlc.Queries {
	return sqlc.New(ExtractDB(ctx, r.pool))
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
		return fmt.Errorf("HintRepo - Create - Cost: %w", err)
	}
	orderIndex, err := intToInt32Safe(h.OrderIndex)
	if err != nil {
		return fmt.Errorf("HintRepo - Create - OrderIndex: %w", err)
	}
	err = r.q(ctx).CreateHint(ctx, sqlc.CreateHintParams{
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

func (r *HintRepo) GetByID(ctx context.Context, ID uuid.UUID) (*entity.Hint, error) {
	h, err := r.q(ctx).GetHintByID(ctx, ID)
	if err != nil {
		if isNoRows(err) {
			return nil, httperr.ErrHintNotFound
		}
		return nil, fmt.Errorf("HintRepo - GetByID: %w", err)
	}
	return toEntityHint(h), nil
}

func (r *HintRepo) GetByChallengeID(ctx context.Context, challengeID uuid.UUID) ([]*entity.Hint, error) {
	rows, err := r.q(ctx).GetHintsByChallengeID(ctx, challengeID)
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
		return fmt.Errorf("HintRepo - Update - Cost: %w", err)
	}
	orderIndex, err := intToInt32Safe(h.OrderIndex)
	if err != nil {
		return fmt.Errorf("HintRepo - Update - OrderIndex: %w", err)
	}
	err = r.q(ctx).UpdateHint(ctx, sqlc.UpdateHintParams{
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

// Delete removes a hint by ID. Idempotent: returns nil if the hint does not exist.
func (r *HintRepo) Delete(ctx context.Context, ID uuid.UUID) error {
	if err := r.q(ctx).DeleteHint(ctx, ID); err != nil {
		return fmt.Errorf("HintRepo - Delete: %w", err)
	}
	return nil
}

func toEntityHintUnlock(u sqlc.HintUnlock) *entity.HintUnlock {
	return &entity.HintUnlock{
		ID:         u.ID,
		HintID:     u.HintID,
		TeamID:     u.TeamID,
		UnlockedAt: ptrTimeToTime(u.UnlockedAt),
	}
}

func (r *HintRepo) GetByTeamAndHint(ctx context.Context, teamID, hintID uuid.UUID) (*entity.HintUnlock, error) {
	u, err := r.q(ctx).GetHintUnlockByTeamAndHint(ctx, sqlc.GetHintUnlockByTeamAndHintParams{
		TeamID: teamID,
		HintID: hintID,
	})
	if err != nil {
		if isNoRows(err) {
			return nil, httperr.ErrHintNotFound
		}
		return nil, fmt.Errorf("HintRepo - GetByTeamAndHint: %w", err)
	}
	return toEntityHintUnlock(u), nil
}

func (r *HintRepo) GetUnlockedHintIDs(ctx context.Context, teamID, challengeID uuid.UUID) ([]uuid.UUID, error) {
	ids, err := r.q(ctx).GetUnlockedHintIDs(ctx, sqlc.GetUnlockedHintIDsParams{
		TeamID:      teamID,
		ChallengeID: challengeID,
	})
	if err != nil {
		return nil, fmt.Errorf("HintRepo - GetUnlockedHintIDs: %w", err)
	}
	return ids, nil
}

func (r *HintRepo) GetAll(ctx context.Context, limit, offset int) ([]*entity.HintUnlockWithDetails, error) {
	limit32, err := intToInt32Safe(limit)
	if err != nil {
		return nil, fmt.Errorf("HintRepo - GetAll - limit: %w", err)
	}
	offset32, err := intToInt32Safe(offset)
	if err != nil {
		return nil, fmt.Errorf("HintRepo - GetAll - offset: %w", err)
	}
	rows, err := r.q(ctx).GetAllHintUnlocks(ctx, sqlc.GetAllHintUnlocksParams{
		Limit:  limit32,
		Offset: offset32,
	})
	if err != nil {
		return nil, fmt.Errorf("HintRepo - GetAll: %w", err)
	}
	out := make([]*entity.HintUnlockWithDetails, 0, len(rows))
	for _, row := range rows {
		out = append(out, &entity.HintUnlockWithDetails{
			ID:          row.ID,
			HintID:      row.HintID,
			TeamID:      row.TeamID,
			UnlockedAt:  ptrTimeToTime(row.UnlockedAt),
			ChallengeID: row.ChallengeID,
			HintCost:    int(row.HintCost),
		})
	}
	return out, nil
}

func (r *HintRepo) CountAll(ctx context.Context) (int, error) {
	n, err := r.q(ctx).CountAllHintUnlocks(ctx)
	if err != nil {
		return 0, fmt.Errorf("HintRepo - CountAll: %w", err)
	}
	return int(n), nil
}

func (r *HintRepo) CountByTeamID(ctx context.Context, teamID uuid.UUID) (int, error) {
	n, err := r.q(ctx).CountHintUnlocksByTeamID(ctx, teamID)
	if err != nil {
		return 0, fmt.Errorf("HintRepo - CountByTeamID: %w", err)
	}
	return int(n), nil
}

func (r *HintRepo) CreateUnlock(ctx context.Context, teamID, hintID uuid.UUID) error {
	err := r.q(ctx).CreateHintUnlock(ctx, sqlc.CreateHintUnlockParams{
		ID:     uuid.New(),
		HintID: hintID,
		TeamID: teamID,
	})
	if err != nil {
		return fmt.Errorf("HintRepo - CreateUnlock: %w", err)
	}
	return nil
}

func (r *HintRepo) GetAllUnlocks(ctx context.Context) ([]*entity.HintUnlock, error) {
	rows, err := r.q(ctx).GetAllHintUnlocksSimple(ctx)
	if err != nil {
		return nil, fmt.Errorf("HintRepo - GetAllUnlocks: %w", err)
	}
	out := make([]*entity.HintUnlock, len(rows))
	for i, u := range rows {
		out[i] = toEntityHintUnlock(u)
	}
	return out, nil
}

func (r *HintRepo) GetByTeamAndHintForUpdate(ctx context.Context, teamID, hintID uuid.UUID) (*entity.HintUnlock, error) {
	u, err := r.q(ctx).GetHintUnlockByTeamAndHintForUpdate(ctx, sqlc.GetHintUnlockByTeamAndHintForUpdateParams{
		TeamID: teamID,
		HintID: hintID,
	})
	if err != nil {
		if isNoRows(err) {
			return nil, httperr.ErrHintNotFound
		}
		return nil, fmt.Errorf("HintRepo - GetByTeamAndHintForUpdate: %w", err)
	}
	return toEntityHintUnlock(u), nil
}
