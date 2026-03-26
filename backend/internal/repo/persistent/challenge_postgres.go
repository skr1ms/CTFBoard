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
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

type ChallengeRepo struct {
	BaseRepo
}

var _ repo.ChallengeRepository = (*ChallengeRepo)(nil)

func NewChallengeRepo(pool *pgxpool.Pool) *ChallengeRepo {
	return &ChallengeRepo{BaseRepo: BaseRepo{pool: pool}}
}

type challengeRow struct {
	ID                uuid.UUID
	Title             string
	Description       string
	Category          *string
	Points            *int32
	InitialValue      int32
	MinValue          int32
	Decay             int32
	SolveCount        int32
	FlagHash          string
	ConnectionInfo    string
	MaxAttempts       int32
	Position          int32
	State             string
	IsRegex           *bool
	IsCaseInsensitive *bool
	FlagRegex         *string
	FlagFormatRegex   *string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func toDomainChallenge(r challengeRow) *domain.Challenge {
	var pts int

	if r.Points != nil {
		pts = int(*r.Points)
	}

	return &domain.Challenge{
		ID:                r.ID,
		Title:             r.Title,
		Description:       r.Description,
		Category:          lo.FromPtr(r.Category),
		Points:            pts,
		InitialValue:      int(r.InitialValue),
		MinValue:          int(r.MinValue),
		Decay:             int(r.Decay),
		SolveCount:        int(r.SolveCount),
		FlagHash:          r.FlagHash,
		ConnectionInfo:    r.ConnectionInfo,
		MaxAttempts:       int(r.MaxAttempts),
		Position:          int(r.Position),
		State:             r.State,
		IsRegex:           lo.FromPtr(r.IsRegex),
		IsCaseInsensitive: lo.FromPtr(r.IsCaseInsensitive),
		FlagRegex:         r.FlagRegex,
		FlagFormatRegex:   r.FlagFormatRegex,
		CreatedAt:         r.CreatedAt,
		UpdatedAt:         r.UpdatedAt,
	}
}

func (r *ChallengeRepo) GetByID(ctx context.Context, ID uuid.UUID) (*domain.Challenge, error) {
	row, err := r.Q(ctx).GetChallengeByID(ctx, ID)
	if err != nil {
		if pgutil.IsNoRows(err) {
			return nil, httperr.ErrChallengeNotFound
		}

		return nil, fmt.Errorf("ChallengeRepo - GetByID: %w", err)
	}

	return toDomainChallenge(challengeRow{
		ID: row.ID, Title: row.Title, Description: row.Description, Category: lo.EmptyableToPtr(row.Category), Points: row.Points,
		InitialValue: row.InitialValue, MinValue: row.MinValue, Decay: row.Decay, SolveCount: row.SolveCount,
		FlagHash: row.FlagHash, ConnectionInfo: row.ConnectionInfo, MaxAttempts: row.MaxAttempts, Position: row.Position, State: row.State,
		IsRegex: row.IsRegex, IsCaseInsensitive: row.IsCaseInsensitive, FlagRegex: row.FlagRegex, FlagFormatRegex: row.FlagFormatRegex,
		CreatedAt: pgutil.TimestamptzToTimeZero(row.CreatedAt), UpdatedAt: pgutil.TimestamptzToTimeZero(row.UpdatedAt),
	}), nil
}

func (r *ChallengeRepo) GetByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*domain.Challenge, error) {
	if len(ids) == 0 {
		return map[uuid.UUID]*domain.Challenge{}, nil
	}

	rows, err := r.Q(ctx).GetChallengesByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("ChallengeRepo - GetByIDs: %w", err)
	}

	out := make(map[uuid.UUID]*domain.Challenge, len(rows))
	for _, row := range rows {
		out[row.ID] = toDomainChallenge(challengeRow{
			ID: row.ID, Title: row.Title, Description: row.Description, Category: lo.EmptyableToPtr(row.Category), Points: row.Points,
			InitialValue: row.InitialValue, MinValue: row.MinValue, Decay: row.Decay, SolveCount: row.SolveCount,
			FlagHash: row.FlagHash, ConnectionInfo: row.ConnectionInfo, MaxAttempts: row.MaxAttempts, Position: row.Position, State: row.State,
			IsRegex: row.IsRegex, IsCaseInsensitive: row.IsCaseInsensitive, FlagRegex: row.FlagRegex, FlagFormatRegex: row.FlagFormatRegex,
			CreatedAt: pgutil.TimestamptzToTimeZero(row.CreatedAt), UpdatedAt: pgutil.TimestamptzToTimeZero(row.UpdatedAt),
		})
	}

	return out, nil
}

