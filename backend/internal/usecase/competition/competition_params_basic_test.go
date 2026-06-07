package competition

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func TestCompetitionParamUseCase_Get_Success(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	key := "k"
	p := newTestCompetitionParam(key, "v", "desc", domain.CompetitionParamTypeString)

	d.configRepo.EXPECT().GetAll(mock.Anything).Return([]*domain.CompetitionParam{p}, nil)

	uc := d.createCompetitionParamUseCase()
	got, err := uc.Get(ctx, key)

	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, key, got.Key)
	assert.Equal(t, "v", got.Value)
}

func TestCompetitionParamUseCase_Get_Error(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	key := "k"

	d.configRepo.EXPECT().GetAll(mock.Anything).Return(nil, assert.AnError)

	uc := d.createCompetitionParamUseCase()
	got, err := uc.Get(ctx, key)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestCompetitionParamUseCase_GetAll_Success(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	list := []*domain.CompetitionParam{newTestCompetitionParam("k1", "v1", "", domain.CompetitionParamTypeString)}

	d.configRepo.EXPECT().GetAll(mock.Anything).Return(list, nil)

	uc := d.createCompetitionParamUseCase()
	_, _ = uc.Get(ctx, "k1")
	got, err := uc.GetAll(ctx)

	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(got), domain.ConfigRegistryCount())

	var k1 *domain.CompetitionParam

	for _, p := range got {
		if p.Key == "k1" {
			k1 = p

			break
		}
	}

	require.NotNil(t, k1)
	assert.Equal(t, "v1", k1.Value)
}

func TestCompetitionParamUseCase_GetAll_Error(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()

	d.configRepo.EXPECT().GetAll(mock.Anything).Return(nil, assert.AnError)

	uc := d.createCompetitionParamUseCase()
	got, err := uc.GetAll(ctx)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestCompetitionParamUseCase_GetString_Success(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	key := "k"
	p := newTestCompetitionParam(key, "val", "", domain.CompetitionParamTypeString)

	d.configRepo.EXPECT().GetAll(mock.Anything).Return([]*domain.CompetitionParam{p}, nil)

	uc := d.createCompetitionParamUseCase()
	got := uc.GetString(ctx, key, "default")

	assert.Equal(t, "val", got)
}

func TestCompetitionParamUseCase_GetString_Error(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	key := "missing"
	defaultVal := "def"

	d.configRepo.EXPECT().GetAll(mock.Anything).Return([]*domain.CompetitionParam{}, nil)
	d.configRepo.EXPECT().GetByKey(mock.Anything, key).Return(nil, apperr.ErrCompetitionParamNotFound)

	uc := d.createCompetitionParamUseCase()
	got := uc.GetString(ctx, key, defaultVal)

	assert.Equal(t, defaultVal, got)
}

func TestCompetitionParamUseCase_GetInt_Success(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	key := "k"
	p := newTestCompetitionParam(key, "42", "", domain.CompetitionParamTypeInt)

	d.configRepo.EXPECT().GetAll(mock.Anything).Return([]*domain.CompetitionParam{p}, nil)

	uc := d.createCompetitionParamUseCase()
	got := uc.GetInt(ctx, key, 0)

	assert.Equal(t, 42, got)
}

func TestCompetitionParamUseCase_GetInt_Error(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	key := "missing"
	defaultVal := 10

	d.configRepo.EXPECT().GetAll(mock.Anything).Return([]*domain.CompetitionParam{}, nil)
	d.configRepo.EXPECT().GetByKey(mock.Anything, key).Return(nil, apperr.ErrCompetitionParamNotFound)

	uc := d.createCompetitionParamUseCase()
	got := uc.GetInt(ctx, key, defaultVal)

	assert.Equal(t, defaultVal, got)
}

func TestCompetitionParamUseCase_GetBool_Success(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	key := "k"
	p := newTestCompetitionParam(key, "true", "", domain.CompetitionParamTypeBool)

	d.configRepo.EXPECT().GetAll(mock.Anything).Return([]*domain.CompetitionParam{p}, nil)

	uc := d.createCompetitionParamUseCase()
	got := uc.GetBool(ctx, key, false)

	assert.True(t, got)
}

func TestCompetitionParamUseCase_GetBool_Error(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	key := "missing"

	d.configRepo.EXPECT().GetAll(mock.Anything).Return([]*domain.CompetitionParam{}, nil)
	d.configRepo.EXPECT().GetByKey(mock.Anything, key).Return(nil, apperr.ErrCompetitionParamNotFound)

	uc := d.createCompetitionParamUseCase()
	got := uc.GetBool(ctx, key, true)

	assert.True(t, got)
}

func TestCompetitionParamUseCase_GetByCategory_ReturnsOnlyCategoryTheme(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	theme1 := newTestCompetitionParam("theme_color_primary", "#fff", "", domain.CompetitionParamTypeString)
	theme1.Category = "theme"
	theme2 := newTestCompetitionParam("theme_dark_mode", "true", "", domain.CompetitionParamTypeBool)
	theme2.Category = "theme"

	d.configRepo.EXPECT().GetByCategory(mock.Anything, "theme").Return([]*domain.CompetitionParam{theme1, theme2}, nil)

	uc := d.createCompetitionParamUseCase()
	got, err := uc.GetByCategory(ctx, "theme")

	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(got), 2)

	for _, p := range got {
		assert.Equal(t, "theme", p.Category)
	}
}

func TestCompetitionParamUseCase_GetByCategory_InvalidCategory_ReturnsError(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()

	uc := d.createCompetitionParamUseCase()
	got, err := uc.GetByCategory(ctx, "invalid")

	assert.Error(t, err)
	assert.Nil(t, got)

	var ve *apperr.ValidationError
	assert.ErrorAs(t, err, &ve)
}

func TestCompetitionParamUseCase_GetAfterSet_ReturnsValue(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	key, value := "my_key", "my_value"
	actorID := uuid.New()
	afterSet := newTestCompetitionParam(key, value, "desc", domain.CompetitionParamTypeString)

	d.configRepo.EXPECT().Upsert(mock.Anything, mock.Anything).Return(nil)
	d.auditLogRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)
	d.configRepo.EXPECT().GetAll(mock.Anything).Return([]*domain.CompetitionParam{afterSet}, nil)

	uc := d.createCompetitionParamUseCase()
	err := uc.Set(ctx, competitionParamSetParams(key, value, "desc", domain.CompetitionParamTypeString, "", actorID, ""))
	assert.NoError(t, err)
	got, err := uc.Get(ctx, key)
	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, key, got.Key)
	assert.Equal(t, value, got.Value)
}

func TestCompetitionParamUseCase_GetAll_IncludesDefaults(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()

	d.configRepo.EXPECT().GetAll(mock.Anything).Return([]*domain.CompetitionParam{}, nil)

	uc := d.createCompetitionParamUseCase()
	got, err := uc.GetAll(ctx)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(got), domain.ConfigRegistryCount())

	seen := make(map[string]struct{})

	for _, p := range got {
		seen[p.Key] = struct{}{}
	}

	domain.RangeConfigRegistry(func(k string, _ domain.ConfigDef) bool {
		assert.Contains(t, seen, k, "GetAll should include registry key %q", k)

		return true
	})
}
