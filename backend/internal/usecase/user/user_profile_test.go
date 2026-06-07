package user

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	validation "github.com/TakuyaYagam1/AstroCTFb/pkg/validator"
)

func TestUserUseCase_GetByID_Success(t *testing.T) {
	t.Parallel()
	d := newUserTestDeps(t)

	userID := uuid.New()
	expectedUser := &domain.User{
		ID:       userID,
		Username: "testuser",
		Email:    "test@example.com",
	}

	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(expectedUser, nil)

	uc := d.createUseCase()

	user, err := uc.GetByID(context.Background(), userID)

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, expectedUser.ID, user.ID)
	assert.Equal(t, expectedUser.Username, user.Username)
}

func TestUserUseCase_GetByID_Error(t *testing.T) {
	t.Parallel()
	d := newUserTestDeps(t)

	userID := uuid.New()
	expectedError := assert.AnError

	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(nil, expectedError)

	uc := d.createUseCase()

	user, err := uc.GetByID(context.Background(), userID)

	assert.Error(t, err)
	assert.Nil(t, user)
}

func TestUserUseCase_GetProfile_Success(t *testing.T) {
	t.Parallel()
	d := newUserTestDeps(t)

	userID := uuid.New()
	user := &domain.User{
		ID:       userID,
		Username: "testuser",
		Email:    "test@example.com",
	}

	solves := []*domain.Solve{newTestSolve(userID, uuid.New())}

	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil)
	d.solveRepo.EXPECT().GetByUserID(mock.Anything, userID).Return(solves, nil)

	uc := d.createUseCase()

	profile, err := uc.GetProfile(context.Background(), userID)

	assert.NoError(t, err)
	assert.NotNil(t, profile)
	assert.Equal(t, user.Username, profile.User.Username)
	assert.Empty(t, profile.User.PasswordHash)
	assert.Len(t, profile.Solves, 1)
}

func TestUserUseCase_GetProfile_FiltersPublicCustomFields(t *testing.T) {
	t.Parallel()
	d := newUserTestDeps(t)

	userID := uuid.New()
	publicFieldID := uuid.New()
	numberFieldID := uuid.New()
	privateFieldID := uuid.New()
	user := &domain.User{
		ID:       userID,
		Username: "testuser",
		Email:    "test@example.com",
	}

	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil)
	d.solveRepo.EXPECT().GetByUserID(mock.Anything, userID).Return([]*domain.Solve{}, nil)

	uc := NewUserUseCase(UserDeps{
		UserRepo:   d.userRepo,
		SolveRepo:  d.solveRepo,
		TM:         d.tm,
		JWTService: d.jwtService,
		FieldRepo: &fakeFieldRepo{fields: []*domain.Field{
			{ID: publicFieldID, EntityType: domain.EntityTypeUser, FieldType: domain.FieldTypeJSON, Public: true},
			{ID: numberFieldID, EntityType: domain.EntityTypeUser, FieldType: domain.FieldTypeNumber, Public: true},
			{ID: privateFieldID, EntityType: domain.EntityTypeUser, Public: false},
		}},
		FieldValueRepo: &fakeFieldValueRepo{values: map[uuid.UUID]string{
			publicFieldID:  `{"rank":3,"roles":["web"]}`,
			numberFieldID:  "42",
			privateFieldID: "hidden",
		}},
	})

	profile, err := uc.GetProfile(context.Background(), userID)

	assert.NoError(t, err)
	assert.Equal(t, usecase.CustomFieldValues{
		publicFieldID.String(): map[string]any{
			"rank":  float64(3),
			"roles": []any{"web"},
		},
		numberFieldID.String(): int64(42),
	}, profile.CustomFields)
}

func TestUserUseCase_GetProfile_GetByIDError(t *testing.T) {
	t.Parallel()
	d := newUserTestDeps(t)

	userID := uuid.New()
	expectedError := assert.AnError

	d.solveRepo.EXPECT().GetByUserID(mock.Anything, userID).Return([]*domain.Solve{}, nil)
	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Run(func(_ context.Context, _ uuid.UUID) {
		time.Sleep(1 * time.Millisecond)
	}).Return(nil, expectedError)

	uc := d.createUseCase()

	profile, err := uc.GetProfile(context.Background(), userID)

	assert.Error(t, err)
	assert.Nil(t, profile)
}

