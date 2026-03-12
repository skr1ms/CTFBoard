package persistent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

type FieldRepo struct {
	pool *pgxpool.Pool
}

var _ repo.FieldRepository = (*FieldRepo)(nil)

func NewFieldRepo(pool *pgxpool.Pool) *FieldRepo {
	return &FieldRepo{pool: pool}
}

func (r *FieldRepo) q(ctx context.Context) *sqlc.Queries {
	return sqlc.New(ExtractDB(ctx, r.pool))
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
		return nil, fmt.Errorf("FieldRepo - optionsFromBytes: %w", err)
	}
	return out, nil
}

func (r *FieldRepo) Create(ctx context.Context, field *entity.Field) error {
	if field.ID == uuid.Nil {
		field.ID = uuid.New()
	}
	required := field.Required
	orderIndex, err := intToInt32Ptr(field.OrderIndex)
	if err != nil {
		return fmt.Errorf("FieldRepo - Create - orderIndex: %w", err)
	}
	createdAt := field.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	opts, err := optionsToBytes(field.Options)
	if err != nil {
		return fmt.Errorf("FieldRepo - Create - options: %w", err)
	}
	if err := r.q(ctx).CreateField(ctx, sqlc.CreateFieldParams{
		ID:         field.ID,
		Name:       field.Name,
		FieldType:  string(field.FieldType),
		EntityType: string(field.EntityType),
		Required:   &required,
		Options:    opts,
		OrderIndex: orderIndex,
		CreatedAt:  timeToTimestamptz(&createdAt),
	}); err != nil {
		return fmt.Errorf("FieldRepo - Create: %w", err)
	}
	return nil
}

func (r *FieldRepo) GetByID(ctx context.Context, ID uuid.UUID) (*entity.Field, error) {
	row, err := r.q(ctx).GetFieldByID(ctx, ID)
	if err != nil {
		if isNoRows(err) {
			return nil, httperr.ErrFieldNotFound
		}
		return nil, fmt.Errorf("FieldRepo - GetByID: %w", err)
	}
	opts, err := optionsFromBytes(row.Options)
	if err != nil {
		return nil, fmt.Errorf("FieldRepo - GetByID - options: %w", err)
	}
	return &entity.Field{
		ID:         row.ID,
		Name:       row.Name,
		FieldType:  entity.FieldType(row.FieldType),
		EntityType: entity.EntityType(row.EntityType),
		Required:   boolPtrToBool(row.Required),
		Options:    opts,
		OrderIndex: int32PtrToInt(row.OrderIndex),
		CreatedAt:  ptrTimeToTime(timestamptzToTime(row.CreatedAt)),
	}, nil
}

func (r *FieldRepo) GetByEntityType(ctx context.Context, entityType entity.EntityType) ([]*entity.Field, error) {
	rows, err := r.q(ctx).GetFieldsByEntityType(ctx, string(entityType))
	if err != nil {
		return nil, fmt.Errorf("FieldRepo - GetByEntityType: %w", err)
	}
	out := make([]*entity.Field, len(rows))
	for i, row := range rows {
		opts, err := optionsFromBytes(row.Options)
		if err != nil {
			return nil, fmt.Errorf("FieldRepo - GetByEntityType - options: %w", err)
		}
		out[i] = &entity.Field{
			ID:         row.ID,
			Name:       row.Name,
			FieldType:  entity.FieldType(row.FieldType),
			EntityType: entity.EntityType(row.EntityType),
			Required:   boolPtrToBool(row.Required),
			Options:    opts,
			OrderIndex: int32PtrToInt(row.OrderIndex),
			CreatedAt:  ptrTimeToTime(timestamptzToTime(row.CreatedAt)),
		}
	}
	return out, nil
}

func (r *FieldRepo) GetAll(ctx context.Context) ([]*entity.Field, error) {
	rows, err := r.q(ctx).GetAllFields(ctx)
	if err != nil {
		return nil, fmt.Errorf("FieldRepo - GetAll: %w", err)
	}
	out := make([]*entity.Field, len(rows))
	for i, row := range rows {
		opts, err := optionsFromBytes(row.Options)
		if err != nil {
			return nil, fmt.Errorf("FieldRepo - GetAll - options: %w", err)
		}
		out[i] = &entity.Field{
			ID:         row.ID,
			Name:       row.Name,
			FieldType:  entity.FieldType(row.FieldType),
			EntityType: entity.EntityType(row.EntityType),
			Required:   boolPtrToBool(row.Required),
			Options:    opts,
			OrderIndex: int32PtrToInt(row.OrderIndex),
			CreatedAt:  ptrTimeToTime(timestamptzToTime(row.CreatedAt)),
		}
	}
	return out, nil
}

