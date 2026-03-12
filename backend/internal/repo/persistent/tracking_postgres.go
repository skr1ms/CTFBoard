package persistent

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
)

type TrackingRepo struct {
	pool *pgxpool.Pool
}

var _ repo.TrackingRepository = (*TrackingRepo)(nil)

func NewTrackingRepo(pool *pgxpool.Pool) *TrackingRepo {
	return &TrackingRepo{pool: pool}
}

func (r *TrackingRepo) q(ctx context.Context) *sqlc.Queries {
	return sqlc.New(ExtractDB(ctx, r.pool))
}

func (r *TrackingRepo) Create(ctx context.Context, entry *entity.TrackingEntry) error {
	if entry.ID == uuid.Nil {
		entry.ID = uuid.New()
	}
	ua := &entry.UserAgent
	if entry.UserAgent == "" {
		ua = nil
	}
	if err := r.q(ctx).CreateTracking(ctx, sqlc.CreateTrackingParams{
		ID:        entry.ID,
		UserID:    entry.UserID,
		IP:        entry.IP,
		UserAgent: ua,
	}); err != nil {
		return fmt.Errorf("TrackingRepo - Create: %w", err)
	}
	return nil
}

func (r *TrackingRepo) GetByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*entity.TrackingEntry, error) {
	limit32, err := intToInt32Safe(limit)
	if err != nil {
		return nil, fmt.Errorf("TrackingRepo - GetByUser - limit: %w", err)
	}
	offset32, err := intToInt32Safe(offset)
	if err != nil {
		return nil, fmt.Errorf("TrackingRepo - GetByUser - offset: %w", err)
	}
	rows, err := r.q(ctx).GetTrackingByUser(ctx, sqlc.GetTrackingByUserParams{
		UserID: userID,
		Limit:  limit32,
		Offset: offset32,
	})
	if err != nil {
		return nil, fmt.Errorf("TrackingRepo - GetByUser: %w", err)
	}
	out := make([]*entity.TrackingEntry, 0, len(rows))
	for _, row := range rows {
		out = append(out, &entity.TrackingEntry{
			ID:        row.ID,
			UserID:    row.UserID,
			IP:        row.IP,
			UserAgent: ptrStrToStr(row.UserAgent),
			TrackedAt: ptrTimeToTime(timestamptzToTime(row.TrackedAt)),
		})
	}
	return out, nil
}

func (r *TrackingRepo) CountByUser(ctx context.Context, userID uuid.UUID) (int, error) {
	n, err := r.q(ctx).CountTrackingByUser(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("TrackingRepo - CountByUser: %w", err)
	}
	return int(n), nil
}

func (r *TrackingRepo) CreateChallengeOpen(ctx context.Context, entry *entity.ChallengeOpen) error {
	if entry.ID == uuid.Nil {
		entry.ID = uuid.New()
	}
	var ip *string
	if entry.IP != "" {
		ip = &entry.IP
	}
	if err := r.q(ctx).CreateChallengeOpen(ctx, sqlc.CreateChallengeOpenParams{
		ID:          entry.ID,
		UserID:      entry.UserID,
		ChallengeID: entry.ChallengeID,
		IP:          ip,
	}); err != nil {
		return fmt.Errorf("TrackingRepo - CreateChallengeOpen: %w", err)
	}
	return nil
}

func (r *TrackingRepo) GetChallengeOpensByChallenge(ctx context.Context, challengeID uuid.UUID, limit, offset int) ([]*entity.ChallengeOpen, error) {
	limit32, err := intToInt32Safe(limit)
	if err != nil {
		return nil, fmt.Errorf("TrackingRepo - GetChallengeOpensByChallenge - limit: %w", err)
	}
	offset32, err := intToInt32Safe(offset)
	if err != nil {
		return nil, fmt.Errorf("TrackingRepo - GetChallengeOpensByChallenge - offset: %w", err)
	}
	rows, err := r.q(ctx).GetChallengeOpensByChallenge(ctx, sqlc.GetChallengeOpensByChallengeParams{
		ChallengeID: challengeID,
		Limit:       limit32,
		Offset:      offset32,
	})
	if err != nil {
		return nil, fmt.Errorf("TrackingRepo - GetChallengeOpensByChallenge: %w", err)
	}
	out := make([]*entity.ChallengeOpen, 0, len(rows))
	for _, row := range rows {
		out = append(out, &entity.ChallengeOpen{
			ID:          row.ID,
			UserID:      row.UserID,
			ChallengeID: row.ChallengeID,
			IP:          ptrStrToStr(row.IP),
			OpenedAt:    ptrTimeToTime(timestamptzToTime(row.OpenedAt)),
		})
	}
	return out, nil
}

func (r *TrackingRepo) CountChallengeOpensByChallenge(ctx context.Context, challengeID uuid.UUID) (int, error) {
	n, err := r.q(ctx).CountChallengeOpensByChallenge(ctx, challengeID)
	if err != nil {
		return 0, fmt.Errorf("TrackingRepo - CountChallengeOpensByChallenge: %w", err)
	}
	return int(n), nil
}
