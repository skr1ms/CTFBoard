package persistent

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/repo"
	"github.com/skr1ms/CTFBoard/internal/repo/persistent/sqlc"
)

type SolveRepo struct {
	db *pgxpool.Pool
	q  *sqlc.Queries
}

func NewSolveRepo(db *pgxpool.Pool) *SolveRepo {
	return &SolveRepo{db: db, q: sqlc.New(db)}
}

func toEntitySolve(s sqlc.Solf) *entity.Solve {
	return &entity.Solve{
		ID:          s.ID,
		UserID:      s.UserID,
		TeamID:      s.TeamID,
		ChallengeID: s.ChallengeID,
		SolvedAt:    ptrTimeToTime(s.SolvedAt),
	}
}

func toScoreboardEntry(row sqlc.GetScoreboardRow) *repo.ScoreboardEntry {
	return &repo.ScoreboardEntry{
		TeamID:   row.TeamID,
		TeamName: row.TeamName,
		Points:   int(row.Points),
		SolvedAt: timeFromNullable(row.SolvedAt),
	}
}

func toScoreboardEntryFrozen(row sqlc.GetScoreboardFrozenRow) *repo.ScoreboardEntry {
	return &repo.ScoreboardEntry{
		TeamID:   row.TeamID,
		TeamName: row.TeamName,
		Points:   int(row.Points),
		SolvedAt: timeFromNullable(row.SolvedAt),
	}
}

func toFirstBloodEntry(row sqlc.GetFirstBloodRow) *repo.FirstBloodEntry {
	return &repo.FirstBloodEntry{
		UserID:   row.UserID,
		Username: row.Username,
		TeamID:   row.TeamID,
		TeamName: row.TeamName,
		SolvedAt: ptrTimeToTime(row.SolvedAt),
	}
}

func (r *SolveRepo) Create(ctx context.Context, s *entity.Solve) error {
	s.ID = uuid.New()
	s.SolvedAt = time.Now()
	err := r.q.CreateSolve(ctx, sqlc.CreateSolveParams{
		ID:          s.ID,
		UserID:      s.UserID,
		TeamID:      s.TeamID,
		ChallengeID: s.ChallengeID,
		SolvedAt:    &s.SolvedAt,
	})
	if err != nil {
		if isPgUniqueViolation(err) {
			return entityError.ErrAlreadySolved
		}
		return fmt.Errorf("SolveRepo - Create: %w", err)
	}
	return nil
}

func (r *SolveRepo) GetByID(ctx context.Context, id uuid.UUID) (*entity.Solve, error) {
	s, err := r.q.GetSolveByID(ctx, id)
	if err != nil {
		if isNoRows(err) {
			return nil, entityError.ErrSolveNotFound
		}
		return nil, fmt.Errorf("SolveRepo - GetByID: %w", err)
	}
	return toEntitySolve(s), nil
}

func (r *SolveRepo) GetByTeamAndChallenge(ctx context.Context, teamID, challengeID uuid.UUID) (*entity.Solve, error) {
	s, err := r.q.GetSolveByTeamAndChallenge(ctx, sqlc.GetSolveByTeamAndChallengeParams{
		TeamID:      teamID,
		ChallengeID: challengeID,
	})
	if err != nil {
		if isNoRows(err) {
			return nil, entityError.ErrSolveNotFound
		}
		return nil, fmt.Errorf("SolveRepo - GetByTeamAndChallenge: %w", err)
	}
	return toEntitySolve(s), nil
}

func (r *SolveRepo) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.Solve, error) {
	rows, err := r.q.GetSolvesByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("SolveRepo - GetByUserID: %w", err)
	}
	out := make([]*entity.Solve, 0, len(rows))
	for _, s := range rows {
		out = append(out, toEntitySolve(s))
	}
	return out, nil
}

func (r *SolveRepo) GetScoreboard(ctx context.Context) ([]*repo.ScoreboardEntry, error) {
	rows, err := r.q.GetScoreboard(ctx)
	if err != nil {
		return nil, fmt.Errorf("SolveRepo - GetScoreboard: %w", err)
	}
	out := make([]*repo.ScoreboardEntry, 0, len(rows))
	for _, row := range rows {
		out = append(out, toScoreboardEntry(row))
	}
	return out, nil
}

func (r *SolveRepo) GetScoreboardFrozen(ctx context.Context, freezeTime time.Time) ([]*repo.ScoreboardEntry, error) {
	rows, err := r.q.GetScoreboardFrozen(ctx, sqlc.GetScoreboardFrozenParams{
		SolvedAt:  &freezeTime,
		CreatedAt: &freezeTime,
	})
	if err != nil {
		return nil, fmt.Errorf("SolveRepo - GetScoreboardFrozen: %w", err)
	}
	out := make([]*repo.ScoreboardEntry, 0, len(rows))
	for _, row := range rows {
		out = append(out, toScoreboardEntryFrozen(row))
	}
	return out, nil
}

func (r *SolveRepo) GetTeamScore(ctx context.Context, teamID uuid.UUID) (int, error) {
	total, err := r.q.GetTeamScore(ctx, teamID)
	if err != nil {
		return 0, fmt.Errorf("SolveRepo - GetTeamScore: %w", err)
	}
	return int(total), nil
}

func (r *SolveRepo) GetFirstBlood(ctx context.Context, challengeID uuid.UUID) (*repo.FirstBloodEntry, error) {
	row, err := r.q.GetFirstBlood(ctx, challengeID)
	if err != nil {
		if isNoRows(err) {
			return nil, entityError.ErrSolveNotFound
		}
		return nil, fmt.Errorf("SolveRepo - GetFirstBlood: %w", err)
	}
	return toFirstBloodEntry(row), nil
}

func (r *SolveRepo) GetAll(ctx context.Context) ([]*entity.Solve, error) {
	rows, err := r.q.GetAllSolves(ctx)
	if err != nil {
		return nil, fmt.Errorf("SolveRepo - GetAll: %w", err)
	}
	out := make([]*entity.Solve, 0, len(rows))
	for _, s := range rows {
		out = append(out, toEntitySolve(s))
	}
	return out, nil
}
