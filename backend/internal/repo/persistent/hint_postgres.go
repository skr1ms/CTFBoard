package persistent

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wahrwelt-kit/go-pgkit/pgutil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

type HintRepo struct {
	BaseRepo
}

var _ repo.HintRepository = (*HintRepo)(nil)

func NewHintRepo(pool *pgxpool.Pool) *HintRepo {
	return &HintRepo{BaseRepo: BaseRepo{pool: pool}}
}

type hintRow struct {
	ID          uuid.UUID
	ChallengeID uuid.UUID
	Title       string
	Content     string
	Cost        int32
	OrderIndex  int32
}

func toDomainHint(r hintRow) *domain.Hint {
	return &domain.Hint{
		ID:          r.ID,
		ChallengeID: r.ChallengeID,
		Title:       r.Title,
		Content:     r.Content,
		Cost:        int(r.Cost),
		OrderIndex:  int(r.OrderIndex),
	}
}

func (r *HintRepo) Create(ctx context.Context, h *domain.Hint) error {
	h.ID = uuid.New()

	cost, err := intToInt32Safe(h.Cost)
	if err != nil {
		return fmt.Errorf("HintRepo - Create - Cost: %w", err)
	}

	orderIndex, err := intToInt32Safe(h.OrderIndex)
	if err != nil {
		return fmt.Errorf("HintRepo - Create - OrderIndex: %w", err)
	}

	err = r.Q(ctx).CreateHint(ctx, sqlc.CreateHintParams{
		ID:          h.ID,
		ChallengeID: h.ChallengeID,
		Title:       h.Title,
		Content:     h.Content,
		Cost:        cost,
		OrderIndex:  orderIndex,
	})
	if err != nil {
		return fmt.Errorf("HintRepo - Create: %w", err)
	}

	return nil
}

func (r *HintRepo) GetByID(ctx context.Context, ID uuid.UUID) (*domain.Hint, error) {
	h, err := r.Q(ctx).GetHintByID(ctx, ID)
	if err != nil {
		if pgutil.IsNoRows(err) {
			return nil, httperr.ErrHintNotFound
		}

		return nil, fmt.Errorf("HintRepo - GetByID: %w", err)
	}

	return toDomainHint(hintRow{ID: h.ID, ChallengeID: h.ChallengeID, Title: h.Title, Content: h.Content, Cost: h.Cost, OrderIndex: h.OrderIndex}), nil
}

func (r *HintRepo) GetByIDForUpdate(ctx context.Context, ID uuid.UUID) (*domain.Hint, error) {
	h, err := r.Q(ctx).GetHintByIDForUpdate(ctx, ID)
	if err != nil {
		if pgutil.IsNoRows(err) {
			return nil, httperr.ErrHintNotFound
		}

		return nil, fmt.Errorf("HintRepo - GetByIDForUpdate: %w", err)
	}

	return toDomainHint(hintRow{ID: h.ID, ChallengeID: h.ChallengeID, Title: h.Title, Content: h.Content, Cost: h.Cost, OrderIndex: h.OrderIndex}), nil
}

func (r *HintRepo) GetByChallengeID(ctx context.Context, challengeID uuid.UUID) ([]*domain.Hint, error) {
	rows, err := r.Q(ctx).GetHintsByChallengeID(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("HintRepo - GetByChallengeID: %w", err)
	}

	out := make([]*domain.Hint, 0, len(rows))
	for _, h := range rows {
		out = append(out, toDomainHint(hintRow{ID: h.ID, ChallengeID: h.ChallengeID, Title: h.Title, Content: h.Content, Cost: h.Cost, OrderIndex: h.OrderIndex}))
	}

	return out, nil
}

func (r *HintRepo) GetByChallengeIDs(ctx context.Context, challengeIDs []uuid.UUID) (map[uuid.UUID][]*domain.Hint, error) {
	if len(challengeIDs) == 0 {
		return map[uuid.UUID][]*domain.Hint{}, nil
	}

	rows, err := r.Q(ctx).GetHintsByChallengeIDs(ctx, challengeIDs)
	if err != nil {
		return nil, fmt.Errorf("HintRepo - GetByChallengeIDs: %w", err)
	}

	out := make(map[uuid.UUID][]*domain.Hint)

	for _, h := range rows {
		out[h.ChallengeID] = append(out[h.ChallengeID], toDomainHint(hintRow{ID: h.ID, ChallengeID: h.ChallengeID, Title: h.Title, Content: h.Content, Cost: h.Cost, OrderIndex: h.OrderIndex}))
	}

	return out, nil
}