func TestUserUseCase_UpdateProfile_CustomFieldsPartialUpsert(t *testing.T) {
	t.Parallel()
	d := newUserTestDeps(t)

	userID := uuid.New()
	fieldID := uuid.New()
	existingFieldID := uuid.New()
	user := &domain.User{
		ID:           userID,
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hash",
	}
	fieldValues := &fakeFieldValueRepo{values: map[uuid.UUID]string{existingFieldID: "kept"}}

	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil).Once()
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, userID).Return(nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil).Once()
	d.userRepo.EXPECT().UpdateProfile(mock.Anything, userID, (*string)(nil), (*string)(nil), (*string)(nil)).Return(nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil).Once()

	uc := NewUserUseCase(UserDeps{
		UserRepo:   d.userRepo,
		TM:         d.tm,
		JWTService: d.jwtService,
		FieldValidator: fakeProfileFieldValidator{
			validateEditable: func(_ context.Context, entityType domain.EntityType, values map[uuid.UUID]any) error {
				assert.Equal(t, domain.EntityTypeUser, entityType)
				assert.Equal(t, map[uuid.UUID]any{fieldID: " updated \x00 "}, values)

				return nil
			},
		},
		FieldRepo: &fakeFieldRepo{fields: []*domain.Field{
			{ID: existingFieldID, EntityType: domain.EntityTypeUser, FieldType: domain.FieldTypeText},
			{ID: fieldID, EntityType: domain.EntityTypeUser, FieldType: domain.FieldTypeText},
		}},
		FieldValueRepo: fieldValues,
	})

	me, err := uc.UpdateProfile(context.Background(), usecase.UserProfileUpdateParams{
		UserID:       userID,
		CustomFields: &usecase.CustomFieldValues{fieldID.String(): " updated \x00 "},
	})

	assert.NoError(t, err)
	assert.Equal(t, usecase.CustomFieldValues{
		existingFieldID.String(): "kept",
		fieldID.String():         "updated",
	}, me.CustomFields)
	assert.Equal(t, map[string]string{fieldID.String(): "updated"}, fieldValues.lastUpsert)
}

type fakeFieldRepo struct {
	fields []*domain.Field
}

func (r *fakeFieldRepo) Create(context.Context, *domain.Field) error {
	return nil
}

func (r *fakeFieldRepo) GetByID(context.Context, uuid.UUID) (*domain.Field, error) {
	return nil, assert.AnError
}

func (r *fakeFieldRepo) GetByEntityType(_ context.Context, entityType domain.EntityType) ([]*domain.Field, error) {
	out := make([]*domain.Field, 0, len(r.fields))
	for _, field := range r.fields {
		if field.EntityType == entityType {
			out = append(out, field)
		}
	}

	return out, nil
}

func (r *fakeFieldRepo) GetAll(context.Context) ([]*domain.Field, error) {
	return r.fields, nil
}

func (r *fakeFieldRepo) Update(context.Context, *domain.Field) error {
	return nil
}

func (r *fakeFieldRepo) Delete(context.Context, uuid.UUID) error {
	return nil
}

type fakeFieldValueRepo struct {
	values     map[uuid.UUID]string
	lastUpsert map[string]string
}

func (r *fakeFieldValueRepo) GetByEntityID(context.Context, uuid.UUID) ([]*domain.FieldValue, error) {
	out := make([]*domain.FieldValue, 0, len(r.values))
	for fieldID, value := range r.values {
		out = append(out, &domain.FieldValue{ID: uuid.New(), FieldID: fieldID, Value: value})
	}

	return out, nil
}

func (r *fakeFieldValueRepo) GetAll(context.Context) ([]*domain.FieldValue, error) {
	return nil, nil
}

func (r *fakeFieldValueRepo) SetValues(context.Context, uuid.UUID, map[string]string) error {
	return nil
}

func (r *fakeFieldValueRepo) UpsertValues(_ context.Context, _ uuid.UUID, values map[string]string) error {
	if r.values == nil {
		r.values = make(map[uuid.UUID]string, len(values))
	}

	r.lastUpsert = make(map[string]string, len(values))
	for fieldIDStr, value := range values {
		fieldID, err := uuid.Parse(fieldIDStr)
		if err != nil {
			return err
		}

		r.values[fieldID] = value
		r.lastUpsert[fieldIDStr] = value
	}

	return nil
}

func (r *fakeFieldValueRepo) DeleteByEntityID(context.Context, uuid.UUID) error {
	return nil
}

type fakeProfileFieldValidator struct {
	validateEditable func(context.Context, domain.EntityType, map[uuid.UUID]any) error
}

func (v fakeProfileFieldValidator) ValidateValues(context.Context, domain.EntityType, map[uuid.UUID]any) (map[uuid.UUID]string, error) {
	return nil, nil
}

func (v fakeProfileFieldValidator) ValidateEditableValues(ctx context.Context, entityType domain.EntityType, values map[uuid.UUID]any) (map[uuid.UUID]string, error) {
	if v.validateEditable != nil {
		if err := v.validateEditable(ctx, entityType, values); err != nil {
			return nil, err
		}
	}

	out := make(map[uuid.UUID]string, len(values))
	for fieldID, value := range values {
		out[fieldID] = validation.SanitizeCustomFieldValue(value.(string))
	}

	return out, nil
}

func TestUserUseCase_GetProfile_GetByUserIDError(t *testing.T) {
	t.Parallel()
	d := newUserTestDeps(t)

	userID := uuid.New()
	user := &domain.User{
		ID:       userID,
		Username: "testuser",
		Email:    "test@example.com",
	}
	expectedError := assert.AnError

	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil)
	d.solveRepo.EXPECT().GetByUserID(mock.Anything, userID).Run(func(_ context.Context, _ uuid.UUID) {
		time.Sleep(1 * time.Millisecond)
	}).Return(nil, expectedError)

	uc := d.createUseCase()

	profile, err := uc.GetProfile(context.Background(), userID)

	assert.Error(t, err)
	assert.Nil(t, profile)
}
