package competition

import (
	"context"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/stretchr/testify/mock"
)

func (h *CompetitionTestHelper) CreateCompetitionParamUseCase() *CompetitionParamUseCase {
	h.t.Helper()
	h.deps.tm.EXPECT().
		Run(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		}).
		Maybe()
	return NewCompetitionParamUseCase(CompetitionParamDeps{
		Repo:         h.deps.configRepo,
		AuditLogRepo: h.deps.auditLogRepo,
		TM:           h.deps.tm,
		Logger:       h.deps.logger,
	})
}

func (h *CompetitionTestHelper) NewCompetitionParam(key, value, description string, valueType entity.CompetitionParamValueType) *entity.CompetitionParam {
	h.t.Helper()
	return &entity.CompetitionParam{
		Key:         key,
		Value:       value,
		ValueType:   valueType,
		Description: description,
	}
}