func (r *HintRepo) Update(ctx context.Context, h *domain.Hint) error {
	cost, err := intToInt32Safe(h.Cost)
	if err != nil {
		return fmt.Errorf("HintRepo - Update - Cost: %w", err)
	}

	orderIndex, err := intToInt32Safe(h.OrderIndex)
	if err != nil {
		return fmt.Errorf("HintRepo - Update - OrderIndex: %w", err)
	}

	err = r.Q(ctx).UpdateHint(ctx, sqlc.UpdateHintParams{
		ID:         h.ID,
		Title:      h.Title,
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
	err := r.Q(ctx).DeleteHint(ctx, ID)
	if err != nil {
		return fmt.Errorf("HintRepo - Delete: %w", err)
	}

	return nil
}

func toDomainHintUnlockFromRow(id, hintID, teamID uuid.UUID, unlockedAt pgtype.Timestamptz) *domain.HintUnlock {
	return &domain.HintUnlock{
		ID:         id,
		HintID:     hintID,
		TeamID:     teamID,
		UnlockedAt: pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(unlockedAt)),
	}
}

func toDomainHintUnlockFromBackup(u sqlc.HintUnlock) *domain.HintUnlock {
	return &domain.HintUnlock{
		ID:           u.ID,
		HintID:       u.HintID,
		TeamID:       u.TeamID,
		UnlockedAt:   pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(u.UnlockedAt)),
		BannedTeamID: u.BannedTeamID,
	}
}

func (r *HintRepo) GetByTeamAndHint(ctx context.Context, teamID, hintID uuid.UUID) (*domain.HintUnlock, error) {
	u, err := r.Q(ctx).GetHintUnlockByTeamAndHint(ctx, sqlc.GetHintUnlockByTeamAndHintParams{
		TeamID: teamID,
		HintID: hintID,
	})
	if err != nil {
		if pgutil.IsNoRows(err) {
			return nil, httperr.ErrHintNotFound
		}

		return nil, fmt.Errorf("HintRepo - GetByTeamAndHint: %w", err)
	}

	return toDomainHintUnlockFromRow(u.ID, u.HintID, u.TeamID, u.UnlockedAt), nil
}

func (r *HintRepo) GetUnlockedHintIDs(ctx context.Context, teamID, challengeID uuid.UUID) ([]uuid.UUID, error) {
	ids, err := r.Q(ctx).GetUnlockedHintIDs(ctx, sqlc.GetUnlockedHintIDsParams{
		TeamID:      teamID,
		ChallengeID: challengeID,
	})
	if err != nil {
		return nil, fmt.Errorf("HintRepo - GetUnlockedHintIDs: %w", err)
	}

	return ids, nil
}

func (r *HintRepo) GetAll(ctx context.Context, limit, offset int) ([]*domain.HintUnlockWithDetails, error) {
	limit32, offset32, err := toLimitOffset(limit, offset)
	if err != nil {
		return nil, fmt.Errorf("HintRepo - GetAll: %w", err)
	}

	rows, err := r.Q(ctx).GetAllHintUnlocks(ctx, sqlc.GetAllHintUnlocksParams{
		Limit:  limit32,
		Offset: offset32,
	})
	if err != nil {
		return nil, fmt.Errorf("HintRepo - GetAll: %w", err)
	}

	out := make([]*domain.HintUnlockWithDetails, 0, len(rows))
	for _, row := range rows {
		out = append(out, &domain.HintUnlockWithDetails{
			ID:          row.ID,
			HintID:      row.HintID,
			TeamID:      row.TeamID,
			UnlockedAt:  pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(row.UnlockedAt)),
			ChallengeID: row.ChallengeID,
			HintCost:    int(row.HintCost),
		})
	}

	return out, nil
}

