package user

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

func TestUserUseCase_AdminUpdate_EmailChangeResetsVerificationByDefault(t *testing.T) {
	t.Parallel()
	d := newUserTestDeps(t)

	userID := uuid.New()
	newEmail := "New.Email+Alias@Example.COM"
	normalizedEmail := "new.email+alias@example.com"
	current := &domain.User{
		ID:         userID,
		Username:   "player",
		Email:      "old@example.com",
		Role:       domain.RoleUser,
		IsVerified: true,
	}
	updated := *current
	updated.Email = normalizedEmail
	updated.IsVerified = false

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, userID).Return(nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(current, nil).Once()
	d.userRepo.EXPECT().GetByEmail(mock.Anything, normalizedEmail).Return(nil, apperr.ErrUserNotFound).Once()
	d.userRepo.EXPECT().UpdateAdmin(
		mock.Anything,
		userID,
		(*string)(nil),
		mock.MatchedBy(func(email *string) bool {
			return email != nil && *email == normalizedEmail
		}),
		(*string)(nil),
		(*string)(nil),
		mock.MatchedBy(func(verified *bool) bool {
			return verified != nil && !*verified
		}),
	).Return(nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(&updated, nil).Once()

	uc := d.createUseCase()
	user, err := uc.AdminUpdate(context.Background(), userID, nil, &newEmail, nil, nil, nil)

	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, normalizedEmail, user.Email)
	assert.False(t, user.IsVerified)
}
