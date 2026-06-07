package team

import (
	"context"
	"errors"
	"maps"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/competition"
	validation "github.com/TakuyaYagam1/AstroCTFb/pkg/validator"
)

func TestTeamUseCase_GetProfile_FiltersPublicCustomFields(t *testing.T) {
	t.Parallel()

	d := newTeamTestDeps(t)
	teamID := uuid.New()
	captainID := uuid.New()
	publicFieldID := uuid.New()
	editableFieldID := uuid.New()
	hiddenFieldID := uuid.New()

	fields := []*domain.Field{
		{ID: publicFieldID, EntityType: domain.EntityTypeTeam, FieldType: domain.FieldTypeJSON, Public: true},
		{ID: editableFieldID, EntityType: domain.EntityTypeTeam, FieldType: domain.FieldTypeText, Editable: true},
		{ID: hiddenFieldID, EntityType: domain.EntityTypeTeam, FieldType: domain.FieldTypeText},
	}
	values := []*domain.FieldValue{
		{FieldID: publicFieldID, EntityID: teamID, Value: `{"division":"open"}`},
		{FieldID: editableFieldID, EntityID: teamID, Value: "private-editable"},
		{FieldID: hiddenFieldID, EntityID: teamID, Value: "hidden"},
	}

	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(newTestTeam(teamID, "Team", captainID, uuid.New(), false), nil).Once()

	uc := newTeamUseCaseWithCustomFields(d, &fakeTeamFieldRepo{fields: fields}, &fakeTeamFieldValueRepo{values: values}, nil)

	profile, err := uc.GetProfile(context.Background(), teamID)

	require.NoError(t, err)
	require.NotNil(t, profile)
	assert.Equal(t, usecase.CustomFieldValues{
		publicFieldID.String(): map[string]any{"division": "open"},
	}, profile.CustomFields)
}

func TestTeamUseCase_GetMyTeam_FiltersSelfCustomFields(t *testing.T) {
	t.Parallel()

	d := newTeamTestDeps(t)
	userID := uuid.New()
	teamID := uuid.New()
	publicFieldID := uuid.New()
	editableFieldID := uuid.New()
	hiddenFieldID := uuid.New()

	user := newTestUser(userID, &teamID, "cap", "cap@example.com")
	team := newTestTeam(teamID, "Team", userID, uuid.New(), false)
	fields := []*domain.Field{
		{ID: publicFieldID, EntityType: domain.EntityTypeTeam, FieldType: domain.FieldTypeNumber, Public: true},
		{ID: editableFieldID, EntityType: domain.EntityTypeTeam, FieldType: domain.FieldTypeBoolean, Editable: true},
		{ID: hiddenFieldID, EntityType: domain.EntityTypeTeam, FieldType: domain.FieldTypeText},
	}
	values := []*domain.FieldValue{
		{FieldID: publicFieldID, EntityID: teamID, Value: "7"},
		{FieldID: editableFieldID, EntityID: teamID, Value: "true"},
		{FieldID: hiddenFieldID, EntityID: teamID, Value: "hidden"},
	}

	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil).Once()
	d.userRepo.EXPECT().GetByTeamID(mock.Anything, teamID).Return([]*domain.User{user}, nil).Once()
	d.compRepo.EXPECT().Get(mock.Anything).Return(&domain.Competition{MinTeamSize: 0}, nil).Once()

	uc := newTeamUseCaseWithCustomFields(d, &fakeTeamFieldRepo{fields: fields}, &fakeTeamFieldValueRepo{values: values}, nil)

	me, err := uc.GetMyTeam(context.Background(), userID)

	require.NoError(t, err)
	require.NotNil(t, me)
	assert.Equal(t, usecase.CustomFieldValues{
		publicFieldID.String():   int64(7),
		editableFieldID.String(): true,
	}, me.CustomFields)
}

