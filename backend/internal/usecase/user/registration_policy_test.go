package user

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	userMock "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/user/mock"
)

type staticCompetitionParamUC struct {
	usecase.CompetitionParamUseCase

	strings map[string]string
	ints    map[string]int
}

func (c staticCompetitionParamUC) GetString(_ context.Context, key, defaultVal string) string {
	if c.strings == nil {
		return defaultVal
	}

	value, ok := c.strings[key]
	if !ok {
		return defaultVal
	}

	return value
}

func (c staticCompetitionParamUC) GetInt(_ context.Context, key string, defaultVal int) int {
	if c.ints == nil {
		return defaultVal
	}

	value, ok := c.ints[key]
	if !ok {
		return defaultVal
	}

	return value
}

func newPolicyTestUseCase(deps *userTestDeps, settingsRepo *userMock.MockSettingsRepository, cfg usecase.CompetitionParamUseCase) *UserUseCase {
	ucDeps := UserDeps{
		UserRepo:       deps.userRepo,
		TeamRepo:       deps.teamRepo,
		SolveRepo:      deps.solveRepo,
		TM:             deps.tm,
		JWTService:     deps.jwtService,
		CompParamUC:    cfg,
		BcryptCost:     bcrypt.MinCost,
		FieldValidator: nil,
		FieldValueRepo: nil,
	}

	if settingsRepo != nil {
		ucDeps.SettingsRepo = settingsRepo
	}

	return NewUserUseCase(ucDeps)
}

func setupPolicyRegisterTx(d *userTestDeps) {
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
}

func TestUserUseCase_Register_RegistrationVisibilityPrivate(t *testing.T) {
	t.Parallel()
	d := newUserTestDeps(t)
	settingsRepo := userMock.NewMockSettingsRepository(t)
	cfg := staticCompetitionParamUC{
		strings: map[string]string{"registration_visibility": "private"},
		ints:    map[string]int{"password_min_length": 8},
	}

	settingsRepo.EXPECT().Get(mock.Anything).Return(&domain.Settings{RegistrationOpen: true}, nil).Once()

	uc := newPolicyTestUseCase(d, settingsRepo, cfg)
	user, err := uc.Register(context.Background(), usecase.UserRegisterParams{
		Username: "privateuser",
		Email:    "private@example.com",
		Password: "ValidPass1",
	})

	require.Error(t, err)
	assert.Nil(t, user)
	assert.ErrorIs(t, err, apperr.ErrVisibilityForbidden)
}

func TestUserUseCase_Register_RegistrationCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		code string
		want error
	}{
		{name: "missing", code: "", want: apperr.ErrRegistrationCodeRequired},
		{name: "wrong", code: "wrong-sauce", want: apperr.ErrInvalidRegistrationCode},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			d := newUserTestDeps(t)
			settingsRepo := userMock.NewMockSettingsRepository(t)
			cfg := staticCompetitionParamUC{
				strings: map[string]string{
					"registration_visibility": "public",
					"registration_code":       "secret-sauce",
				},
				ints: map[string]int{"password_min_length": 8},
			}

			settingsRepo.EXPECT().Get(mock.Anything).Return(&domain.Settings{RegistrationOpen: true}, nil).Once()

			uc := newPolicyTestUseCase(d, settingsRepo, cfg)
			user, err := uc.Register(context.Background(), usecase.UserRegisterParams{
				Username:         "codeuser",
				Email:            "code@example.com",
				Password:         "ValidPass1",
				RegistrationCode: tt.code,
			})

			require.Error(t, err)
			assert.Nil(t, user)
			assert.ErrorIs(t, err, tt.want)
		})
	}
}