func (r *ChallengeRepo) listForTeamByTag(ctx context.Context, teamID, tagID uuid.UUID) ([]*repo.ChallengeWithSolved, error) {
	rows, err := r.Q(ctx).GetChallengesForTeamByTag(ctx, sqlc.GetChallengesForTeamByTagParams{TagID: tagID, TeamID: teamID})
	if err != nil {
		return nil, fmt.Errorf("ChallengeRepo - listForTeamByTag: %w", err)
	}

	out := make([]*repo.ChallengeWithSolved, 0, len(rows))
	for _, row := range rows {
		out = append(out, &repo.ChallengeWithSolved{
			Challenge: toDomainChallenge(challengeRow{
				ID: row.ID, Title: row.Title, Description: row.Description, Category: lo.EmptyableToPtr(row.Category), Points: row.Points,
				InitialValue: row.InitialValue, MinValue: row.MinValue, Decay: row.Decay, SolveCount: row.SolveCount,
				FlagHash: row.FlagHash, ConnectionInfo: row.ConnectionInfo, MaxAttempts: row.MaxAttempts, Position: row.Position, State: row.State, IsRegex: row.IsRegex, IsCaseInsensitive: row.IsCaseInsensitive,
				FlagRegex: row.FlagRegex, FlagFormatRegex: row.FlagFormatRegex,
				CreatedAt: pgutil.TimestamptzToTimeZero(row.CreatedAt), UpdatedAt: pgutil.TimestamptzToTimeZero(row.UpdatedAt),
			}),
			Solved: row.Solved == 1,
		})
	}

	return out, nil
}

func (r *ChallengeRepo) listByTag(ctx context.Context, tagID uuid.UUID) ([]*repo.ChallengeWithSolved, error) {
	rows, err := r.Q(ctx).GetChallengesByTag(ctx, tagID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeRepo - listByTag: %w", err)
	}

	out := make([]*repo.ChallengeWithSolved, 0, len(rows))
	for _, row := range rows {
		out = append(out, &repo.ChallengeWithSolved{
			Challenge: toDomainChallenge(challengeRow{
				ID: row.ID, Title: row.Title, Description: row.Description, Category: lo.EmptyableToPtr(row.Category), Points: row.Points,
				InitialValue: row.InitialValue, MinValue: row.MinValue, Decay: row.Decay, SolveCount: row.SolveCount,
				FlagHash: row.FlagHash, ConnectionInfo: row.ConnectionInfo, MaxAttempts: row.MaxAttempts, Position: row.Position, State: row.State, IsRegex: row.IsRegex, IsCaseInsensitive: row.IsCaseInsensitive,
				FlagRegex: row.FlagRegex, FlagFormatRegex: row.FlagFormatRegex,
				CreatedAt: pgutil.TimestamptzToTimeZero(row.CreatedAt), UpdatedAt: pgutil.TimestamptzToTimeZero(row.UpdatedAt),
			}),
			Solved: false,
		})
	}

	return out, nil
}

func (r *ChallengeRepo) listForTeam(ctx context.Context, teamID uuid.UUID) ([]*repo.ChallengeWithSolved, error) {
	rows, err := r.Q(ctx).GetChallengesForTeam(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeRepo - listForTeam: %w", err)
	}

	out := make([]*repo.ChallengeWithSolved, 0, len(rows))
	for _, row := range rows {
		out = append(out, &repo.ChallengeWithSolved{
			Challenge: toDomainChallenge(challengeRow{
				ID: row.ID, Title: row.Title, Description: row.Description, Category: lo.EmptyableToPtr(row.Category), Points: row.Points,
				InitialValue: row.InitialValue, MinValue: row.MinValue, Decay: row.Decay, SolveCount: row.SolveCount,
				FlagHash: row.FlagHash, ConnectionInfo: row.ConnectionInfo, MaxAttempts: row.MaxAttempts, Position: row.Position, State: row.State, IsRegex: row.IsRegex, IsCaseInsensitive: row.IsCaseInsensitive,
				FlagRegex: row.FlagRegex, FlagFormatRegex: row.FlagFormatRegex,
				CreatedAt: pgutil.TimestamptzToTimeZero(row.CreatedAt), UpdatedAt: pgutil.TimestamptzToTimeZero(row.UpdatedAt),
			}),
			Solved: row.Solved == 1,
		})
	}

	return out, nil
}

