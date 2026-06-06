package settings

import (
	"context"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

type SettingsRepository interface {
	Get(ctx context.Context) (*domain.Settings, error)
	GetForUpdate(ctx context.Context) (*domain.Settings, error)
	UpdateIfCurrent(ctx context.Context, s *domain.Settings) error
}

type AuditLogRepository interface {
	Create(ctx context.Context, log *domain.AuditLog) error
}

type TransactionManager interface {
	Run(ctx context.Context, fn func(context.Context) error) error
}

type CompetitionRepository interface {
	Get(ctx context.Context) (*domain.Competition, error)
}

type FieldRepository interface {
	Create(ctx context.Context, field *domain.Field) error
	GetByID(ctx context.Context, ID uuid.UUID) (*domain.Field, error)
	GetByEntityType(ctx context.Context, entityType domain.EntityType) ([]*domain.Field, error)
	GetAll(ctx context.Context) ([]*domain.Field, error)
	Update(ctx context.Context, field *domain.Field) error
	Delete(ctx context.Context, ID uuid.UUID) error
}
