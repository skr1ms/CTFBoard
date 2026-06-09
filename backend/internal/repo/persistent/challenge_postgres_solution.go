package persistent

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-pgkit/pgutil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
)

// GetSolution fetches the solution text for a challenge and its associated
// writeup files in two sequential queries, then assembles a ChallengeSolution.
func (r *ChallengeRepo) GetSolution(ctx context.Context, ID uuid.UUID) (*domain.ChallengeSolution, error) {
	row, err := GetOrNotFound(func() (sqlc.Solution, error) { return r.Q(ctx).GetSolutionByChallengeID(ctx, ID) },
		apperr.ErrChallengeNotFound, "ChallengeRepo - GetSolution")
	if err != nil {
		return nil, err
	}

	files, err := r.Q(ctx).GetFilesByChallengeIDAndType(ctx, sqlc.GetFilesByChallengeIDAndTypeParams{
		ChallengeID: &ID,
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

	return &domain.ChallengeSolution{
		ChallengeID: row.ChallengeID,
		Content:     row.Content,
		State:       domain.SolutionStateOrDefault(row.State),
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
			State:       domain.SolutionStateOrDefault(row.State),
		}
	}

	return out, nil
}

// ListSolutions returns all solutions that the given team has access to, with
// their associated writeup files. It first fetches solution rows, collects the
// challenge IDs, then batches a second query for writeup files and merges them
// into a challengeID->files map before assembling the final entries.
func (r *ChallengeRepo) ListSolutions(ctx context.Context) ([]*domain.ChallengeSolutionEntry, error) {
	rows, err := r.Q(ctx).GetCandidateSolutions(ctx)
	if err != nil {
		return nil, fmt.Errorf("ChallengeRepo - ListSolutions: %w", err)
	}

	if len(rows) == 0 {
		return []*domain.ChallengeSolutionEntry{}, nil
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
		if f.ChallengeID == nil {
			continue
		}

		fileMap[*f.ChallengeID] = append(fileMap[*f.ChallengeID], &domain.File{
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

	out := make([]*domain.ChallengeSolutionEntry, len(rows))
	for i, row := range rows {
		ef := fileMap[row.ChallengeID]
		if ef == nil {
			ef = []*domain.File{}
		}

		out[i] = &domain.ChallengeSolutionEntry{
			ChallengeID:       row.ChallengeID,
			ChallengeTitle:    row.ChallengeTitle,
			ChallengeCategory: row.ChallengeCategory,
			Content:           row.Content,
			State:             domain.SolutionStateOrDefault(row.State),
			Files:             ef,
		}
	}

	return out, nil
}

// UpsertSolution inserts or updates the solution text for a challenge, then
// fetches the current writeup files to return the full ChallengeSolution.
func (r *ChallengeRepo) UpsertSolution(ctx context.Context, challengeID uuid.UUID, content, state string) (*domain.ChallengeSolution, error) {
	row, err := r.Q(ctx).UpsertSolution(ctx, sqlc.UpsertSolutionParams{
		ChallengeID: challengeID,
		Content:     content,
		State:       domain.SolutionStateOrDefault(state),
	})
	if err != nil {
		return nil, fmt.Errorf("ChallengeRepo - UpsertSolution: %w", err)
	}

	files, err := r.Q(ctx).GetFilesByChallengeIDAndType(ctx, sqlc.GetFilesByChallengeIDAndTypeParams{
		ChallengeID: &challengeID,
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

	return &domain.ChallengeSolution{
		ChallengeID: row.ChallengeID,
		Content:     row.Content,
		State:       domain.SolutionStateOrDefault(row.State),
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