func (r *FieldRepo) Update(ctx context.Context, field *entity.Field) error {
	required := field.Required
	orderIndex, err := intToInt32Ptr(field.OrderIndex)
	if err != nil {
		return fmt.Errorf("FieldRepo - Update - orderIndex: %w", err)
	}
	opts, err := optionsToBytes(field.Options)
	if err != nil {
		return fmt.Errorf("FieldRepo - Update - options: %w", err)
	}
	if err := r.q(ctx).UpdateField(ctx, sqlc.UpdateFieldParams{
		ID:         field.ID,
		Name:       field.Name,
		FieldType:  string(field.FieldType),
		Required:   &required,
		Options:    opts,
		OrderIndex: orderIndex,
	}); err != nil {
		return fmt.Errorf("FieldRepo - Update: %w", err)
	}
	return nil
}

// Delete removes a field by ID. Idempotent: returns nil if the field does not exist.
func (r *FieldRepo) Delete(ctx context.Context, ID uuid.UUID) error {
	if err := r.q(ctx).DeleteField(ctx, ID); err != nil {
		return fmt.Errorf("FieldRepo - Delete: %w", err)
	}
	return nil
}

type FieldValueRepo struct {
	pool *pgxpool.Pool
}

func NewFieldValueRepo(pool *pgxpool.Pool) *FieldValueRepo {
	return &FieldValueRepo{pool: pool}
}

var _ repo.FieldValueRepository = (*FieldValueRepo)(nil)

func (r *FieldValueRepo) q(ctx context.Context) *sqlc.Queries {
	return sqlc.New(ExtractDB(ctx, r.pool))
}

func (r *FieldValueRepo) GetByEntityID(ctx context.Context, entityID uuid.UUID) ([]*entity.FieldValue, error) {
	rows, err := r.q(ctx).GetFieldValuesByEntityID(ctx, entityID)
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
			CreatedAt: ptrTimeToTime(timestamptzToTime(row.CreatedAt)),
		}
	}
	return out, nil
}

func (r *FieldValueRepo) GetAll(ctx context.Context) ([]*entity.FieldValue, error) {
	rows, err := r.q(ctx).GetAllFieldValues(ctx)
	if err != nil {
		return nil, fmt.Errorf("FieldValueRepo - GetAll: %w", err)
	}
	out := make([]*entity.FieldValue, len(rows))
	for i, row := range rows {
		out[i] = &entity.FieldValue{
			ID:        row.ID,
			FieldID:   row.FieldID,
			EntityID:  row.EntityID,
			Value:     row.Value,
			CreatedAt: ptrTimeToTime(timestamptzToTime(row.CreatedAt)),
		}
	}
	return out, nil
}

const maxFieldValueLength = 65536

// SetValues replaces all field values for the given entity with the provided map.
// It performs Delete then Insert and is not atomic on its own; callers must run
// this inside a transaction (e.g. TransactionManager.Run) to avoid lost updates
// under concurrent calls for the same entityID.
func (r *FieldValueRepo) SetValues(ctx context.Context, entityID uuid.UUID, values map[string]string) error {
	return r.setValuesInner(ctx, entityID, values)
}

func (r *FieldValueRepo) DeleteByEntityID(ctx context.Context, entityID uuid.UUID) error {
	if err := r.q(ctx).DeleteFieldValuesByEntityID(ctx, entityID); err != nil {
		return fmt.Errorf("FieldValueRepo - DeleteByEntityID: %w", err)
	}
	return nil
}

func (r *FieldValueRepo) setValuesInner(ctx context.Context, entityID uuid.UUID, values map[string]string) error {
	if err := r.q(ctx).DeleteFieldValuesByEntityID(ctx, entityID); err != nil {
		return fmt.Errorf("FieldValueRepo - SetValues - Delete: %w", err)
	}
	if len(values) == 0 {
		return nil
	}
	for _, value := range values {
		if len(value) > maxFieldValueLength {
			return httperr.NewValidationErrorf("field value exceeds maximum length (%d)", maxFieldValueLength)
		}
	}
	now := time.Now()
	qb := squirrel.Insert("field_values").
		Columns("id", "field_id", "entity_id", "value", "created_at").
		Suffix("ON CONFLICT (field_id, entity_id) DO UPDATE SET value = EXCLUDED.value").
		PlaceholderFormat(squirrel.Dollar)
	for fieldIDStr, value := range values {
		fieldID, err := uuid.Parse(fieldIDStr)
		if err != nil {
			return fmt.Errorf("FieldValueRepo - SetValues: invalid field_id %s: %w", fieldIDStr, err)
		}
		qb = qb.Values(uuid.New(), fieldID, entityID, value, now)
	}
	query, args, err := qb.ToSql()
	if err != nil {
		return fmt.Errorf("FieldValueRepo - SetValues - ToSql: %w", err)
	}
	if _, err := ExtractDB(ctx, r.pool).Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("FieldValueRepo - SetValues - Exec: %w", err)
	}
	return nil
}