func TestTeamUseCase_UpdateMyTeam_CustomFieldsOnly_SkipsTeamSwitchGuard(t *testing.T) {
	t.Parallel()

	d := newTeamTestDeps(t)
	captainID := uuid.New()
	teamID := uuid.New()
	fieldID := uuid.New()

	user := newTestUser(captainID, &teamID, "cap", "cap@example.com")
	team := newTestTeam(teamID, "Team", captainID, uuid.New(), false)
	fieldRepo := &fakeTeamFieldRepo{fields: []*domain.Field{
		{ID: fieldID, EntityType: domain.EntityTypeTeam, FieldType: domain.FieldTypeText, Editable: true},
	}}
	valueRepo := &fakeTeamFieldValueRepo{}
	validator := &fakeTeamFieldValidator{}

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, captainID).Return(nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, captainID).Return(user, nil).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil).Once()

	uc := newTeamUseCaseWithCustomFields(d, fieldRepo, valueRepo, validator)
	customFields := usecase.CustomFieldValues{fieldID.String(): "  captain note\x00 "}

	profile, err := uc.UpdateMyTeam(context.Background(), captainID, usecase.TeamUpdateParams{CustomFields: &customFields})

	require.NoError(t, err)
	require.NotNil(t, profile)
	assert.Equal(t, "Team", profile.Team.Name)
	assert.Equal(t, domain.EntityTypeTeam, validator.entityType)
	assert.Equal(t, map[uuid.UUID]any{fieldID: "  captain note\x00 "}, validator.values)
	assert.Equal(t, map[string]string{fieldID.String(): "captain note"}, valueRepo.upserted)
	assert.Equal(t, usecase.CustomFieldValues{fieldID.String(): "captain note"}, profile.CustomFields)
}

func TestTeamUseCase_UpdateMyTeam_SameNameAndCustomFields_SkipsTeamSwitchGuard(t *testing.T) {
	t.Parallel()

	d := newTeamTestDeps(t)
	captainID := uuid.New()
	teamID := uuid.New()
	fieldID := uuid.New()
	currentName := "Team"

	user := newTestUser(captainID, &teamID, "cap", "cap@example.com")
	team := newTestTeam(teamID, currentName, captainID, uuid.New(), false)
	fieldRepo := &fakeTeamFieldRepo{fields: []*domain.Field{
		{ID: fieldID, EntityType: domain.EntityTypeTeam, FieldType: domain.FieldTypeText, Editable: true},
	}}
	valueRepo := &fakeTeamFieldValueRepo{}
	validator := &fakeTeamFieldValidator{}

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, captainID).Return(nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, captainID).Return(user, nil).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil).Once()

	uc := newTeamUseCaseWithCustomFields(d, fieldRepo, valueRepo, validator)
	customFields := usecase.CustomFieldValues{fieldID.String(): "value"}

	profile, err := uc.UpdateMyTeam(context.Background(), captainID, usecase.TeamUpdateParams{
		Name:         &currentName,
		CustomFields: &customFields,
	})

	require.NoError(t, err)
	require.NotNil(t, profile)
	assert.Equal(t, currentName, profile.Team.Name)
	assert.Equal(t, usecase.CustomFieldValues{fieldID.String(): "value"}, profile.CustomFields)
}

func TestTeamUseCase_UpdateMyTeam_CustomFieldsRejectsValidatorError(t *testing.T) {
	t.Parallel()

	d := newTeamTestDeps(t)
	captainID := uuid.New()
	teamID := uuid.New()
	fieldID := uuid.New()
	user := newTestUser(captainID, &teamID, "cap", "cap@example.com")
	team := newTestTeam(teamID, "Team", captainID, uuid.New(), false)
	valueRepo := &fakeTeamFieldValueRepo{}
	validator := &fakeTeamFieldValidator{err: apperr.NewValidationErrorf("field is not editable")}

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, captainID).Return(nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, captainID).Return(user, nil).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil).Once()

	uc := newTeamUseCaseWithCustomFields(d, &fakeTeamFieldRepo{}, valueRepo, validator)
	customFields := usecase.CustomFieldValues{fieldID.String(): "value"}

	profile, err := uc.UpdateMyTeam(context.Background(), captainID, usecase.TeamUpdateParams{CustomFields: &customFields})

	require.Error(t, err)
	assert.ErrorContains(t, err, "field is not editable")
	assert.Nil(t, profile)
	assert.Nil(t, valueRepo.upserted)
}

