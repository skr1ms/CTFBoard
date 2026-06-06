package persistent

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wahrwelt-kit/go-pgkit/pgutil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
)

type RatingRepo struct {
	BaseRepo
}

var _ repo.RatingRepository = (*RatingRepo)(nil)

func NewRatingRepo(pool *pgxpool.Pool) *RatingRepo {
	return &RatingRepo{BaseRepo: BaseRepo{pool: pool}}
}

func (r *RatingRepo) Upsert(ctx context.Context, rating *domain.Rating) error {
	now := time.Now()

	row, err := r.Q(ctx).UpsertRating(ctx, sqlc.UpsertRatingParams{
		ChallengeID: rating.ChallengeID,
		UserID:      rating.UserID,
		TeamID:      rating.TeamID,
		Value:       int32(rating.Value),
		Review:      rating.Review,
		UpdatedAt:   pgutil.TimeToTimestamptz(&now),
	})
	if err != nil {
		return fmt.Errorf("RatingRepo - Upsert: %w", err)
	}

	rating.ID = row.ID
	rating.CreatedAt = pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(row.CreatedAt))
	rating.UpdatedAt = pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(row.UpdatedAt))

	return nil
}

func (r *RatingRepo) GetByChallengeID(ctx context.Context, challengeID uuid.UUID) ([]*domain.Rating, error) {
	rows, err := r.Q(ctx).GetRatingsByChallengeID(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("RatingRepo - GetByChallengeID: %w", err)
	}

	out := make([]*domain.Rating, len(rows))
	for i := range rows {
		out[i] = toDomainRating(rows[i])
	}

	return out, nil
}

func (r *RatingRepo) GetByTeamAndChallenge(ctx context.Context, teamID, challengeID uuid.UUID) (*domain.Rating, error) {
	row, err := GetOrNotFound(func() (sqlc.Rating, error) {
		return r.Q(ctx).GetRatingByTeamAndChallenge(ctx, sqlc.GetRatingByTeamAndChallengeParams{TeamID: teamID, ChallengeID: challengeID})
	}, apperr.ErrRatingNotFound, "RatingRepo - GetByTeamAndChallenge")
	if err != nil {
		return nil, err
	}

	return toDomainRating(row), nil
}

func (r *RatingRepo) GetAll(ctx context.Context) ([]*domain.Rating, error) {
	rows, err := r.Q(ctx).GetAllRatings(ctx)
	if err != nil {
		return nil, fmt.Errorf("RatingRepo - GetAll: %w", err)
	}

	out := make([]*domain.Rating, len(rows))
	for i := range rows {
		out[i] = toDomainRating(rows[i])
	}

	return out, nil
}

func (r *RatingRepo) DeleteByTeamID(ctx context.Context, teamID uuid.UUID) error {
	if err := r.Q(ctx).DeleteRatingsByTeamID(ctx, teamID); err != nil {
		return fmt.Errorf("RatingRepo - DeleteByTeamID: %w", err)
	}

	return nil
}

func toDomainRating(row sqlc.Rating) *domain.Rating {
	return &domain.Rating{
		ID:          row.ID,
		ChallengeID: row.ChallengeID,
		UserID:      row.UserID,
		TeamID:      row.TeamID,
		Value:       int(row.Value),
		Review:      row.Review,
		CreatedAt:   pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(row.CreatedAt)),
		UpdatedAt:   pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(row.UpdatedAt)),
	}
}
