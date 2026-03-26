package persistent

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wahrwelt-kit/go-pgkit/pgutil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

type FileRepo struct {
	BaseRepo
}

var _ repo.FileRepository = (*FileRepo)(nil)

func NewFileRepo(pool *pgxpool.Pool) *FileRepo {
	return &FileRepo{BaseRepo: BaseRepo{pool: pool}}
}

func toDomainFile(f sqlc.File) *domain.File {
	return &domain.File{
		ID:          f.ID,
		Type:        domain.FileType(f.Type),
		ChallengeID: f.ChallengeID,
		Location:    f.Location,
		Filename:    f.Filename,
		Size:        f.Size,
		SHA256:      f.SHA256,
		CreatedAt:   pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(f.CreatedAt)),
	}
}

func (r *FileRepo) Create(ctx context.Context, file *domain.File) error {
	EnsureID(&file.ID)

	if file.CreatedAt.IsZero() {
		file.CreatedAt = time.Now()
	}

	err := r.Q(ctx).CreateFile(ctx, sqlc.CreateFileParams{
		ID:          file.ID,
		Type:        string(file.Type),
		ChallengeID: file.ChallengeID,
		Location:    file.Location,
		Filename:    file.Filename,
		Size:        file.Size,
		SHA256:      file.SHA256,
		CreatedAt:   pgutil.TimeToTimestamptz(&file.CreatedAt),
	})
	if err != nil {
		return fmt.Errorf("FileRepo - Create: %w", err)
	}

	return nil
}

func (r *FileRepo) GetByID(ctx context.Context, ID uuid.UUID) (*domain.File, error) {
	f, err := r.Q(ctx).GetFileByID(ctx, ID)
	if err != nil {
		if pgutil.IsNoRows(err) {
			return nil, httperr.ErrFileNotFound
		}

		return nil, fmt.Errorf("FileRepo - GetByID: %w", err)
	}

	return toDomainFile(f), nil
}

func (r *FileRepo) GetByLocation(ctx context.Context, location string) (*domain.File, error) {
	f, err := r.Q(ctx).GetFileByLocation(ctx, location)
	if err != nil {
		if pgutil.IsNoRows(err) {
			return nil, httperr.ErrFileNotFound
		}

		return nil, fmt.Errorf("FileRepo - GetByLocation: %w", err)
	}

	return toDomainFile(f), nil
}

func (r *FileRepo) GetByChallengeID(ctx context.Context, challengeID uuid.UUID, fileType domain.FileType) ([]*domain.File, error) {
	rows, err := r.Q(ctx).GetFilesByChallengeIDAndType(ctx, sqlc.GetFilesByChallengeIDAndTypeParams{
		ChallengeID: challengeID,
		Type:        string(fileType),
	})
	if err != nil {
		return nil, fmt.Errorf("FileRepo - GetByChallengeID: %w", err)
	}

	out := make([]*domain.File, 0, len(rows))
	for _, f := range rows {
		out = append(out, toDomainFile(f))
	}

	return out, nil
}

func (r *FileRepo) GetAllByChallengeID(ctx context.Context, challengeID uuid.UUID) ([]*domain.File, error) {
	rows, err := r.Q(ctx).GetFilesByChallengeID(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("FileRepo - GetAllByChallengeID: %w", err)
	}

	out := make([]*domain.File, 0, len(rows))
	for _, f := range rows {
		out = append(out, toDomainFile(f))
	}

	return out, nil
}

func (r *FileRepo) GetByChallengeIDs(ctx context.Context, challengeIDs []uuid.UUID) (map[uuid.UUID][]*domain.File, error) {
	if len(challengeIDs) == 0 {
		return map[uuid.UUID][]*domain.File{}, nil
	}

	rows, err := r.Q(ctx).GetFilesByChallengeIDs(ctx, challengeIDs)
	if err != nil {
		return nil, fmt.Errorf("FileRepo - GetByChallengeIDs: %w", err)
	}

	out := make(map[uuid.UUID][]*domain.File)

	for _, f := range rows {
		ef := toDomainFile(f)
		out[f.ChallengeID] = append(out[f.ChallengeID], ef)
	}

	return out, nil
}

func (r *FileRepo) GetAll(ctx context.Context) ([]*domain.File, error) {
	rows, err := r.Q(ctx).GetAllFiles(ctx)
	if err != nil {
		return nil, fmt.Errorf("FileRepo - GetAll: %w", err)
	}

	out := make([]*domain.File, 0, len(rows))
	for _, f := range rows {
		out = append(out, toDomainFile(f))
	}

	return out, nil
}

func (r *FileRepo) ListLocations(ctx context.Context, limit, offset int) ([]string, error) {
	limit32, offset32, err := toLimitOffset(limit, offset)
	if err != nil {
		return nil, fmt.Errorf("FileRepo - ListLocations: %w", err)
	}

	rows, err := r.Q(ctx).ListFileLocations(ctx, sqlc.ListFileLocationsParams{
		Limit:  limit32,
		Offset: offset32,
	})
	if err != nil {
		return nil, fmt.Errorf("FileRepo - ListLocations: %w", err)
	}

	return rows, nil
}

func (r *FileRepo) Delete(ctx context.Context, ID uuid.UUID) error {
	_, err := r.Q(ctx).DeleteFile(ctx, ID)
	if err != nil {
		if pgutil.IsNoRows(err) {
			return httperr.ErrFileNotFound
		}

		return fmt.Errorf("FileRepo - Delete: %w", err)
	}

	return nil
}
