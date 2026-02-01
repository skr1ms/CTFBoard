package persistent

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
)

var fileColumns = []string{"id", "type", "challenge_id", "location", "filename", "size", "sha256", "created_at"}

func scanFile(row rowScanner) (*entity.File, error) {
	var f entity.File
	err := row.Scan(&f.ID, &f.Type, &f.ChallengeID, &f.Location, &f.Filename, &f.Size, &f.SHA256, &f.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

type FileRepository struct {
	pool *pgxpool.Pool
}

func NewFileRepository(pool *pgxpool.Pool) *FileRepository {
	return &FileRepository{pool: pool}
}

func (r *FileRepository) Create(ctx context.Context, file *entity.File) error {
	if file.ID == uuid.Nil {
		file.ID = uuid.New()
	}
	if file.CreatedAt.IsZero() {
		file.CreatedAt = time.Now()
	}

	query := squirrel.Insert("files").
		Columns(fileColumns...).
		Values(
			file.ID,
			file.Type,
			file.ChallengeID,
			file.Location,
			file.Filename,
			file.Size,
			file.SHA256,
			file.CreatedAt,
		).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("FileRepository - Create - BuildQuery: %w", err)
	}

	_, err = r.pool.Exec(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("FileRepository - Create: %w", err)
	}
	return nil
}

func (r *FileRepository) GetByID(ctx context.Context, ID uuid.UUID) (*entity.File, error) {
	query := squirrel.Select(fileColumns...).
		From("files").
		Where(squirrel.Eq{"id": ID}).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("FileRepository - GetByID - BuildQuery: %w", err)
	}

	row := r.pool.QueryRow(ctx, sqlQuery, args...)
	file, err := scanFile(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entityError.ErrFileNotFound
		}
		return nil, fmt.Errorf("FileRepository - GetByID: %w", err)
	}
	return file, nil
}

func (r *FileRepository) GetByChallengeID(ctx context.Context, challengeID uuid.UUID, fileType entity.FileType) ([]*entity.File, error) {
	query := squirrel.Select(fileColumns...).
		From("files").
		Where(squirrel.Eq{"challenge_id": challengeID, "type": fileType}).
		OrderBy("created_at DESC").
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("FileRepository - GetByChallengeID - BuildQuery: %w", err)
	}

	rows, err := r.pool.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("FileRepository - GetByChallengeID: %w", err)
	}
	defer rows.Close()

	var files []*entity.File
	for rows.Next() {
		file, err := scanFile(rows)
		if err != nil {
			return nil, fmt.Errorf("FileRepository - GetByChallengeID - Scan: %w", err)
		}
		files = append(files, file)
	}
	return files, nil
}

func (r *FileRepository) GetAll(ctx context.Context) ([]*entity.File, error) {
	query := squirrel.Select(fileColumns...).
		From("files").
		OrderBy("created_at DESC").
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("FileRepository - GetAll - BuildQuery: %w", err)
	}

	rows, err := r.pool.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("FileRepository - GetAll: %w", err)
	}
	defer rows.Close()

	var files []*entity.File
	for rows.Next() {
		file, err := scanFile(rows)
		if err != nil {
			return nil, fmt.Errorf("FileRepository - GetAll - Scan: %w", err)
		}
		files = append(files, file)
	}
	return files, nil
}

func (r *FileRepository) Delete(ctx context.Context, ID uuid.UUID) error {
	query := squirrel.Delete("files").
		Where(squirrel.Eq{"id": ID}).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("FileRepository - Delete - BuildQuery: %w", err)
	}

	result, err := r.pool.Exec(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("FileRepository - Delete: %w", err)
	}
	if result.RowsAffected() == 0 {
		return entityError.ErrFileNotFound
	}
	return nil
}
