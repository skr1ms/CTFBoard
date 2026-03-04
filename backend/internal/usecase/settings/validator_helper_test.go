package settings

import (
	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/stretchr/testify/mock"
)

func (h *FieldTestHelper) SetupGetByEntityType(entityType entity.EntityType, fields []*entity.Field) {
	h.t.Helper()
	h.deps.fieldRepo.EXPECT().GetByEntityType(mock.Anything, entityType).Return(fields, nil)
}

func (h *FieldTestHelper) SetupGetByEntityTypeError(entityType entity.EntityType, err error) {
	h.t.Helper()
	h.deps.fieldRepo.EXPECT().GetByEntityType(mock.Anything, entityType).Return(nil, err)
}