func (r *ChallengeRepo) listAllChallenges(ctx context.Context) ([]*repo.ChallengeWithSolved, error) {
	rows, err := r.Q(ctx).GetChallenges(ctx)
	if err != nil {
		return nil, fmt.Errorf("ChallengeRepo - listAllChallenges: %w", err)
	}

	out := make([]*repo.ChallengeWithSolved, 0, len(rows))
	for _, row := range rows {
		out = append(out, &repo.ChallengeWithSolved{
			Challenge: toDomainChallenge(challengeRow{
				ID: row.ID, Title: row.Title, Description: row.Description, Category: lo.EmptyableToPtr(row.Category), Points: row.Points,
				InitialValue: row.InitialValue, MinValue: row.MinValue, Decay: row.Decay, SolveCount: row.SolveCount,
				FlagHash: row.FlagHash, ConnectionInfo: row.ConnectionInfo, MaxAttempts: row.MaxAttempts, Position: row.Position, State: row.State, IsRegex: row.IsRegex, IsCaseInsensitive: row.IsCaseInsensitive,
				FlagRegex: row.FlagRegex, FlagFormatRegex: row.FlagFormatRegex,
				CreatedAt: pgutil.TimestamptzToTimeZero(row.CreatedAt), UpdatedAt: pgutil.TimestamptzToTimeZero(row.UpdatedAt),
			}),
			Solved: false,
		})
	}

	return out, nil
}

