package competition

import (
	"github.com/skr1ms/CTFBoard/internal/entity"
)

func (h *CompetitionTestHelper) CreateDynamicConfigUseCase() *DynamicConfigUseCase {
	h.t.Helper()
	return NewDynamicConfigUseCase(h.deps.configRepo, h.deps.auditLogRepo)
}

func (h *CompetitionTestHelper) NewConfig(key, value, description string, valueType entity.ConfigValueType) *entity.Config {
	h.t.Helper()
	return &entity.Config{
		Key:         key,
		Value:       value,
		ValueType:   valueType,
		Description: description,
	}
}
