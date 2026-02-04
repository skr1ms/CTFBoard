package persistent

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/repo"
	"github.com/skr1ms/CTFBoard/internal/repo/persistent/sqlc"
)

type ChallengeRepo struct {
	db *pgxpool.Pool
	q  *sqlc.Queries
}

func NewChallengeRepo(db *pgxpool.Pool) *ChallengeRepo {
	return &ChallengeRepo{db: db, q: sqlc.New(db)}
}

func toEntityChallengeFromRow(id uuid.UUID, title, description string, category *string, points *int32, initialValue, minValue, decay, solveCount int32, flagHash string, isHidden, isRegex, isCaseInsensitive *bool, flagRegex, flagFormatRegex *string) *entity.Challenge {
	var pts int
	if points != nil {
		pts = int(*points)
	}
	return &entity.Challenge{
		ID:                id,
		Title:             title,
		Description:       description,
		Category:          ptrStrToStr(category),
		Points:            pts,
		InitialValue:      int(initialValue),
		MinValue:          int(minValue),
		Decay:             int(decay),
		SolveCount:        int(solveCount),
		FlagHash:          flagHash,
		IsHidden:          boolPtrToBool(isHidden),
		IsRegex:           boolPtrToBool(isRegex),
		IsCaseInsensitive: boolPtrToBool(isCaseInsensitive),
		FlagRegex:         ptrStrToStr(flagRegex),
		FlagFormatRegex:   flagFormatRegex,
	}
}

func (r *ChallengeRepo) Create(ctx context.Context, c *entity.Challenge) error {
	c.ID = uuid.New()
	pts, err := intToInt32Safe(c.Points)
	if err != nil {
		return fmt.Errorf("ChallengeRepo - Create Points: %w", err)
	}
	initialValue, err := intToInt32Safe(c.InitialValue)
	if err != nil {
		return fmt.Errorf("ChallengeRepo - Create InitialValue: %w", err)
	}
	minValue, err := intToInt32Safe(c.MinValue)
	if err != nil {
		return fmt.Errorf("ChallengeRepo - Create MinValue: %w", err)
	}
	decay, err := intToInt32Safe(c.Decay)
	if err != nil {
		return fmt.Errorf("ChallengeRepo - Create Decay: %w", err)
	}
	solveCount, err := intToInt32Safe(c.SolveCount)
	if err != nil {
		return fmt.Errorf("ChallengeRepo - Create SolveCount: %w", err)
	}
	err = r.q.CreateChallenge(ctx, sqlc.CreateChallengeParams{
		ID:                c.ID,
		Title:             c.Title,
		Description:       c.Description,
		Category:          strPtrOrNil(c.Category),
		Points:            &pts,
		InitialValue:      initialValue,
		MinValue:          minValue,
		Decay:             decay,
		SolveCount:        solveCount,
		FlagHash:          c.FlagHash,
		IsHidden:          &c.IsHidden,
		IsRegex:           &c.IsRegex,
		IsCaseInsensitive: &c.IsCaseInsensitive,
		FlagRegex:         strPtrOrNil(c.FlagRegex),
		FlagFormatRegex:   c.FlagFormatRegex,
	})
	if err != nil {
		return fmt.Errorf("ChallengeRepo - Create: %w", err)
	}
	return nil
}

func (r *ChallengeRepo) GetByID(ctx context.Context, id uuid.UUID) (*entity.Challenge, error) {
	row, err := r.q.GetChallengeByID(ctx, id)
	if err != nil {
		if isNoRows(err) {
			return nil, entityError.ErrChallengeNotFound
		}
		return nil, fmt.Errorf("ChallengeRepo - GetByID: %w", err)
	}
	return toEntityChallengeFromRow(row.ID, row.Title, row.Description, row.Category, row.Points, row.InitialValue, row.MinValue, row.Decay, row.SolveCount, row.FlagHash, row.IsHidden, row.IsRegex, row.IsCaseInsensitive, row.FlagRegex, row.FlagFormatRegex), nil
}

func (r *ChallengeRepo) listForTeamByTag(ctx context.Context, teamID, tagID uuid.UUID) ([]*repo.ChallengeWithSolved, error) {
	rows, err := r.q.ListChallengesForTeamByTag(ctx, sqlc.ListChallengesForTeamByTagParams{TagID: tagID, TeamID: teamID})
	if err != nil {
		return nil, fmt.Errorf("ChallengeRepo - GetAll: %w", err)
	}
	out := make([]*repo.ChallengeWithSolved, 0, len(rows))
	for _, row := range rows {
		out = append(out, &repo.ChallengeWithSolved{
			Challenge: toEntityChallengeFromRow(row.ID, row.Title, row.Description, row.Category, row.Points, row.InitialValue, row.MinValue, row.Decay, row.SolveCount, row.FlagHash, row.IsHidden, row.IsRegex, row.IsCaseInsensitive, row.FlagRegex, row.FlagFormatRegex),
			Solved:    row.Solved == 1,
		})
	}
	return out, nil
}