func (r *ChallengeRepo) listAllChallengesForBackup(ctx context.Context) ([]*repo.ChallengeWithSolved, error) {
	rows, err := r.Q(ctx).GetChallengesAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("ChallengeRepo - listAllChallengesForBackup: %w", err)
	}

	out := make([]*repo.ChallengeWithSolved, 0, len(rows))
	for _, row := range rows {
		out = append(out, &repo.ChallengeWithSolved{
			Challenge: toDomainChallenge(challengeRow{
				ID: row.ID, Title: row.Title, Description: row.Description, Category: lo.EmptyableToPtr(row.Category), Points: row.Points,
				InitialValue: row.InitialValue, MinValue: row.MinValue, Decay: row.Decay, SolveCount: row.SolveCount,
				FlagHash: row.FlagHash, ConnectionInfo: row.ConnectionInfo, MaxAttempts: row.MaxAttempts, Position: row.Position, State: row.State, IsRegex: row.IsRegex, IsCaseInsensitive: row.IsCaseInsensitive,
				FlagRegex: row.FlagRegex, FlagFormatRegex: row.FlagFormatRegex,
				CreatedAt: pgutil.TimestamptzToTimeZero(row.CreatedAt), UpdatedAt: pgutil.TimestamptzToTimeZero(row.UpdatedAt),
			}),
			Solved: false,
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

func (r *ChallengeRepo) GetAllForBackup(ctx context.Context) ([]*repo.ChallengeWithSolved, error) {
	return r.listAllChallengesForBackup(ctx)
}

func (r *ChallengeRepo) Delete(ctx context.Context, ID uuid.UUID) error {
	_, err := r.Q(ctx).DeleteChallenge(ctx, ID)
	if err != nil && !pgutil.IsNoRows(err) {
		return fmt.Errorf("ChallengeRepo - Delete: %w", err)
	}

	return nil
}

func (r *ChallengeRepo) IncrementSolveCount(ctx context.Context, ID uuid.UUID) (int, error) {
	n, err := r.Q(ctx).IncrementChallengeSolveCount(ctx, ID)
	if err != nil {
		return 0, fmt.Errorf("ChallengeRepo - IncrementSolveCount: %w", err)
	}

	return int(n), nil
}

func (r *ChallengeRepo) UpdatePoints(ctx context.Context, ID uuid.UUID, points int) error {
	pts, err := intToInt32Safe(points)
	if err != nil {
		return fmt.Errorf("ChallengeRepo - UpdatePoints: %w", err)
	}

	_, err = r.Q(ctx).UpdateChallengePoints(ctx, sqlc.UpdateChallengePointsParams{ID: ID, Points: &pts})
	if err != nil {
		if pgutil.IsNoRows(err) {
			return httperr.ErrChallengeNotFound
		}

		return fmt.Errorf("ChallengeRepo - UpdatePoints: %w", err)
	}

	return nil
}

func (r *ChallengeRepo) GetFlags(ctx context.Context, ID uuid.UUID) (*repo.ChallengeFlags, error) {
	row, err := r.Q(ctx).GetChallengeFlags(ctx, ID)
	if err != nil {
		if pgutil.IsNoRows(err) {
			return nil, httperr.ErrChallengeNotFound
		}

		return nil, fmt.Errorf("ChallengeRepo - GetFlags: %w", err)
	}

	return &repo.ChallengeFlags{
		FlagHash:          row.FlagHash,
		IsRegex:           lo.FromPtr(row.IsRegex),
		IsCaseInsensitive: lo.FromPtr(row.IsCaseInsensitive),
		FlagRegex:         row.FlagRegex,
		FlagFormatRegex:   row.FlagFormatRegex,
	}, nil
}

func (r *ChallengeRepo) GetRequirements(ctx context.Context, ID uuid.UUID) ([]*repo.ChallengeRequirement, error) {
	rows, err := r.Q(ctx).GetChallengeRequirements(ctx, ID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeRepo - GetRequirements: %w", err)
	}

	out := make([]*repo.ChallengeRequirement, len(rows))
	for i, row := range rows {
		out[i] = &repo.ChallengeRequirement{
			ChallengeID:    row.ID,
			ChallengeTitle: row.Title,
			Category:       lo.EmptyableToPtr(row.Category),
		}
	}

	return out, nil
}

func (r *ChallengeRepo) GetAllRequirementPairs(ctx context.Context) ([]*domain.ChallengeRequirementPair, error) {
	rows, err := r.Q(ctx).GetAllChallengeRequirements(ctx)
	if err != nil {
		return nil, fmt.Errorf("ChallengeRepo - GetAllRequirementPairs: %w", err)
	}

	out := make([]*domain.ChallengeRequirementPair, len(rows))
	for i, row := range rows {
		out[i] = &domain.ChallengeRequirementPair{
			ChallengeID:         row.ChallengeID,
			RequiredChallengeID: row.RequiredChallengeID,
		}
	}

	return out, nil
}

func (r *ChallengeRepo) SetRequirements(ctx context.Context, challengeID uuid.UUID, requirementIDs []uuid.UUID) error {
	if err := r.Q(ctx).DeleteChallengeRequirements(ctx, challengeID); err != nil {
		return fmt.Errorf("ChallengeRepo - SetRequirements - Delete: %w", err)
	}

	if len(requirementIDs) == 0 {
		return nil
	}

	qb := SB.Insert("challenge_requirements").
		Columns("challenge_id", "required_challenge_id").
		Suffix("ON CONFLICT (challenge_id, required_challenge_id) DO NOTHING")

	for _, reqID := range requirementIDs {
		qb = qb.Values(challengeID, reqID)
	}

	query, args, err := qb.ToSql()
	if err != nil {
		return fmt.Errorf("ChallengeRepo - SetRequirements - ToSql: %w", err)
	}

	if _, err := r.DB(ctx).Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("ChallengeRepo - SetRequirements - Exec: %w", err)
	}

	return nil
}

func (r *ChallengeRepo) GetSolution(ctx context.Context, ID uuid.UUID) (*repo.ChallengeSolution, error) {
	row, err := r.Q(ctx).GetSolutionByChallengeID(ctx, ID)
	if err != nil {
		if pgutil.IsNoRows(err) {
			return nil, httperr.ErrChallengeNotFound
		}

		return nil, fmt.Errorf("ChallengeRepo - GetSolution: %w", err)
	}

	files, err := r.Q(ctx).GetFilesByChallengeIDAndType(ctx, sqlc.GetFilesByChallengeIDAndTypeParams{
		ChallengeID: ID,
		Type:        string(domain.FileTypeWriteup),
	})
	if err != nil {
		return nil, fmt.Errorf("ChallengeRepo - GetSolution - GetFiles: %w", err)
	}

	entityFiles := make([]*domain.File, 0, len(files))
	for _, f := range files {
		entityFiles = append(entityFiles, &domain.File{
			ID:          f.ID,
			Type:        domain.FileType(f.Type),
			ChallengeID: f.ChallengeID,
			Location:    f.Location,
			Filename:    f.Filename,
			Size:        f.Size,
			SHA256:      f.SHA256,
			CreatedAt:   pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(f.CreatedAt)),
		})
	}

	return &repo.ChallengeSolution{
		ChallengeID: row.ChallengeID,
		Content:     row.Content,
		Files:       entityFiles,
	}, nil
}

func (r *ChallengeRepo) GetAllSolutions(ctx context.Context) ([]*domain.SolutionBackup, error) {
	rows, err := r.Q(ctx).GetAllSolutions(ctx)
	if err != nil {
		return nil, fmt.Errorf("ChallengeRepo - GetAllSolutions: %w", err)
	}

	out := make([]*domain.SolutionBackup, len(rows))
	for i, row := range rows {
		out[i] = &domain.SolutionBackup{
			ID:          row.ID,
			ChallengeID: row.ChallengeID,
			Content:     row.Content,
		}
	}

	return out, nil
}

func (r *ChallengeRepo) ListSolutions(ctx context.Context, teamID uuid.UUID) ([]*repo.ChallengeSolutionEntry, error) {
	rows, err := r.Q(ctx).GetSolutionsByTeamID(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeRepo - ListSolutions: %w", err)
	}

	if len(rows) == 0 {
		return []*repo.ChallengeSolutionEntry{}, nil
	}

	ids := make([]uuid.UUID, len(rows))
	for i, row := range rows {
		ids[i] = row.ChallengeID
	}

	files, err := r.Q(ctx).GetWriteupFilesByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("ChallengeRepo - ListSolutions - ListWriteupFilesByIDs: %w", err)
	}

	fileMap := make(map[uuid.UUID][]*domain.File, len(files))
	for _, f := range files {
		fileMap[f.ChallengeID] = append(fileMap[f.ChallengeID], &domain.File{
			ID:          f.ID,
			Type:        domain.FileType(f.Type),
			ChallengeID: f.ChallengeID,
			Location:    f.Location,
			Filename:    f.Filename,
			Size:        f.Size,
			SHA256:      f.SHA256,
			CreatedAt:   pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(f.CreatedAt)),
		})
	}

	out := make([]*repo.ChallengeSolutionEntry, len(rows))
	for i, row := range rows {
		ef := fileMap[row.ChallengeID]
		if ef == nil {
			ef = []*domain.File{}
		}

		out[i] = &repo.ChallengeSolutionEntry{
			ChallengeID:       row.ChallengeID,
			ChallengeTitle:    row.ChallengeTitle,
			ChallengeCategory: row.ChallengeCategory,
			Content:           row.Content,
			Files:             ef,
		}
	}

	return out, nil
}

