package persistent

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/samber/lo"
	"github.com/wahrwelt-kit/go-pgkit/pgutil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
)

type TrackingRepo struct {
	BaseRepo
}

var _ repo.TrackingRepository = (*TrackingRepo)(nil)

func NewTrackingRepo(pool *pgxpool.Pool) *TrackingRepo {
	return &TrackingRepo{BaseRepo: BaseRepo{pool: pool}}
}

func (r *TrackingRepo) Create(ctx context.Context, entry *domain.TrackingEntry) error {
	EnsureID(&entry.ID)

	ua := &entry.UserAgent
	if entry.UserAgent == "" {
		ua = nil
	}

	err := r.Q(ctx).CreateTracking(ctx, sqlc.CreateTrackingParams{
		ID:        entry.ID,
		UserID:    entry.UserID,
		IP:        entry.IP,
		UserAgent: ua,
	})
	if err != nil {
		return fmt.Errorf("TrackingRepo - Create: %w", err)
	}

	return nil
}

func (r *TrackingRepo) GetByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.TrackingEntry, error) {
	limit32, offset32, err := toLimitOffset(limit, offset)
	if err != nil {
		return nil, fmt.Errorf("TrackingRepo - GetByUser: %w", err)
	}

	rows, err := r.Q(ctx).GetTrackingByUser(ctx, sqlc.GetTrackingByUserParams{
		UserID: userID,
		Limit:  limit32,
		Offset: offset32,
	})
	if err != nil {
		return nil, fmt.Errorf("TrackingRepo - GetByUser - scan: %w", err)
	}

	out := make([]*domain.TrackingEntry, 0, len(rows))
	for _, row := range rows {
		out = append(out, &domain.TrackingEntry{
			ID:        row.ID,
			UserID:    row.UserID,
			IP:        row.IP,
			UserAgent: lo.FromPtr(row.UserAgent),
			TrackedAt: pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(row.TrackedAt)),
		})
	}

	return out, nil
}

func (r *TrackingRepo) CountByUser(ctx context.Context, userID uuid.UUID) (int, error) {
	n, err := r.Q(ctx).CountTrackingByUser(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("TrackingRepo - CountByUser: %w", err)
	}

	return int(n), nil
}

func (r *TrackingRepo) CreateChallengeOpen(ctx context.Context, entry *domain.ChallengeOpen) error {
	EnsureID(&entry.ID)

	var ip *string

	if entry.IP != "" {
		ip = &entry.IP
	}

	err := r.Q(ctx).CreateChallengeOpen(ctx, sqlc.CreateChallengeOpenParams{
		ID:          entry.ID,
		UserID:      entry.UserID,
		ChallengeID: entry.ChallengeID,
		IP:          ip,
	})
	if err != nil {
		return fmt.Errorf("TrackingRepo - CreateChallengeOpen: %w", err)
	}

	return nil
}

func (r *TrackingRepo) GetChallengeOpensByChallenge(ctx context.Context, challengeID uuid.UUID, limit, offset int) ([]*domain.ChallengeOpen, error) {
	limit32, offset32, err := toLimitOffset(limit, offset)
	if err != nil {
		return nil, fmt.Errorf("TrackingRepo - GetChallengeOpensByChallenge: %w", err)
	}

	rows, err := r.Q(ctx).GetChallengeOpensByChallenge(ctx, sqlc.GetChallengeOpensByChallengeParams{
		ChallengeID: challengeID,
		Limit:       limit32,
		Offset:      offset32,
	})
	if err != nil {
		return nil, fmt.Errorf("TrackingRepo - GetChallengeOpensByChallenge - scan: %w", err)
	}

	out := make([]*domain.ChallengeOpen, 0, len(rows))
	for _, row := range rows {
		out = append(out, &domain.ChallengeOpen{
			ID:          row.ID,
			UserID:      row.UserID,
			ChallengeID: row.ChallengeID,
			IP:          lo.FromPtr(row.IP),
			OpenedAt:    pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(row.OpenedAt)),
		})
	}

	return out, nil
}

func (r *TrackingRepo) CountChallengeOpensByChallenge(ctx context.Context, challengeID uuid.UUID) (int, error) {
	n, err := r.Q(ctx).CountChallengeOpensByChallenge(ctx, challengeID)
	if err != nil {
		return 0, fmt.Errorf("TrackingRepo - CountChallengeOpensByChallenge: %w", err)
	}

	return int(n), nil
}

// DeleteOlderThan removes all tracking rows whose tracked_at is before
// cutoffDate. Used by the cleanup job to bound table growth.
func (r *TrackingRepo) DeleteOlderThan(ctx context.Context, cutoffDate time.Time) error {
	db := ExtractDB(ctx, r.pool)

	_, err := db.Exec(ctx, "DELETE FROM tracking WHERE tracked_at < $1", pgutil.TimeToTimestamptz(&cutoffDate))
	if err != nil {
		return fmt.Errorf("TrackingRepo - DeleteOlderThan: %w", err)
	}

	return nil
}

// DeleteChallengeOpensOlderThan removes challenge_opens rows whose opened_at
// is before cutoffDate. Used by the cleanup job to bound table growth.
func (r *TrackingRepo) DeleteChallengeOpensOlderThan(ctx context.Context, cutoffDate time.Time) error {
	db := ExtractDB(ctx, r.pool)

	_, err := db.Exec(ctx, "DELETE FROM challenge_opens WHERE opened_at < $1", pgutil.TimeToTimestamptz(&cutoffDate))
	if err != nil {
		return fmt.Errorf("TrackingRepo - DeleteChallengeOpensOlderThan: %w", err)
	}

	return nil
}
