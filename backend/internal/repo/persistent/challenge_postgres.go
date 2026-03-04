package persistent

import (
	"context"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ChallengeRepo struct {
	pool *pgxpool.Pool
}

var _ repo.ChallengeRepository = (*ChallengeRepo)(nil)

func NewChallengeRepo(pool *pgxpool.Pool) *ChallengeRepo {
	return &ChallengeRepo{pool: pool}
}

func (r *ChallengeRepo) q(ctx context.Context) *sqlc.Queries {
	return sqlc.New(ExtractDB(ctx, r.pool))
}

func toEntityChallenge(ID uuid.UUID, title, description string, category *string, points *int32, initialValue, minValue, decay, solveCount int32, flagHash string, isHidden, isRegex, isCaseInsensitive *bool, flagRegex, flagFormatRegex *string) *entity.Challenge {
	var pts int
	if points != nil {
		pts = int(*points)
	}
	return &entity.Challenge{
		ID:                ID,
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

func (r *ChallengeRepo) GetByID(ctx context.Context, ID uuid.UUID) (*entity.Challenge, error) {
	row, err := r.q(ctx).GetChallengeByID(ctx, ID)
	if err != nil {
		if isNoRows(err) {
			return nil, httperr.ErrChallengeNotFound
		}
		return nil, fmt.Errorf("ChallengeRepo - GetByID: %w", err)
	}
	return toEntityChallenge(row.ID, row.Title, row.Description, row.Category, row.Points, row.InitialValue, row.MinValue, row.Decay, row.SolveCount, row.FlagHash, row.IsHidden, row.IsRegex, row.IsCaseInsensitive, row.FlagRegex, row.FlagFormatRegex), nil
}

func (r *ChallengeRepo) GetByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*entity.Challenge, error) {
	if len(ids) == 0 {
		return map[uuid.UUID]*entity.Challenge{}, nil
	}
	rows, err := r.q(ctx).GetChallengesByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("ChallengeRepo - GetByIDs: %w", err)
	}
	out := make(map[uuid.UUID]*entity.Challenge, len(rows))
	for _, row := range rows {
		out[row.ID] = toEntityChallenge(row.ID, row.Title, row.Description, row.Category, row.Points, row.InitialValue, row.MinValue, row.Decay, row.SolveCount, row.FlagHash, row.IsHidden, row.IsRegex, row.IsCaseInsensitive, row.FlagRegex, row.FlagFormatRegex)
	}
	return out, nil
}

func (r *ChallengeRepo) listForTeamByTag(ctx context.Context, teamID, tagID uuid.UUID) ([]*repo.ChallengeWithSolved, error) {
	rows, err := r.q(ctx).GetChallengesForTeamByTag(ctx, sqlc.GetChallengesForTeamByTagParams{TagID: tagID, TeamID: teamID})
	if err != nil {
		return nil, fmt.Errorf("ChallengeRepo - listForTeamByTag: %w", err)
	}
	out := make([]*repo.ChallengeWithSolved, 0, len(rows))
	for _, row := range rows {
		out = append(out, &repo.ChallengeWithSolved{
			Challenge: toEntityChallenge(row.ID, row.Title, row.Description, row.Category, row.Points, row.InitialValue, row.MinValue, row.Decay, row.SolveCount, row.FlagHash, row.IsHidden, row.IsRegex, row.IsCaseInsensitive, row.FlagRegex, row.FlagFormatRegex),
			Solved:    row.Solved == 1,
		})
	}
	return out, nil
}

func (r *ChallengeRepo) listByTag(ctx context.Context, tagID uuid.UUID) ([]*repo.ChallengeWithSolved, error) {
	rows, err := r.q(ctx).GetChallengesByTag(ctx, tagID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeRepo - listByTag: %w", err)
	}
	out := make([]*repo.ChallengeWithSolved, 0, len(rows))
	for _, row := range rows {
		out = append(out, &repo.ChallengeWithSolved{
			Challenge: toEntityChallenge(row.ID, row.Title, row.Description, row.Category, row.Points, row.InitialValue, row.MinValue, row.Decay, row.SolveCount, row.FlagHash, row.IsHidden, row.IsRegex, row.IsCaseInsensitive, row.FlagRegex, row.FlagFormatRegex),
			Solved:    false,
		})
	}
	return out, nil
}

func (r *ChallengeRepo) listForTeam(ctx context.Context, teamID uuid.UUID) ([]*repo.ChallengeWithSolved, error) {
	rows, err := r.q(ctx).GetChallengesForTeam(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeRepo - listForTeam: %w", err)
	}
	out := make([]*repo.ChallengeWithSolved, 0, len(rows))
	for _, row := range rows {
		out = append(out, &repo.ChallengeWithSolved{
			Challenge: toEntityChallenge(row.ID, row.Title, row.Description, row.Category, row.Points, row.InitialValue, row.MinValue, row.Decay, row.SolveCount, row.FlagHash, row.IsHidden, row.IsRegex, row.IsCaseInsensitive, row.FlagRegex, row.FlagFormatRegex),
			Solved:    row.Solved == 1,
		})
	}
	return out, nil
}

func (r *ChallengeRepo) listAllChallenges(ctx context.Context) ([]*repo.ChallengeWithSolved, error) {
	rows, err := r.q(ctx).GetChallenges(ctx)
	if err != nil {
		return nil, fmt.Errorf("ChallengeRepo - listAllChallenges: %w", err)
	}
	out := make([]*repo.ChallengeWithSolved, 0, len(rows))
	for _, row := range rows {
		out = append(out, &repo.ChallengeWithSolved{
			Challenge: toEntityChallenge(row.ID, row.Title, row.Description, row.Category, row.Points, row.InitialValue, row.MinValue, row.Decay, row.SolveCount, row.FlagHash, row.IsHidden, row.IsRegex, row.IsCaseInsensitive, row.FlagRegex, row.FlagFormatRegex),
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

func (r *ChallengeRepo) Delete(ctx context.Context, ID uuid.UUID) error {
	_, err := r.q(ctx).DeleteChallenge(ctx, ID)
	if err != nil && !isNoRows(err) {
		return fmt.Errorf("ChallengeRepo - Delete: %w", err)
	}
	return nil
}

func (r *ChallengeRepo) IncrementSolveCount(ctx context.Context, ID uuid.UUID) (int, error) {
	n, err := r.q(ctx).IncrementChallengeSolveCount(ctx, ID)
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
	_, err = r.q(ctx).UpdateChallengePoints(ctx, sqlc.UpdateChallengePointsParams{ID: ID, Points: &pts})
	if err != nil {
		if isNoRows(err) {
			return httperr.ErrChallengeNotFound
		}
		return fmt.Errorf("ChallengeRepo - UpdatePoints: %w", err)
	}
	return nil
}

func (r *ChallengeRepo) GetFlags(ctx context.Context, ID uuid.UUID) (*repo.ChallengeFlags, error) {
	row, err := r.q(ctx).GetChallengeFlags(ctx, ID)
	if err != nil {
		if isNoRows(err) {
			return nil, httperr.ErrChallengeNotFound
		}
		return nil, fmt.Errorf("ChallengeRepo - GetFlags: %w", err)
	}
	return &repo.ChallengeFlags{
		FlagHash:          row.FlagHash,
		IsRegex:           boolPtrToBool(row.IsRegex),
		IsCaseInsensitive: boolPtrToBool(row.IsCaseInsensitive),
		FlagRegex:         row.FlagRegex,
		FlagFormatRegex:   row.FlagFormatRegex,
	}, nil
}

func (r *ChallengeRepo) GetRequirements(ctx context.Context, ID uuid.UUID) ([]*repo.ChallengeRequirement, error) {
	rows, err := r.q(ctx).GetChallengeRequirements(ctx, ID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeRepo - GetRequirements: %w", err)
	}
	out := make([]*repo.ChallengeRequirement, len(rows))
	for i, row := range rows {
		out[i] = &repo.ChallengeRequirement{
			ChallengeID:    row.ID,
			ChallengeTitle: row.Title,
			Category:       row.Category,
		}
	}
	return out, nil
}

func (r *ChallengeRepo) SetRequirements(ctx context.Context, challengeID uuid.UUID, requirementIDs []uuid.UUID) error {
	if err := r.q(ctx).DeleteChallengeRequirements(ctx, challengeID); err != nil {
		return fmt.Errorf("ChallengeRepo - SetRequirements - Delete: %w", err)
	}
	if len(requirementIDs) == 0 {
		return nil
	}
	qb := squirrel.Insert("challenge_requirements").
		Columns("challenge_id", "required_challenge_id").
		Suffix("ON CONFLICT (challenge_id, required_challenge_id) DO NOTHING").
		PlaceholderFormat(squirrel.Dollar)
	for _, reqID := range requirementIDs {
		qb = qb.Values(challengeID, reqID)
	}
	query, args, err := qb.ToSql()
	if err != nil {
		return fmt.Errorf("ChallengeRepo - SetRequirements - ToSql: %w", err)
	}
	if _, err := ExtractDB(ctx, r.pool).Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("ChallengeRepo - SetRequirements - Exec: %w", err)
	}
	return nil
}

func (r *ChallengeRepo) GetSolution(ctx context.Context, ID uuid.UUID) (*repo.ChallengeSolution, error) {
	row, err := r.q(ctx).GetSolutionByChallengeID(ctx, ID)
	if err != nil {
		if isNoRows(err) {
			return nil, httperr.ErrChallengeNotFound
		}
		return nil, fmt.Errorf("ChallengeRepo - GetSolution: %w", err)
	}
	files, err := r.q(ctx).GetFilesByChallengeIDAndType(ctx, sqlc.GetFilesByChallengeIDAndTypeParams{
		ChallengeID: ID,
		Type:        string(entity.FileTypeWriteup),
	})
	if err != nil {
		return nil, fmt.Errorf("ChallengeRepo - GetSolution - GetFiles: %w", err)
	}
	entityFiles := make([]*entity.File, 0, len(files))
	for _, f := range files {
		entityFiles = append(entityFiles, &entity.File{
			ID:          f.ID,
			Type:        entity.FileType(f.Type),
			ChallengeID: f.ChallengeID,
			Location:    f.Location,
			Filename:    f.Filename,
			Size:        f.Size,
			SHA256:      f.SHA256,
			CreatedAt:   f.CreatedAt,
		})
	}
	return &repo.ChallengeSolution{
		ChallengeID: row.ChallengeID,
		Content:     row.Content,
		Files:       entityFiles,
	}, nil
}

func (r *ChallengeRepo) ListSolutions(ctx context.Context, teamID uuid.UUID) ([]*repo.ChallengeSolutionEntry, error) {
	rows, err := r.q(ctx).GetSolutionsByTeamID(ctx, teamID)
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

	files, err := r.q(ctx).GetWriteupFilesByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("ChallengeRepo - ListSolutions - ListWriteupFilesByIDs: %w", err)
	}

	fileMap := make(map[uuid.UUID][]*entity.File, len(files))
	for _, f := range files {
		fileMap[f.ChallengeID] = append(fileMap[f.ChallengeID], &entity.File{
			ID:          f.ID,
			Type:        entity.FileType(f.Type),
			ChallengeID: f.ChallengeID,
			Location:    f.Location,
			Filename:    f.Filename,
			Size:        f.Size,
			SHA256:      f.SHA256,
			CreatedAt:   f.CreatedAt,
		})
	}

	out := make([]*repo.ChallengeSolutionEntry, len(rows))
	for i, row := range rows {
		ef := fileMap[row.ChallengeID]
		if ef == nil {
			ef = []*entity.File{}
		}
		cat := ""
		if row.ChallengeCategory != nil {
			cat = *row.ChallengeCategory
		}
		out[i] = &repo.ChallengeSolutionEntry{
			ChallengeID:       row.ChallengeID,
			ChallengeTitle:    row.ChallengeTitle,
			ChallengeCategory: cat,
			Content:           row.Content,
			Files:             ef,
		}
	}
	return out, nil
}

func (r *ChallengeRepo) UpsertSolution(ctx context.Context, challengeID uuid.UUID, content string) (*repo.ChallengeSolution, error) {
	row, err := r.q(ctx).UpsertSolution(ctx, sqlc.UpsertSolutionParams{
		ChallengeID: challengeID,
		Content:     content,
	})
	if err != nil {
		return nil, fmt.Errorf("ChallengeRepo - UpsertSolution: %w", err)
	}
	files, err := r.q(ctx).GetFilesByChallengeIDAndType(ctx, sqlc.GetFilesByChallengeIDAndTypeParams{
		ChallengeID: challengeID,
		Type:        string(entity.FileTypeWriteup),
	})
	if err != nil {
		return nil, fmt.Errorf("ChallengeRepo - UpsertSolution - GetFiles: %w", err)
	}
	entityFiles := make([]*entity.File, 0, len(files))
	for _, f := range files {
		entityFiles = append(entityFiles, &entity.File{
			ID:          f.ID,
			Type:        entity.FileType(f.Type),
			ChallengeID: f.ChallengeID,
			Location:    f.Location,
			Filename:    f.Filename,
			Size:        f.Size,
			SHA256:      f.SHA256,
			CreatedAt:   f.CreatedAt,
		})
	}
	return &repo.ChallengeSolution{
		ChallengeID: row.ChallengeID,
		Content:     row.Content,
		Files:       entityFiles,
	}, nil
}

func (r *ChallengeRepo) DeleteSolution(ctx context.Context, challengeID uuid.UUID) error {
	if err := r.q(ctx).DeleteSolution(ctx, challengeID); err != nil {
		return fmt.Errorf("ChallengeRepo - DeleteSolution: %w", err)
	}
	return nil
}

func (r *ChallengeRepo) GetMissingChallengesByTeamID(ctx context.Context, teamID uuid.UUID) ([]*entity.Challenge, error) {
	rows, err := r.q(ctx).GetMissingChallengesByTeamID(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeRepo - GetMissingChallengesByTeamID: %w", err)
	}
	out := make([]*entity.Challenge, 0, len(rows))
	for _, row := range rows {
		out = append(out, toEntityChallenge(row.ID, row.Title, row.Description, row.Category, row.Points, row.InitialValue, row.MinValue, row.Decay, row.SolveCount, row.FlagHash, row.IsHidden, row.IsRegex, row.IsCaseInsensitive, row.FlagRegex, row.FlagFormatRegex))
	}
	return out, nil
}

func (r *ChallengeRepo) GetMissingChallengesByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.Challenge, error) {
	rows, err := r.q(ctx).GetMissingChallengesByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeRepo - GetMissingChallengesByUserID: %w", err)
	}
	out := make([]*entity.Challenge, 0, len(rows))
	for _, row := range rows {
		out = append(out, toEntityChallenge(row.ID, row.Title, row.Description, row.Category, row.Points, row.InitialValue, row.MinValue, row.Decay, row.SolveCount, row.FlagHash, row.IsHidden, row.IsRegex, row.IsCaseInsensitive, row.FlagRegex, row.FlagFormatRegex))
	}
	return out, nil
}

func (r *ChallengeRepo) Create(ctx context.Context, c *entity.Challenge) error {
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
	if err := r.q(ctx).CreateChallenge(ctx, sqlc.CreateChallengeParams{
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
	}); err != nil {
		return fmt.Errorf("ChallengeRepo - Create: %w", err)
	}
	return nil
}

func (r *ChallengeRepo) Update(ctx context.Context, c *entity.Challenge) error {
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
	if err := r.q(ctx).UpdateChallenge(ctx, sqlc.UpdateChallengeParams{
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
	}); err != nil {
		return fmt.Errorf("ChallengeRepo - Update: %w", err)
	}
	return nil
}

func (r *ChallengeRepo) GetByIDForUpdate(ctx context.Context, ID uuid.UUID) (*entity.Challenge, error) {
	row, err := r.q(ctx).GetChallengeByIDForUpdate(ctx, ID)
	if err != nil {
		if isNoRows(err) {
			return nil, httperr.ErrChallengeNotFound
		}
		return nil, fmt.Errorf("ChallengeRepo - GetByIDForUpdate: %w", err)
	}
	return toEntityChallenge(row.ID, row.Title, row.Description, row.Category, row.Points, row.InitialValue, row.MinValue, row.Decay, row.SolveCount, row.FlagHash, row.IsHidden, row.IsRegex, row.IsCaseInsensitive, row.FlagRegex, row.FlagFormatRegex), nil
}

func (r *ChallengeRepo) DecrementSolveCount(ctx context.Context, ID uuid.UUID) (int, error) {
	n, err := r.q(ctx).DecrementChallengeSolveCount(ctx, ID)
	if err != nil {
		if isNoRows(err) {
			return 0, httperr.ErrChallengeNotFound
		}
		return 0, fmt.Errorf("ChallengeRepo - DecrementSolveCount: %w", err)
	}
	return int(n), nil
}

func (r *ChallengeRepo) SetTags(ctx context.Context, challengeID uuid.UUID, tagIDs []uuid.UUID) error {
	if err := r.q(ctx).DeleteChallengeTags(ctx, challengeID); err != nil {
		return fmt.Errorf("ChallengeRepo - SetTags - Delete: %w", err)
	}
	if len(tagIDs) == 0 {
		return nil
	}
	qb := squirrel.Insert("challenge_tags").
		Columns("challenge_id", "tag_id").
		Suffix("ON CONFLICT (challenge_id, tag_id) DO NOTHING").
		PlaceholderFormat(squirrel.Dollar)
	for _, tagID := range tagIDs {
		qb = qb.Values(challengeID, tagID)
	}
	query, args, err := qb.ToSql()
	if err != nil {
		return fmt.Errorf("ChallengeRepo - SetTags - ToSql: %w", err)
	}
	if _, err := ExtractDB(ctx, r.pool).Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("ChallengeRepo - SetTags - Exec: %w", err)
	}
	return nil
}