func (r *HintRepo) CountAll(ctx context.Context) (int, error) {
	n, err := r.Q(ctx).CountAllHintUnlocks(ctx)
	if err != nil {
		return 0, fmt.Errorf("HintRepo - CountAll: %w", err)
	}

	return int(n), nil
}

func (r *HintRepo) CountByTeamID(ctx context.Context, teamID uuid.UUID) (int, error) {
	n, err := r.Q(ctx).CountHintUnlocksByTeamID(ctx, teamID)
	if err != nil {
		return 0, fmt.Errorf("HintRepo - CountByTeamID: %w", err)
	}

	return int(n), nil
}

func (r *HintRepo) CreateUnlock(ctx context.Context, teamID, hintID uuid.UUID) error {
	_, err := r.Q(ctx).CreateHintUnlock(ctx, sqlc.CreateHintUnlockParams{
		ID:     uuid.New(),
		HintID: hintID,
		TeamID: teamID,
	})
	if err != nil {
		if pgutil.IsNoRows(err) {
			return httperr.ErrHintAlreadyUnlocked
		}

		return fmt.Errorf("HintRepo - CreateUnlock: %w", err)
	}

	return nil
}

func (r *HintRepo) DeleteUnlocksByTeamID(ctx context.Context, teamID uuid.UUID) error {
	err := r.Q(ctx).DeleteHintUnlocksByTeamID(ctx, teamID)
	if err != nil {
		return fmt.Errorf("HintRepo - DeleteUnlocksByTeamID: %w", err)
	}

	return nil
}

func (r *HintRepo) SoftBanUnlocksByTeamID(ctx context.Context, teamID uuid.UUID) error {
	err := r.Q(ctx).SoftBanHintUnlocksByTeamID(ctx, teamID)
	if err != nil {
		return fmt.Errorf("HintRepo - SoftBanUnlocksByTeamID: %w", err)
	}

	return nil
}

func (r *HintRepo) RestoreUnlocksByBannedTeamID(ctx context.Context, teamID uuid.UUID) error {
	err := r.Q(ctx).RestoreHintUnlocksByBannedTeamID(ctx, &teamID)
	if err != nil {
		return fmt.Errorf("HintRepo - RestoreUnlocksByBannedTeamID: %w", err)
	}

	return nil
}

func (r *HintRepo) GetAllUnlocks(ctx context.Context) ([]*domain.HintUnlock, error) {
	rows, err := r.Q(ctx).GetAllHintUnlocksSimple(ctx)
	if err != nil {
		return nil, fmt.Errorf("HintRepo - GetAllUnlocks: %w", err)
	}

	out := make([]*domain.HintUnlock, len(rows))
	for i, u := range rows {
		out[i] = toDomainHintUnlockFromRow(u.ID, u.HintID, u.TeamID, u.UnlockedAt)
	}

	return out, nil
}

func (r *HintRepo) GetAllUnlocksForBackup(ctx context.Context) ([]*domain.HintUnlock, error) {
	rows, err := r.Q(ctx).GetHintUnlocksForBackup(ctx)
	if err != nil {
		return nil, fmt.Errorf("HintRepo - GetAllUnlocksForBackup: %w", err)
	}

	out := make([]*domain.HintUnlock, len(rows))
	for i, u := range rows {
		out[i] = toDomainHintUnlockFromBackup(u)
	}

	return out, nil
}

func (r *HintRepo) GetByTeamAndHintForUpdate(ctx context.Context, teamID, hintID uuid.UUID) (*domain.HintUnlock, error) {
	u, err := r.Q(ctx).GetHintUnlockByTeamAndHintForUpdate(ctx, sqlc.GetHintUnlockByTeamAndHintForUpdateParams{
		TeamID: teamID,
		HintID: hintID,
	})
	if err != nil {
		if pgutil.IsNoRows(err) {
			return nil, httperr.ErrHintNotFound
		}

		return nil, fmt.Errorf("HintRepo - GetByTeamAndHintForUpdate: %w", err)
	}

	return toDomainHintUnlockFromRow(u.ID, u.HintID, u.TeamID, u.UnlockedAt), nil
}
