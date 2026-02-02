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

type FileRepository struct {
	db *pgxpool.Pool
	q  *sqlc.Queries
}

func NewFileRepository(db *pgxpool.Pool) *FileRepository {
	return &FileRepository{db: db, q: sqlc.New(db)}
}

func toEntityFile(f sqlc.File) *entity.File {
	return &entity.File{
		ID:          f.ID,
		Type:        entity.FileType(f.Type),
		ChallengeID: f.ChallengeID,
		Location:    f.Location,
		Filename:    f.Filename,
		Size:        f.Size,
		SHA256:      f.Sha256,
		CreatedAt:   f.CreatedAt,
	}
}

func (r *FileRepository) Create(ctx context.Context, file *entity.File) error {
	if file.ID == uuid.Nil {
		file.ID = uuid.New()
	}
	if file.CreatedAt.IsZero() {
		file.CreatedAt = time.Now()
	}
	err := r.q.CreateFile(ctx, sqlc.CreateFileParams{
		ID:          file.ID,
		Type:        string(file.Type),
		ChallengeID: file.ChallengeID,
		Location:    file.Location,
		Filename:    file.Filename,
		Size:        file.Size,
		Sha256:      file.SHA256,
		CreatedAt:   file.CreatedAt,
	})
	if err != nil {
		return fmt.Errorf("FileRepository - Create: %w", err)
	}
	return nil
}

func (r *FileRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.File, error) {
	f, err := r.q.GetFileByID(ctx, id)
	if err != nil {
		if isNoRows(err) {
			return nil, entityError.ErrFileNotFound
		}
		return nil, fmt.Errorf("FileRepository - GetByID: %w", err)
	}
	return toEntityFile(f), nil
}

func (r *FileRepository) GetByChallengeID(ctx context.Context, challengeID uuid.UUID, fileType entity.FileType) ([]*entity.File, error) {
	rows, err := r.q.GetFilesByChallengeIDAndType(ctx, sqlc.GetFilesByChallengeIDAndTypeParams{
		ChallengeID: challengeID,
		Type:        string(fileType),
	})
	if err != nil {
		return nil, fmt.Errorf("FileRepository - GetByChallengeID: %w", err)
	}
	out := make([]*entity.File, 0, len(rows))
	for _, f := range rows {
		out = append(out, toEntityFile(f))
	}
	return out, nil
}

func (r *FileRepository) GetAll(ctx context.Context) ([]*entity.File, error) {
	rows, err := r.q.GetAllFiles(ctx)
	if err != nil {
		return nil, fmt.Errorf("FileRepository - GetAll: %w", err)
	}
	out := make([]*entity.File, 0, len(rows))
	for _, f := range rows {
		out = append(out, toEntityFile(f))
	}
	return out, nil
}

func (r *FileRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.q.DeleteFile(ctx, id)
	if err != nil {
		if isNoRows(err) {
			return entityError.ErrFileNotFound
		}
		return fmt.Errorf("FileRepository - Delete: %w", err)
	}
	return nil
}
