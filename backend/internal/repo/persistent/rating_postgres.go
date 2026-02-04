package persistent

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/repo/persistent/sqlc"
)

type RatingRepo struct {
	pool *pgxpool.Pool
	q    *sqlc.Queries
}

func NewRatingRepo(pool *pgxpool.Pool) *RatingRepo {
	return &RatingRepo{
		pool: pool,
		q:    sqlc.New(pool),
	}
}

func (r *RatingRepo) CreateCTFEvent(ctx context.Context, event *entity.CTFEvent) error {
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}
	createdAt := &event.CreatedAt
	_, err := r.q.CreateCTFEvent(ctx, sqlc.CreateCTFEventParams{
		ID:        event.ID,
		Name:      event.Name,
		StartTime: event.StartTime,
		EndTime:   event.EndTime,
		Weight:    event.Weight,
		CreatedAt: createdAt,
	})
	if err != nil {
		return fmt.Errorf("RatingRepo - CreateCTFEvent: %w", err)
	}
	return nil
}

func (r *RatingRepo) GetCTFEventByID(ctx context.Context, id uuid.UUID) (*entity.CTFEvent, error) {
	row, err := r.q.GetCTFEventByID(ctx, id)
	if err != nil {
		if isNoRows(err) {
			return nil, entityError.ErrCTFEventNotFound
		}
		return nil, fmt.Errorf("RatingRepo - GetCTFEventByID: %w", err)
	}
	return ctfEventRowToEntity(row), nil
}

func (r *RatingRepo) GetAllCTFEvents(ctx context.Context) ([]*entity.CTFEvent, error) {
	rows, err := r.q.GetAllCTFEvents(ctx)
	if err != nil {
		return nil, fmt.Errorf("RatingRepo - GetAllCTFEvents: %w", err)
	}
	out := make([]*entity.CTFEvent, len(rows))
	for i := range rows {
		out[i] = ctfEventRowToEntity(rows[i])
	}
	return out, nil
}

func (r *RatingRepo) CreateTeamRating(ctx context.Context, tr *entity.TeamRating) error {
	if tr.ID == uuid.Nil {
		tr.ID = uuid.New()
	}
	if tr.CreatedAt.IsZero() {
		tr.CreatedAt = time.Now()
	}
	createdAt := &tr.CreatedAt
	rank32, err := intToInt32Safe(tr.Rank)
	if err != nil {
		return fmt.Errorf("RatingRepo - CreateTeamRating Rank: %w", err)
	}
	score32, err := intToInt32Safe(tr.Score)
	if err != nil {
		return fmt.Errorf("RatingRepo - CreateTeamRating Score: %w", err)
	}
	return r.q.CreateTeamRating(ctx, sqlc.CreateTeamRatingParams{
		ID:           tr.ID,
		TeamID:       tr.TeamID,
		CtfEventID:   tr.CTFEventID,
		Rank:         rank32,
		Score:        score32,
		RatingPoints: tr.RatingPoints,
		CreatedAt:    createdAt,
	})
}

func (r *RatingRepo) GetTeamRatingsByTeamID(ctx context.Context, teamID uuid.UUID) ([]*entity.TeamRating, error) {
	rows, err := r.q.GetTeamRatingsByTeamID(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("RatingRepo - GetTeamRatingsByTeamID: %w", err)
	}
	out := make([]*entity.TeamRating, len(rows))
	for i := range rows {
		out[i] = teamRatingRowToEntity(rows[i])
	}
	return out, nil
}

func (r *RatingRepo) UpsertGlobalRating(ctx context.Context, gr *entity.GlobalRating) error {
	var bestRank *int32
	if gr.BestRank != nil {
		v, err := intToInt32Safe(*gr.BestRank)
		if err != nil {
			return fmt.Errorf("RatingRepo - UpsertGlobalRating BestRank: %w", err)
		}
		bestRank = &v
	}
	eventsCount32, err := intToInt32Safe(gr.EventsCount)
	if err != nil {
		return fmt.Errorf("RatingRepo - UpsertGlobalRating EventsCount: %w", err)
	}
	lastUpdated := &gr.LastUpdated
	return r.q.UpsertGlobalRating(ctx, sqlc.UpsertGlobalRatingParams{
		TeamID:      gr.TeamID,
		TotalPoints: gr.TotalPoints,
		EventsCount: eventsCount32,
		BestRank:    bestRank,
		LastUpdated: lastUpdated,
	})
}