func (r *ChallengeRepo) listByTag(ctx context.Context, tagID uuid.UUID) ([]*repo.ChallengeWithSolved, error) {
	rows, err := r.q.ListChallengesByTag(ctx, tagID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeRepo - GetAll: %w", err)
	}
	out := make([]*repo.ChallengeWithSolved, 0, len(rows))
	for _, row := range rows {
		out = append(out, &repo.ChallengeWithSolved{
			Challenge: toEntityChallengeFromRow(row.ID, row.Title, row.Description, row.Category, row.Points, row.InitialValue, row.MinValue, row.Decay, row.SolveCount, row.FlagHash, row.IsHidden, row.IsRegex, row.IsCaseInsensitive, row.FlagRegex, row.FlagFormatRegex),
			Solved:    false,
		})
	}
	return out, nil
}

func (r *ChallengeRepo) listForTeam(ctx context.Context, teamID uuid.UUID) ([]*repo.ChallengeWithSolved, error) {
	rows, err := r.q.ListChallengesForTeam(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeRepo - GetAll: %w", err)
	}
	out := make([]*repo.ChallengeWithSolved, 0, len(rows))
	for _, row := range rows {
		out = append(out, &repo.ChallengeWithSolved{
			Challenge: toEntityChallengeFromRow(row.ID, row.Title, row.Description, row.Category, row.Points, row.InitialValue, row.MinValue, row.Decay, row.SolveCount, row.FlagHash, row.IsHidden, row.IsRegex, row.IsCaseInsensitive, row.FlagRegex, row.FlagFormatRegex),
			Solved:    row.Solved == 1,
		})
	}
	return out, nil
}

func (r *ChallengeRepo) listAllChallenges(ctx context.Context) ([]*repo.ChallengeWithSolved, error) {
	rows, err := r.q.ListChallenges(ctx)
	if err != nil {
		return nil, fmt.Errorf("ChallengeRepo - GetAll: %w", err)
	}
	out := make([]*repo.ChallengeWithSolved, 0, len(rows))
	for _, row := range rows {
		out = append(out, &repo.ChallengeWithSolved{
			Challenge: toEntityChallengeFromRow(row.ID, row.Title, row.Description, row.Category, row.Points, row.InitialValue, row.MinValue, row.Decay, row.SolveCount, row.FlagHash, row.IsHidden, row.IsRegex, row.IsCaseInsensitive, row.FlagRegex, row.FlagFormatRegex),
			Solved:    false,
		})
	}
	return out, nil
}

func (r *ChallengeRepo) GetAll(ctx context.Context, teamID, tagID *uuid.UUID) ([]*repo.ChallengeWithSolved, error) {
	if tagID != nil && teamID != nil {
		return r.listForTeamByTag(ctx, *teamID, *tagID)
	}
	if tagID != nil {
		return r.listByTag(ctx, *tagID)
	}
	if teamID != nil {
		return r.listForTeam(ctx, *teamID)
	}
	return r.listAllChallenges(ctx)
}

func (r *ChallengeRepo) Update(ctx context.Context, c *entity.Challenge) error {
	pts, err := intToInt32Safe(c.Points)
	if err != nil {
		return fmt.Errorf("ChallengeRepo - Update Points: %w", err)
	}
	initialValue, err := intToInt32Safe(c.InitialValue)
	if err != nil {
		return fmt.Errorf("ChallengeRepo - Update InitialValue: %w", err)
	}
	minValue, err := intToInt32Safe(c.MinValue)
	if err != nil {
		return fmt.Errorf("ChallengeRepo - Update MinValue: %w", err)
	}
	decay, err := intToInt32Safe(c.Decay)
	if err != nil {
		return fmt.Errorf("ChallengeRepo - Update Decay: %w", err)
	}
	err = r.q.UpdateChallenge(ctx, sqlc.UpdateChallengeParams{
		ID:                c.ID,
		Title:             c.Title,
		Description:       c.Description,
		Category:          strPtrOrNil(c.Category),
		Points:            &pts,
		InitialValue:      initialValue,
		MinValue:          minValue,
		Decay:             decay,
		FlagHash:          c.FlagHash,
		IsHidden:          &c.IsHidden,
		IsRegex:           &c.IsRegex,
		IsCaseInsensitive: &c.IsCaseInsensitive,
		FlagRegex:         strPtrOrNil(c.FlagRegex),
		FlagFormatRegex:   c.FlagFormatRegex,
	})
	if err != nil {
		return fmt.Errorf("ChallengeRepo - Update: %w", err)
	}
	return nil
}

func (r *ChallengeRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.q.DeleteChallenge(ctx, id)
	if err != nil {
		if isNoRows(err) {
			return entityError.ErrChallengeNotFound
		}
		return fmt.Errorf("ChallengeRepo - Delete: %w", err)
	}
	return nil
}

func (r *ChallengeRepo) IncrementSolveCount(ctx context.Context, id uuid.UUID) (int, error) {
	n, err := r.q.IncrementChallengeSolveCount(ctx, id)
	if err != nil {
		return 0, fmt.Errorf("ChallengeRepo - IncrementSolveCount: %w", err)
	}
	return int(n), nil
}

func (r *ChallengeRepo) UpdatePoints(ctx context.Context, id uuid.UUID, points int) error {
	pts, err := intToInt32Safe(points)
	if err != nil {
		return fmt.Errorf("ChallengeRepo - UpdatePoints: %w", err)
	}
	_, err = r.q.UpdateChallengePoints(ctx, sqlc.UpdateChallengePointsParams{ID: id, Points: &pts})
	if err != nil {
		if isNoRows(err) {
			return entityError.ErrChallengeNotFound
		}
		return fmt.Errorf("ChallengeRepo - UpdatePoints: %w", err)
	}
	return nil
}
