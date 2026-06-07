package competition

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func TestCompetitionParamUseCase_Set_Success(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	key, value, desc := "k", "v", "d"
	valueType := domain.CompetitionParamTypeString
	actorID := uuid.New()
	clientIP := "127.0.0.1"

	d.configRepo.EXPECT().Upsert(mock.Anything, mock.Anything).Return(nil).Run(func(_ context.Context, p *domain.CompetitionParam) {
		assert.Equal(t, key, p.Key)
		assert.Equal(t, value, p.Value)
		assert.Equal(t, valueType, p.ValueType)
	})
	d.auditLogRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)

	uc := d.createCompetitionParamUseCase()
	err := uc.Set(ctx, competitionParamSetParams(key, value, desc, valueType, "", actorID, clientIP))

	assert.NoError(t, err)
}

func TestCompetitionParamUseCase_Set_Error(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	key, value := "k", "v"
	actorID := uuid.New()

	d.configRepo.EXPECT().Upsert(mock.Anything, mock.Anything).Return(assert.AnError)

	uc := d.createCompetitionParamUseCase()
	err := uc.Set(ctx, competitionParamSetParams(key, value, "", domain.CompetitionParamTypeString, "", actorID, ""))

	assert.Error(t, err)
}

func TestCompetitionParamUseCase_Set_InvalidVisibility_ReturnsError(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()

	uc := d.createCompetitionParamUseCase()
	err := uc.Set(ctx, competitionParamSetParams("score_visibility", "garbage", "", domain.CompetitionParamTypeString, "", uuid.New(), ""))

	assert.Error(t, err)

	var ve *apperr.ValidationError
	assert.ErrorAs(t, err, &ve)
}

func TestCompetitionParamUseCase_Delete_Success(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	key := "k"
	p := newTestCompetitionParam(key, "v", "", domain.CompetitionParamTypeString)
	actorID := uuid.New()
	clientIP := "127.0.0.1"

	d.configRepo.EXPECT().GetByKeyForUpdate(mock.Anything, key).Return(p, nil)
	d.configRepo.EXPECT().Delete(mock.Anything, key).Return(nil)
	d.auditLogRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)

	uc := d.createCompetitionParamUseCase()
	err := uc.Delete(ctx, key, actorID, clientIP)

	assert.NoError(t, err)
}

func TestCompetitionParamUseCase_Delete_Error(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	key := "k"
	actorID := uuid.New()

	d.configRepo.EXPECT().GetByKeyForUpdate(mock.Anything, key).Return(nil, assert.AnError)

	uc := d.createCompetitionParamUseCase()
	err := uc.Delete(ctx, key, actorID, "")

	assert.Error(t, err)
}

func TestCompetitionParamUseCase_SetBatch_InvalidCategory_ReturnsError(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	params := []*domain.CompetitionParam{
		{Key: "k", Value: "v", ValueType: domain.CompetitionParamTypeString, Category: "invalid"},
	}
	actorID := uuid.New()

	uc := d.createCompetitionParamUseCase()
	err := uc.SetBatch(ctx, params, actorID, "")

	assert.Error(t, err)

	var ve2 *apperr.ValidationError
	assert.ErrorAs(t, err, &ve2)
}

func TestCompetitionParamUseCase_SetBatch_InvalidVisibility_ReturnsError(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	params := []*domain.CompetitionParam{
		{Key: "challenge_visibility", Value: "invalid-private", ValueType: domain.CompetitionParamTypeString, Category: "visibility"},
	}

	uc := d.createCompetitionParamUseCase()
	err := uc.SetBatch(ctx, params, uuid.New(), "")

	assert.Error(t, err)

	var ve *apperr.ValidationError
	assert.ErrorAs(t, err, &ve)
}

func TestCompetitionParamUseCase_SetBatch_Success(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	params := []*domain.CompetitionParam{
		{Key: "ctf_name", Value: "MyCTF", ValueType: domain.CompetitionParamTypeString, Category: "general"},
		{Key: "theme_color_primary", Value: "#ff0000", ValueType: domain.CompetitionParamTypeString, Category: "theme"},
	}
	actorID := uuid.New()

	d.configRepo.EXPECT().Upsert(mock.Anything, mock.Anything).Return(nil).Times(2)
	d.auditLogRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)

	uc := d.createCompetitionParamUseCase()
	err := uc.SetBatch(ctx, params, actorID, "")
	assert.NoError(t, err)
}

func TestCompetitionParamUseCase_Set_JSONValueType_InvalidReturnsError(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	key := "some_json_key"
	actorID := uuid.New()

	uc := d.createCompetitionParamUseCase()
	err := uc.Set(ctx, competitionParamSetParams(key, "not valid json", "", domain.CompetitionParamTypeJSON, "", actorID, ""))

	assert.Error(t, err)
	assert.True(t, errors.Is(err, apperr.ErrCompetitionParamInvalidValueType) ||
		strings.Contains(err.Error(), "validateValueType"))
}

func TestCompetitionParamUseCase_Set_JSONValueType_ValidSucceeds(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	key := "some_json_key"
	actorID := uuid.New()

	d.configRepo.EXPECT().Upsert(mock.Anything, mock.Anything).Return(nil)
	d.auditLogRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)

	uc := d.createCompetitionParamUseCase()
	err := uc.Set(ctx, competitionParamSetParams(key, `{"a":1}`, "", domain.CompetitionParamTypeJSON, "", actorID, ""))

	assert.NoError(t, err)
}