func newTeamUseCaseWithCustomFields(
	d *teamTestDeps,
	fieldRepo *fakeTeamFieldRepo,
	fieldValueRepo *fakeTeamFieldValueRepo,
	validator *fakeTeamFieldValidator,
) *TeamUseCase {
	return NewTeamUseCase(TeamDeps{
		TeamRepo:           d.teamRepo,
		UserRepo:           d.userRepo,
		SolveRepo:          d.solveRepo,
		SubmissionRepo:     d.submissionRepo,
		AwardRepo:          d.awardRepo,
		CompRepo:           d.compRepo,
		SettingsGetter:     d.SettingsRepo,
		ChallengeRepo:      d.challengeRepo,
		TM:                 d.tm,
		Guard:              competition.NewGuard(d.compRepo),
		FieldRepo:          fieldRepo,
		FieldValueRepo:     fieldValueRepo,
		FieldValidator:     validator,
		DefaultMaxTeamSize: 10,
	})
}

type fakeTeamFieldRepo struct {
	fields []*domain.Field
}

func (r *fakeTeamFieldRepo) Create(context.Context, *domain.Field) error { return nil }
func (r *fakeTeamFieldRepo) GetByID(context.Context, uuid.UUID) (*domain.Field, error) {
	return nil, apperr.ErrFieldNotFound
}
func (r *fakeTeamFieldRepo) GetByEntityType(context.Context, domain.EntityType) ([]*domain.Field, error) {
	return r.fields, nil
}
func (r *fakeTeamFieldRepo) GetAll(context.Context) ([]*domain.Field, error) { return r.fields, nil }
func (r *fakeTeamFieldRepo) Update(context.Context, *domain.Field) error     { return nil }
func (r *fakeTeamFieldRepo) Delete(context.Context, uuid.UUID) error         { return nil }

type fakeTeamFieldValueRepo struct {
	values   []*domain.FieldValue
	upserted map[string]string
}

func (r *fakeTeamFieldValueRepo) GetByEntityID(_ context.Context, entityID uuid.UUID) ([]*domain.FieldValue, error) {
	if len(r.upserted) > 0 {
		values := make([]*domain.FieldValue, 0, len(r.upserted))
		for fieldIDRaw, value := range r.upserted {
			fieldID, err := uuid.Parse(fieldIDRaw)
			if err != nil {
				return nil, err
			}

			values = append(values, &domain.FieldValue{FieldID: fieldID, EntityID: entityID, Value: value})
		}

		return values, nil
	}

	return r.values, nil
}
func (r *fakeTeamFieldValueRepo) GetAll(context.Context) ([]*domain.FieldValue, error) {
	return r.values, nil
}
func (r *fakeTeamFieldValueRepo) SetValues(context.Context, uuid.UUID, map[string]string) error {
	return errors.New("not implemented")
}
func (r *fakeTeamFieldValueRepo) UpsertValues(_ context.Context, _ uuid.UUID, values map[string]string) error {
	r.upserted = maps.Clone(values)

	return nil
}
func (r *fakeTeamFieldValueRepo) DeleteByEntityID(context.Context, uuid.UUID) error { return nil }

type fakeTeamFieldValidator struct {
	entityType domain.EntityType
	values     map[uuid.UUID]any
	err        error
}

func (v *fakeTeamFieldValidator) ValidateEditableValues(
	_ context.Context,
	entityType domain.EntityType,
	values map[uuid.UUID]any,
) (map[uuid.UUID]string, error) {
	v.entityType = entityType
	v.values = maps.Clone(values)

	if v.err != nil {
		return nil, v.err
	}

	out := make(map[uuid.UUID]string, len(values))
	for fieldID, value := range values {
		out[fieldID] = validation.SanitizeCustomFieldValue(value.(string))
	}

	return out, nil
}
