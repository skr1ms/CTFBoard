package persistent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/repo/persistent/sqlc"
)

type FieldRepo struct {
	pool *pgxpool.Pool
	q    *sqlc.Queries
}

func NewFieldRepo(pool *pgxpool.Pool) *FieldRepo {
	return &FieldRepo{
		pool: pool,
		q:    sqlc.New(pool),
	}
}

func optionsToBytes(opts []string) ([]byte, error) {
	if len(opts) == 0 {
		return nil, nil
	}
	return json.Marshal(opts)
}

func optionsFromBytes(b []byte) ([]string, error) {
	if len(b) == 0 {
		return nil, nil
	}
	var out []string
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *FieldRepo) Create(ctx context.Context, field *entity.Field) error {
	if field.ID == uuid.Nil {
		field.ID = uuid.New()
	}
	required := field.Required
	orderIndex := intToInt32Ptr(field.OrderIndex)
	createdAt := field.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	opts, err := optionsToBytes(field.Options)
	if err != nil {
		return fmt.Errorf("FieldRepo - Create options: %w", err)
	}
	return r.q.CreateField(ctx, sqlc.CreateFieldParams{
		ID:         field.ID,
		Name:       field.Name,
		FieldType:  string(field.FieldType),
		EntityType: string(field.EntityType),
		Required:   &required,
		Options:    opts,
		OrderIndex: orderIndex,
		CreatedAt:  &createdAt,
	})
}

func (r *FieldRepo) GetByID(ctx context.Context, id uuid.UUID) (*entity.Field, error) {
	row, err := r.q.GetFieldByID(ctx, id)
	if err != nil {
		if isNoRows(err) {
			return nil, entityError.ErrFieldNotFound
		}
		return nil, fmt.Errorf("FieldRepo - GetByID: %w", err)
	}
	opts, err := optionsFromBytes(row.Options)
	if err != nil {
		return nil, fmt.Errorf("FieldRepo - GetByID options: %w", err)
	}
	return &entity.Field{
		ID:         row.ID,
		Name:       row.Name,
		FieldType:  entity.FieldType(row.FieldType),
		EntityType: entity.EntityType(row.EntityType),
		Required:   boolPtrToBool(row.Required),
		Options:    opts,
		OrderIndex: int32PtrToInt(row.OrderIndex),
		CreatedAt:  ptrTimeToTime(row.CreatedAt),
	}, nil
}

func (r *FieldRepo) GetByEntityType(ctx context.Context, entityType entity.EntityType) ([]*entity.Field, error) {
	rows, err := r.q.GetFieldsByEntityType(ctx, string(entityType))
	if err != nil {
		return nil, fmt.Errorf("FieldRepo - GetByEntityType: %w", err)
	}
	out := make([]*entity.Field, len(rows))
	for i, row := range rows {
		opts, err := optionsFromBytes(row.Options)
		if err != nil {
			return nil, fmt.Errorf("FieldRepo - GetByEntityType options: %w", err)
		}
		out[i] = &entity.Field{
			ID:         row.ID,
			Name:       row.Name,
			FieldType:  entity.FieldType(row.FieldType),
			EntityType: entity.EntityType(row.EntityType),
			Required:   boolPtrToBool(row.Required),
			Options:    opts,
			OrderIndex: int32PtrToInt(row.OrderIndex),
			CreatedAt:  ptrTimeToTime(row.CreatedAt),
		}
	}
	return out, nil
}

func (r *FieldRepo) GetAll(ctx context.Context) ([]*entity.Field, error) {
	rows, err := r.q.GetAllFields(ctx)
	if err != nil {
		return nil, fmt.Errorf("FieldRepo - GetAll: %w", err)
	}
	out := make([]*entity.Field, len(rows))
	for i, row := range rows {
		opts, err := optionsFromBytes(row.Options)
		if err != nil {
			return nil, fmt.Errorf("FieldRepo - GetAll options: %w", err)
		}
		out[i] = &entity.Field{
			ID:         row.ID,
			Name:       row.Name,
			FieldType:  entity.FieldType(row.FieldType),
			EntityType: entity.EntityType(row.EntityType),
			Required:   boolPtrToBool(row.Required),
			Options:    opts,
			OrderIndex: int32PtrToInt(row.OrderIndex),
			CreatedAt:  ptrTimeToTime(row.CreatedAt),
		}
	}
	return out, nil
}

func (r *FieldRepo) Update(ctx context.Context, field *entity.Field) error {
	required := field.Required
	orderIndex := intToInt32Ptr(field.OrderIndex)
	opts, err := optionsToBytes(field.Options)
	if err != nil {
		return fmt.Errorf("FieldRepo - Update options: %w", err)
	}
	return r.q.UpdateField(ctx, sqlc.UpdateFieldParams{
		ID:         field.ID,
		Name:       field.Name,
		FieldType:  string(field.FieldType),
		Required:   &required,
		Options:    opts,
		OrderIndex: orderIndex,
	})
}

func (r *FieldRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return r.q.DeleteField(ctx, id)
}

type FieldValueRepo struct {
	pool *pgxpool.Pool
	q    *sqlc.Queries
}

func NewFieldValueRepo(pool *pgxpool.Pool) *FieldValueRepo {
	return &FieldValueRepo{
		pool: pool,
		q:    sqlc.New(pool),
	}
}

func (r *FieldValueRepo) GetByEntityID(ctx context.Context, entityID uuid.UUID) ([]*entity.FieldValue, error) {
	rows, err := r.q.GetFieldValuesByEntityID(ctx, entityID)
	if err != nil {
		return nil, fmt.Errorf("FieldValueRepo - GetByEntityID: %w", err)
	}
	out := make([]*entity.FieldValue, len(rows))
	for i, row := range rows {
		out[i] = &entity.FieldValue{
			ID:        row.ID,
			FieldID:   row.FieldID,
			EntityID:  row.EntityID,
			Value:     row.Value,
			CreatedAt: ptrTimeToTime(row.CreatedAt),
		}
	}
	return out, nil
}

func (r *FieldValueRepo) SetValues(ctx context.Context, entityID uuid.UUID, values map[string]string) error {
	if err := r.q.DeleteFieldValuesByEntityID(ctx, entityID); err != nil {
		return fmt.Errorf("FieldValueRepo - SetValues - Delete: %w", err)
	}
	for fieldIDStr, value := range values {
		fieldID, err := uuid.Parse(fieldIDStr)
		if err != nil {
			return fmt.Errorf("FieldValueRepo - SetValues: invalid field_id %s: %w", fieldIDStr, err)
		}
		now := time.Now()
		if err := r.q.UpsertFieldValue(ctx, sqlc.UpsertFieldValueParams{
			ID:        uuid.New(),
			FieldID:   fieldID,
			EntityID:  entityID,
			Value:     value,
			CreatedAt: &now,
		}); err != nil {
			return fmt.Errorf("FieldValueRepo - SetValues - Upsert: %w", err)
		}
	}
	return nil
}
