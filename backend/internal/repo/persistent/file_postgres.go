package persistent

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

type FileRepo struct {
	pool *pgxpool.Pool
}

var _ repo.FileRepository = (*FileRepo)(nil)

func NewFileRepo(pool *pgxpool.Pool) *FileRepo {
	return &FileRepo{pool: pool}
}

func (r *FileRepo) q(ctx context.Context) *sqlc.Queries {
	return sqlc.New(ExtractDB(ctx, r.pool))
}

func toEntityFile(f sqlc.File) *entity.File {
	return &entity.File{
		ID:          f.ID,
		Type:        entity.FileType(f.Type),
		ChallengeID: f.ChallengeID,
		Location:    f.Location,
		Filename:    f.Filename,
		Size:        f.Size,
		SHA256:      f.SHA256,
		CreatedAt:   ptrTimeToTime(timestamptzToTime(f.CreatedAt)),
	}
}

func (r *FileRepo) Create(ctx context.Context, file *entity.File) error {
	if file.ID == uuid.Nil {
		file.ID = uuid.New()
	}
	if file.CreatedAt.IsZero() {
		file.CreatedAt = time.Now()
	}
	err := r.q(ctx).CreateFile(ctx, sqlc.CreateFileParams{
		ID:          file.ID,
		Type:        string(file.Type),
		ChallengeID: file.ChallengeID,
		Location:    file.Location,
		Filename:    file.Filename,
		Size:        file.Size,
		SHA256:      file.SHA256,
		CreatedAt:   timeToTimestamptz(&file.CreatedAt),
	})
	if err != nil {
		return fmt.Errorf("FileRepo - Create: %w", err)
	}
	return nil
}

func (r *FileRepo) GetByID(ctx context.Context, ID uuid.UUID) (*entity.File, error) {
	f, err := r.q(ctx).GetFileByID(ctx, ID)
	if err != nil {
		if isNoRows(err) {
			return nil, httperr.ErrFileNotFound
		}
		return nil, fmt.Errorf("FileRepo - GetByID: %w", err)
	}
	return toEntityFile(f), nil
}

func (r *FileRepo) GetByLocation(ctx context.Context, location string) (*entity.File, error) {
	f, err := r.q(ctx).GetFileByLocation(ctx, location)
	if err != nil {
		if isNoRows(err) {
			return nil, httperr.ErrFileNotFound
		}
		return nil, fmt.Errorf("FileRepo - GetByLocation: %w", err)
	}
	return toEntityFile(f), nil
}

func (r *FileRepo) GetByChallengeID(ctx context.Context, challengeID uuid.UUID, fileType entity.FileType) ([]*entity.File, error) {
	rows, err := r.q(ctx).GetFilesByChallengeIDAndType(ctx, sqlc.GetFilesByChallengeIDAndTypeParams{
		ChallengeID: challengeID,
		Type:        string(fileType),
	})
	if err != nil {
		return nil, fmt.Errorf("FileRepo - GetByChallengeID: %w", err)
	}
	out := make([]*entity.File, 0, len(rows))
	for _, f := range rows {
		out = append(out, toEntityFile(f))
	}
	return out, nil
}

func (r *FileRepo) GetAllByChallengeID(ctx context.Context, challengeID uuid.UUID) ([]*entity.File, error) {
	rows, err := r.q(ctx).GetFilesByChallengeID(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("FileRepo - GetAllByChallengeID: %w", err)
	}
	out := make([]*entity.File, 0, len(rows))
	for _, f := range rows {
		out = append(out, toEntityFile(f))
	}
	return out, nil
}

func (r *FileRepo) GetByChallengeIDs(ctx context.Context, challengeIDs []uuid.UUID) (map[uuid.UUID][]*entity.File, error) {
	if len(challengeIDs) == 0 {
		return map[uuid.UUID][]*entity.File{}, nil
	}
	rows, err := r.q(ctx).GetFilesByChallengeIDs(ctx, challengeIDs)
	if err != nil {
		return nil, fmt.Errorf("FileRepo - GetByChallengeIDs: %w", err)
	}
	out := make(map[uuid.UUID][]*entity.File)
	for _, f := range rows {
		ef := toEntityFile(f)
		out[f.ChallengeID] = append(out[f.ChallengeID], ef)
	}
	return out, nil
}

func (r *FileRepo) GetAll(ctx context.Context) ([]*entity.File, error) {
	rows, err := r.q(ctx).GetAllFiles(ctx)
	if err != nil {
		return nil, fmt.Errorf("FileRepo - GetAll: %w", err)
	}
	out := make([]*entity.File, 0, len(rows))
	for _, f := range rows {
		out = append(out, toEntityFile(f))
	}
	return out, nil
}

func (r *FileRepo) ListLocations(ctx context.Context, limit, offset int) ([]string, error) {
	limit32, err := intToInt32Safe(limit)
	if err != nil {
		return nil, fmt.Errorf("FileRepo - ListLocations - limit: %w", err)
	}
	offset32, err := intToInt32Safe(offset)
	if err != nil {
		return nil, fmt.Errorf("FileRepo - ListLocations - offset: %w", err)
	}
	rows, err := r.q(ctx).ListFileLocations(ctx, sqlc.ListFileLocationsParams{
		Limit:  limit32,
		Offset: offset32,
	})
	if err != nil {
		return nil, fmt.Errorf("FileRepo - ListLocations: %w", err)
	}
	return rows, nil
}

func (r *FileRepo) Delete(ctx context.Context, ID uuid.UUID) error {
	_, err := r.q(ctx).DeleteFile(ctx, ID)
	if err != nil && !isNoRows(err) {
		return fmt.Errorf("FileRepo - Delete: %w", err)
	}
	return nil
}