func (r *ChallengeRepo) UpsertSolution(ctx context.Context, challengeID uuid.UUID, content string) (*repo.ChallengeSolution, error) {
	row, err := r.Q(ctx).UpsertSolution(ctx, sqlc.UpsertSolutionParams{
		ChallengeID: challengeID,
		Content:     content,
	})
	if err != nil {
		return nil, fmt.Errorf("ChallengeRepo - UpsertSolution: %w", err)
	}

	files, err := r.Q(ctx).GetFilesByChallengeIDAndType(ctx, sqlc.GetFilesByChallengeIDAndTypeParams{
		ChallengeID: challengeID,
		Type:        string(domain.FileTypeWriteup),
	})
	if err != nil {
		return nil, fmt.Errorf("ChallengeRepo - UpsertSolution - GetFiles: %w", err)
	}

	entityFiles := make([]*domain.File, 0, len(files))
	for _, f := range files {
		entityFiles = append(entityFiles, &domain.File{
			ID:          f.ID,
			Type:        domain.FileType(f.Type),
			ChallengeID: f.ChallengeID,
			Location:    f.Location,
			Filename:    f.Filename,
			Size:        f.Size,
			SHA256:      f.SHA256,
			CreatedAt:   pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(f.CreatedAt)),
		})
	}

	return &repo.ChallengeSolution{
		ChallengeID: row.ChallengeID,
		Content:     row.Content,
		Files:       entityFiles,
	}, nil
}

