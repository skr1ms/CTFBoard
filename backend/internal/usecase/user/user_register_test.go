package user

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	userMock "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/user/mock"
)

func TestUserUseCase_Register(t *testing.T) {
	t.Parallel()

	for _, tt := range registerTestCases() {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			runRegisterTest(t, tt)
		})
	}
}

func TestUserUseCase_Register_RegistrationClosed(t *testing.T) {
	t.Parallel()
	d := newUserTestDeps(t)
	settingsRepo := userMock.NewMockSettingsRepository(t)

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	settingsRepo.EXPECT().Get(mock.Anything).Return(&domain.Settings{RegistrationOpen: false}, nil).Once()
	uc := NewUserUseCase(UserDeps{
		UserRepo: d.userRepo, TeamRepo: d.teamRepo, SolveRepo: d.solveRepo,
		TM: d.tm, JWTService: d.jwtService, FieldValidator: nil, FieldValueRepo: nil,
		SettingsRepo: settingsRepo,
	})
	user, err := uc.Register(context.Background(), usecase.UserRegisterParams{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	})
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.ErrorIs(t, err, apperr.ErrRegistrationClosed)
}