func TestUserUseCase_Register_RegistrationCodeValid(t *testing.T) {
	t.Parallel()
	d := newUserTestDeps(t)
	settingsRepo := userMock.NewMockSettingsRepository(t)
	cfg := staticCompetitionParamUC{
		strings: map[string]string{
			"registration_visibility": "public",
			"registration_code":       "secret-sauce",
		},
		ints: map[string]int{"password_min_length": 8},
	}

	setupPolicyRegisterTx(d)
	settingsRepo.EXPECT().Get(mock.Anything).Return(&domain.Settings{RegistrationOpen: true}, nil).Twice()
	d.userRepo.EXPECT().AcquireAdvisoryLock(mock.Anything, mock.Anything).Return(nil).Twice()
	d.userRepo.EXPECT().GetByUsername(mock.Anything, "codeuser").Return(nil, apperr.ErrUserNotFound).Once()
	d.userRepo.EXPECT().GetByEmail(mock.Anything, "code@example.com").Return(nil, apperr.ErrUserNotFound).Once()
	d.userRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Run(func(_ context.Context, u *domain.User) {
		u.ID = uuid.New()
	}).Once()

	uc := newPolicyTestUseCase(d, settingsRepo, cfg)
	user, err := uc.Register(context.Background(), usecase.UserRegisterParams{
		Username:         "codeuser",
		Email:            "code@example.com",
		Password:         "ValidPass1",
		RegistrationCode: "SECRET-SAUCE",
	})

	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, "codeuser", user.Username)
}

func TestUserUseCase_Register_MaxUsersReached(t *testing.T) {
	t.Parallel()
	d := newUserTestDeps(t)
	settingsRepo := userMock.NewMockSettingsRepository(t)
	cfg := staticCompetitionParamUC{
		strings: map[string]string{"registration_visibility": "public"},
		ints:    map[string]int{"password_min_length": 8},
	}

	settingsRepo.EXPECT().Get(mock.Anything).Return(&domain.Settings{RegistrationOpen: true, MaxUsers: 1}, nil).Once()
	d.userRepo.EXPECT().CountActiveUsers(mock.Anything).Return(int64(1), nil).Once()

	uc := newPolicyTestUseCase(d, settingsRepo, cfg)
	user, err := uc.Register(context.Background(), usecase.UserRegisterParams{
		Username: "fulluser",
		Email:    "full@example.com",
		Password: "ValidPass1",
	})

	require.Error(t, err)
	assert.Nil(t, user)
	assert.ErrorIs(t, err, apperr.ErrMaxUsersReached)
}

func TestUserUseCase_Register_ConfiguredPasswordMinLength(t *testing.T) {
	t.Parallel()
	d := newUserTestDeps(t)
	cfg := staticCompetitionParamUC{ints: map[string]int{"password_min_length": 12}}

	uc := newPolicyTestUseCase(d, nil, cfg)
	user, err := uc.Register(context.Background(), usecase.UserRegisterParams{
		Username: "shortpass",
		Email:    "short@example.com",
		Password: "ValidP1!",
	})

	var validationErr *apperr.ValidationError

	require.Error(t, err)
	assert.Nil(t, user)
	assert.ErrorAs(t, err, &validationErr)
}

func TestUserUseCase_AdminCreate_BypassesRegistrationPolicyAndUserCap(t *testing.T) {
	t.Parallel()
	d := newUserTestDeps(t)
	cfg := staticCompetitionParamUC{
		strings: map[string]string{
			"registration_visibility": "private",
			"registration_code":       "secret-sauce",
		},
		ints: map[string]int{"password_min_length": 8},
	}

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.userRepo.EXPECT().AcquireAdvisoryLock(mock.Anything, mock.Anything).Return(nil).Twice()
	d.userRepo.EXPECT().GetByUsername(mock.Anything, "adminmade").Return(nil, apperr.ErrUserNotFound).Once()
	d.userRepo.EXPECT().GetByEmail(mock.Anything, "adminmade@example.com").Return(nil, apperr.ErrUserNotFound).Once()
	d.userRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Run(func(_ context.Context, u *domain.User) {
		u.ID = uuid.New()
	}).Once()

	uc := newPolicyTestUseCase(d, nil, cfg)
	user, err := uc.AdminCreate(context.Background(), "adminmade", "adminmade@example.com", "ValidPass1", "user")

	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, "adminmade", user.Username)
}
