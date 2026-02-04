package settings

import (
	"testing"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/usecase/settings/mocks"
)

type FieldTestHelper struct {
	t    *testing.T
	deps *fieldTestDeps
}

type fieldTestDeps struct {
	fieldRepo *mocks.MockFieldRepository
}

func NewFieldTestHelper(t *testing.T) *FieldTestHelper {
	t.Helper()
	return &FieldTestHelper{
		t: t,
		deps: &fieldTestDeps{
			fieldRepo: mocks.NewMockFieldRepository(t),
		},
	}
}

func (h *FieldTestHelper) Deps() *fieldTestDeps {
	h.t.Helper()
	return h.deps
}

func (h *FieldTestHelper) CreateUseCase() *FieldUseCase {
	h.t.Helper()
	return NewFieldUseCase(h.deps.fieldRepo)
}

func (h *FieldTestHelper) CreateFieldValidator() *FieldValidator {
	h.t.Helper()
	return NewFieldValidator(h.deps.fieldRepo)
}

func (h *FieldTestHelper) NewField(name string, fieldType entity.FieldType, entityType entity.EntityType, required bool, options []string, orderIndex int) *entity.Field {
	h.t.Helper()
	return &entity.Field{
		ID:         uuid.New(),
		Name:       name,
		FieldType:  fieldType,
		EntityType: entityType,
		Required:   required,
		Options:    options,
		OrderIndex: orderIndex,
	}
}
