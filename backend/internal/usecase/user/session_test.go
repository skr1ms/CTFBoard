package user

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUserUseCase_Logout_RevokesAccessWithoutRefreshToken(t *testing.T) {
	t.Parallel()

	d := newUserTestDeps(t)
	uc := d.createUseCase()
	accessToken := "access-token"

	d.jwtService.EXPECT().RevokeAccessToken(context.Background(), accessToken).Return(nil).Once()

	err := uc.Logout(context.Background(), "", &accessToken)

	require.NoError(t, err)
}

func TestUserUseCase_Logout_IgnoresRefreshRevokeError(t *testing.T) {
	t.Parallel()

	d := newUserTestDeps(t)
	uc := d.createUseCase()

	d.jwtService.EXPECT().RevokeRefreshToken(context.Background(), "stale-refresh").Return(errors.New("expired")).Once()

	err := uc.Logout(context.Background(), "stale-refresh", nil)

	require.NoError(t, err)
}
