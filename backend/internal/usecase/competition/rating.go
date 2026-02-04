package competition

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/repo"
)

func CalculateRatingPoints(rank, totalTeams int, weight float64) float64 {
	if rank <= 0 || totalTeams <= 0 {
		return 0
	}
	basePoints := float64(totalTeams-rank+1) / float64(totalTeams) * 100
	return basePoints * weight
}

type RatingUseCase struct {
	ratingRepo repo.RatingRepository
	solveRepo  repo.SolveRepository
	teamRepo   repo.TeamRepository
}

func NewRatingUseCase(
	ratingRepo repo.RatingRepository,
	solveRepo repo.SolveRepository,
	teamRepo repo.TeamRepository,
) *RatingUseCase {
	return &RatingUseCase{
		ratingRepo: ratingRepo,
		solveRepo:  solveRepo,
		teamRepo:   teamRepo,
	}
}

func (uc *RatingUseCase) GetGlobalRatings(ctx context.Context, page, perPage int) ([]*entity.GlobalRating, int64, error) {
	if perPage <= 0 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}
	offset := (page - 1) * perPage
	if offset < 0 {
		offset = 0
	}
	list, err := uc.ratingRepo.GetGlobalRatings(ctx, perPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("RatingUseCase - GetGlobalRatings: %w", err)
	}
	total, err := uc.ratingRepo.CountGlobalRatings(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("RatingUseCase - GetGlobalRatings count: %w", err)
	}
	return list, total, nil
}

func (uc *RatingUseCase) GetTeamRating(ctx context.Context, teamID uuid.UUID) (*entity.GlobalRating, []*entity.TeamRating, error) {
	global, err := uc.ratingRepo.GetGlobalRatingByTeamID(ctx, teamID)
	if err != nil {
		if errors.Is(err, entityError.ErrGlobalRatingNotFound) {
			return nil, nil, err
		}
		return nil, nil, fmt.Errorf("RatingUseCase - GetTeamRating: %w", err)
	}
	teamRatings, err := uc.ratingRepo.GetTeamRatingsByTeamID(ctx, teamID)
	if err != nil {
		return nil, nil, fmt.Errorf("RatingUseCase - GetTeamRating team ratings: %w", err)
	}
	return global, teamRatings, nil
}

func (uc *RatingUseCase) GetCTFEvents(ctx context.Context) ([]*entity.CTFEvent, error) {
	list, err := uc.ratingRepo.GetAllCTFEvents(ctx)
	if err != nil {
		return nil, fmt.Errorf("RatingUseCase - GetCTFEvents: %w", err)
	}
	return list, nil
}

func (uc *RatingUseCase) CreateCTFEvent(ctx context.Context, name string, startTime, endTime time.Time, weight float64) (*entity.CTFEvent, error) {
	if name == "" {
		return nil, fmt.Errorf("RatingUseCase - CreateCTFEvent: name is required")
	}
	if endTime.Before(startTime) {
		return nil, fmt.Errorf("RatingUseCase - CreateCTFEvent: end_time must be after start_time")
	}
	if weight <= 0 {
		weight = 1.0
	}
	event := &entity.CTFEvent{
		ID:        uuid.New(),
		Name:      name,
		StartTime: startTime,
		EndTime:   endTime,
		Weight:    weight,
	}
	if err := uc.ratingRepo.CreateCTFEvent(ctx, event); err != nil {
		return nil, fmt.Errorf("RatingUseCase - CreateCTFEvent: %w", err)
	}
	return event, nil
}

func (uc *RatingUseCase) FinalizeCTFEvent(ctx context.Context, eventID uuid.UUID) error {
	event, err := uc.ratingRepo.GetCTFEventByID(ctx, eventID)
	if err != nil {
		if errors.Is(err, entityError.ErrCTFEventNotFound) {
			return err
		}
		return fmt.Errorf("RatingUseCase - FinalizeCTFEvent get event: %w", err)
	}
	scoreboard, err := uc.solveRepo.GetScoreboard(ctx)
	if err != nil {
		return fmt.Errorf("RatingUseCase - FinalizeCTFEvent get scoreboard: %w", err)
	}
	totalTeams := len(scoreboard)
	for rank, entry := range scoreboard {
		ratingPoints := CalculateRatingPoints(rank+1, totalTeams, event.Weight)
		tr := &entity.TeamRating{
			TeamID:       entry.TeamID,
			CTFEventID:   eventID,
			Rank:         rank + 1,
			Score:        entry.Points,
			RatingPoints: ratingPoints,
		}
		if err := uc.ratingRepo.CreateTeamRating(ctx, tr); err != nil {
			return fmt.Errorf("RatingUseCase - FinalizeCTFEvent create team rating: %w", err)
		}
	}
	return uc.recalculateGlobalRatings(ctx)
}

func (uc *RatingUseCase) recalculateGlobalRatings(ctx context.Context) error {
	teams, err := uc.teamRepo.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("RatingUseCase - recalculateGlobalRatings get teams: %w", err)
	}
	for _, team := range teams {
		ratings, err := uc.ratingRepo.GetTeamRatingsByTeamID(ctx, team.ID)
		if err != nil {
			return fmt.Errorf("RatingUseCase - recalculateGlobalRatings get team ratings: %w", err)
		}
		var totalPoints float64
		var bestRank *int
		for _, r := range ratings {
			totalPoints += r.RatingPoints
			if bestRank == nil || r.Rank < *bestRank {
				bestRank = &r.Rank
			}
		}
		gr := &entity.GlobalRating{
			TeamID:      team.ID,
			TeamName:    team.Name,
			TotalPoints: totalPoints,
			EventsCount: len(ratings),
			BestRank:    bestRank,
			LastUpdated: time.Now(),
		}
		if err := uc.ratingRepo.UpsertGlobalRating(ctx, gr); err != nil {
			return fmt.Errorf("RatingUseCase - recalculateGlobalRatings upsert: %w", err)
		}
	}
	return nil
}