func (r *ChallengeRepo) DeleteSolution(ctx context.Context, challengeID uuid.UUID) error {
	err := r.Q(ctx).DeleteSolution(ctx, challengeID)
	if err != nil {
		return fmt.Errorf("ChallengeRepo - DeleteSolution: %w", err)
	}

	return nil
}

func (r *ChallengeRepo) GetMissingChallengesByTeamID(ctx context.Context, teamID uuid.UUID) ([]*domain.Challenge, error) {
	rows, err := r.Q(ctx).GetMissingChallengesByTeamID(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeRepo - GetMissingChallengesByTeamID: %w", err)
	}

	out := make([]*domain.Challenge, 0, len(rows))
	for _, row := range rows {
		out = append(out, toDomainChallenge(challengeRow{
			ID: row.ID, Title: row.Title, Description: row.Description, Category: lo.EmptyableToPtr(row.Category), Points: row.Points,
			InitialValue: row.InitialValue, MinValue: row.MinValue, Decay: row.Decay, SolveCount: row.SolveCount,
			FlagHash: row.FlagHash, ConnectionInfo: row.ConnectionInfo, MaxAttempts: row.MaxAttempts, Position: row.Position, State: row.State, IsRegex: row.IsRegex, IsCaseInsensitive: row.IsCaseInsensitive,
			FlagRegex: row.FlagRegex, FlagFormatRegex: row.FlagFormatRegex,
			CreatedAt: pgutil.TimestamptzToTimeZero(row.CreatedAt), UpdatedAt: pgutil.TimestamptzToTimeZero(row.UpdatedAt),
		}))
	}

	return out, nil
}

func (r *ChallengeRepo) GetMissingChallengesByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Challenge, error) {
	rows, err := r.Q(ctx).GetMissingChallengesByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeRepo - GetMissingChallengesByUserID: %w", err)
	}

	out := make([]*domain.Challenge, 0, len(rows))
	for _, row := range rows {
		out = append(out, toDomainChallenge(challengeRow{
			ID: row.ID, Title: row.Title, Description: row.Description, Category: lo.EmptyableToPtr(row.Category), Points: row.Points,
			InitialValue: row.InitialValue, MinValue: row.MinValue, Decay: row.Decay, SolveCount: row.SolveCount,
			FlagHash: row.FlagHash, ConnectionInfo: row.ConnectionInfo, MaxAttempts: row.MaxAttempts, Position: row.Position, State: row.State, IsRegex: row.IsRegex, IsCaseInsensitive: row.IsCaseInsensitive,
			FlagRegex: row.FlagRegex, FlagFormatRegex: row.FlagFormatRegex,
			CreatedAt: pgutil.TimestamptzToTimeZero(row.CreatedAt), UpdatedAt: pgutil.TimestamptzToTimeZero(row.UpdatedAt),
		}))
	}

	return out, nil
}