func (r *RatingRepo) GetGlobalRatings(ctx context.Context, limit, offset int) ([]*entity.GlobalRating, error) {
	limit32, err := intToInt32Safe(limit)
	if err != nil {
		return nil, fmt.Errorf("RatingRepo - GetGlobalRatings limit: %w", err)
	}
	offset32, err := intToInt32Safe(offset)
	if err != nil {
		return nil, fmt.Errorf("RatingRepo - GetGlobalRatings offset: %w", err)
	}
	rows, err := r.q.GetGlobalRatings(ctx, sqlc.GetGlobalRatingsParams{
		Limit:  limit32,
		Offset: offset32,
	})
	if err != nil {
		return nil, fmt.Errorf("RatingRepo - GetGlobalRatings: %w", err)
	}
	out := make([]*entity.GlobalRating, len(rows))
	for i := range rows {
		out[i] = globalRatingRowToEntity(rows[i])
	}
	return out, nil
}

func (r *RatingRepo) CountGlobalRatings(ctx context.Context) (int64, error) {
	return r.q.CountGlobalRatings(ctx)
}

func (r *RatingRepo) GetGlobalRatingByTeamID(ctx context.Context, teamID uuid.UUID) (*entity.GlobalRating, error) {
	row, err := r.q.GetGlobalRatingByTeamID(ctx, teamID)
	if err != nil {
		if isNoRows(err) {
			return nil, entityError.ErrGlobalRatingNotFound
		}
		return nil, fmt.Errorf("RatingRepo - GetGlobalRatingByTeamID: %w", err)
	}
	return globalRatingByTeamIDRowToEntity(row), nil
}

func ctfEventRowToEntity(row sqlc.CtfEvent) *entity.CTFEvent {
	createdAt := time.Time{}
	if row.CreatedAt != nil {
		createdAt = *row.CreatedAt
	}
	return &entity.CTFEvent{
		ID:        row.ID,
		Name:      row.Name,
		StartTime: row.StartTime,
		EndTime:   row.EndTime,
		Weight:    row.Weight,
		CreatedAt: createdAt,
	}
}

func teamRatingRowToEntity(row sqlc.TeamRating) *entity.TeamRating {
	createdAt := time.Time{}
	if row.CreatedAt != nil {
		createdAt = *row.CreatedAt
	}
	return &entity.TeamRating{
		ID:           row.ID,
		TeamID:       row.TeamID,
		CTFEventID:   row.CtfEventID,
		Rank:         int(row.Rank),
		Score:        int(row.Score),
		RatingPoints: row.RatingPoints,
		CreatedAt:    createdAt,
	}
}

func globalRatingRowToEntity(row sqlc.GetGlobalRatingsRow) *entity.GlobalRating {
	lastUpdated := time.Time{}
	if row.LastUpdated != nil {
		lastUpdated = *row.LastUpdated
	}
	var bestRank *int
	if row.BestRank != nil {
		v := int(*row.BestRank)
		bestRank = &v
	}
	return &entity.GlobalRating{
		TeamID:      row.TeamID,
		TeamName:    row.TeamName,
		TotalPoints: row.TotalPoints,
		EventsCount: int(row.EventsCount),
		BestRank:    bestRank,
		LastUpdated: lastUpdated,
	}
}

func globalRatingByTeamIDRowToEntity(row sqlc.GetGlobalRatingByTeamIDRow) *entity.GlobalRating {
	lastUpdated := time.Time{}
	if row.LastUpdated != nil {
		lastUpdated = *row.LastUpdated
	}
	var bestRank *int
	if row.BestRank != nil {
		v := int(*row.BestRank)
		bestRank = &v
	}
	return &entity.GlobalRating{
		TeamID:      row.TeamID,
		TeamName:    row.TeamName,
		TotalPoints: row.TotalPoints,
		EventsCount: int(row.EventsCount),
		BestRank:    bestRank,
		LastUpdated: lastUpdated,
	}
}