func (r *ChallengeRepo) Create(ctx context.Context, c *domain.Challenge) error {
	pts, err := intToInt32Safe(c.Points)
	if err != nil {
		return fmt.Errorf("ChallengeRepo - Create - Points: %w", err)
	}

	initialValue, err := intToInt32Safe(c.InitialValue)
	if err != nil {
		return fmt.Errorf("ChallengeRepo - Create - InitialValue: %w", err)
	}

	minValue, err := intToInt32Safe(c.MinValue)
	if err != nil {
		return fmt.Errorf("ChallengeRepo - Create - MinValue: %w", err)
	}

	decay, err := intToInt32Safe(c.Decay)
	if err != nil {
		return fmt.Errorf("ChallengeRepo - Create - Decay: %w", err)
	}

	solveCount, err := intToInt32Safe(c.SolveCount)
	if err != nil {
		return fmt.Errorf("ChallengeRepo - Create - SolveCount: %w", err)
	}

	maxAttempts, err := intToInt32Safe(c.MaxAttempts)
	if err != nil {
		return fmt.Errorf("ChallengeRepo - Create - MaxAttempts: %w", err)
	}

	position, err := intToInt32Safe(c.Position)
	if err != nil {
		return fmt.Errorf("ChallengeRepo - Create - Position: %w", err)
	}

	state := c.State
	if state == "" {
		state = domain.ChallengeStateHidden
	}

	now := time.Now()
	if err := r.Q(ctx).CreateChallenge(ctx, sqlc.CreateChallengeParams{
		ID:                c.ID,
		Title:             c.Title,
		Description:       c.Description,
		Category:          c.Category,
		Points:            &pts,
		InitialValue:      initialValue,
		MinValue:          minValue,
		Decay:             decay,
		SolveCount:        solveCount,
		FlagHash:          c.FlagHash,
		ConnectionInfo:    c.ConnectionInfo,
		MaxAttempts:       maxAttempts,
		Position:          position,
		State:             state,
		IsRegex:           &c.IsRegex,
		IsCaseInsensitive: &c.IsCaseInsensitive,
		FlagRegex:         c.FlagRegex,
		FlagFormatRegex:   c.FlagFormatRegex,
		CreatedAt:         pgutil.TimeToTimestamptz(&now),
		UpdatedAt:         pgutil.TimeToTimestamptz(&now),
	}); err != nil {
		return fmt.Errorf("ChallengeRepo - Create: %w", err)
	}

	return nil
}

func (r *ChallengeRepo) Update(ctx context.Context, c *domain.Challenge) error {
	pts, err := intToInt32Safe(c.Points)
	if err != nil {
		return fmt.Errorf("ChallengeRepo - Update - Points: %w", err)
	}

	initialValue, err := intToInt32Safe(c.InitialValue)
	if err != nil {
		return fmt.Errorf("ChallengeRepo - Update - InitialValue: %w", err)
	}

	minValue, err := intToInt32Safe(c.MinValue)
	if err != nil {
		return fmt.Errorf("ChallengeRepo - Update - MinValue: %w", err)
	}

	decay, err := intToInt32Safe(c.Decay)
	if err != nil {
		return fmt.Errorf("ChallengeRepo - Update - Decay: %w", err)
	}

	maxAttempts, err := intToInt32Safe(c.MaxAttempts)
	if err != nil {
		return fmt.Errorf("ChallengeRepo - Update - MaxAttempts: %w", err)
	}

	position, err := intToInt32Safe(c.Position)
	if err != nil {
		return fmt.Errorf("ChallengeRepo - Update - Position: %w", err)
	}

	state := c.State
	if state == "" {
		state = domain.ChallengeStateHidden
	}

	c.UpdatedAt = time.Now()
	if err := r.Q(ctx).UpdateChallenge(ctx, sqlc.UpdateChallengeParams{
		ID:                c.ID,
		Title:             c.Title,
		Description:       c.Description,
		Category:          c.Category,
		Points:            &pts,
		InitialValue:      initialValue,
		MinValue:          minValue,
		Decay:             decay,
		FlagHash:          c.FlagHash,
		ConnectionInfo:    c.ConnectionInfo,
		MaxAttempts:       maxAttempts,
		Position:          position,
		State:             state,
		IsRegex:           &c.IsRegex,
		IsCaseInsensitive: &c.IsCaseInsensitive,
		FlagRegex:         c.FlagRegex,
		FlagFormatRegex:   c.FlagFormatRegex,
		UpdatedAt:         pgutil.TimeToTimestamptz(&c.UpdatedAt),
	}); err != nil {
		return fmt.Errorf("ChallengeRepo - Update: %w", err)
	}

	return nil
}

func (r *ChallengeRepo) GetByIDForUpdate(ctx context.Context, ID uuid.UUID) (*domain.Challenge, error) {
	row, err := r.Q(ctx).GetChallengeByIDForUpdate(ctx, ID)
	if err != nil {
		if pgutil.IsNoRows(err) {
			return nil, httperr.ErrChallengeNotFound
		}

		return nil, fmt.Errorf("ChallengeRepo - GetByIDForUpdate: %w", err)
	}

	return toDomainChallenge(challengeRow{
		ID: row.ID, Title: row.Title, Description: row.Description, Category: lo.EmptyableToPtr(row.Category), Points: row.Points,
		InitialValue: row.InitialValue, MinValue: row.MinValue, Decay: row.Decay, SolveCount: row.SolveCount,
		FlagHash: row.FlagHash, ConnectionInfo: row.ConnectionInfo, MaxAttempts: row.MaxAttempts, Position: row.Position, State: row.State, IsRegex: row.IsRegex, IsCaseInsensitive: row.IsCaseInsensitive,
		FlagRegex: row.FlagRegex, FlagFormatRegex: row.FlagFormatRegex,
		CreatedAt: pgutil.TimestamptzToTimeZero(row.CreatedAt), UpdatedAt: pgutil.TimestamptzToTimeZero(row.UpdatedAt),
	}), nil
}

func (r *ChallengeRepo) DecrementSolveCount(ctx context.Context, ID uuid.UUID) (int, error) {
	n, err := r.Q(ctx).DecrementChallengeSolveCount(ctx, ID)
	if err != nil {
		if pgutil.IsNoRows(err) {
			return 0, httperr.ErrChallengeNotFound
		}

		return 0, fmt.Errorf("ChallengeRepo - DecrementSolveCount: %w", err)
	}

	return int(n), nil
}

func (r *ChallengeRepo) BatchDecrementSolveCount(ctx context.Context, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}

	err := r.Q(ctx).BatchDecrementChallengeSolveCount(ctx, ids)
	if err != nil {
		return fmt.Errorf("ChallengeRepo - BatchDecrementSolveCount: %w", err)
	}

	return nil
}

func (r *ChallengeRepo) BatchIncrementSolveCount(ctx context.Context, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}

	err := r.Q(ctx).BatchIncrementChallengeSolveCount(ctx, ids)
	if err != nil {
		return fmt.Errorf("ChallengeRepo - BatchIncrementSolveCount: %w", err)
	}

	return nil
}

func (r *ChallengeRepo) BatchUpdatePoints(ctx context.Context, ids []uuid.UUID, points []int) error {
	if len(ids) == 0 {
		return nil
	}

	if len(ids) != len(points) {
		return fmt.Errorf("ChallengeRepo - BatchUpdatePoints: ids and points length mismatch (%d != %d)", len(ids), len(points))
	}

	pts := make([]int32, len(points))
	for i, p := range points {
		v, err := intToInt32Safe(p)
		if err != nil {
			return fmt.Errorf("ChallengeRepo - BatchUpdatePoints: %w", err)
		}

		pts[i] = v
	}

	err := r.Q(ctx).BatchUpdateChallengePoints(ctx, sqlc.BatchUpdateChallengePointsParams{Column1: ids, Column2: pts})
	if err != nil {
		return fmt.Errorf("ChallengeRepo - BatchUpdatePoints: %w", err)
	}

	return nil
}

func (r *ChallengeRepo) SetTags(ctx context.Context, challengeID uuid.UUID, tagIDs []uuid.UUID) error {
	if err := r.Q(ctx).DeleteChallengeTags(ctx, challengeID); err != nil {
		return fmt.Errorf("ChallengeRepo - SetTags - Delete: %w", err)
	}

	if len(tagIDs) == 0 {
		return nil
	}

	qb := SB.Insert("challenge_tags").
		Columns("challenge_id", "tag_id").
		Suffix("ON CONFLICT (challenge_id, tag_id) DO NOTHING")

	for _, tagID := range tagIDs {
		qb = qb.Values(challengeID, tagID)
	}

	query, args, err := qb.ToSql()
	if err != nil {
		return fmt.Errorf("ChallengeRepo - SetTags - ToSql: %w", err)
	}

	if _, err := r.DB(ctx).Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("ChallengeRepo - SetTags - Exec: %w", err)
	}

	return nil
}
