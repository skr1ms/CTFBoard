package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/wahrwelt-kit/go-jwtkit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

const jwtRevocationBoundarySlack = 5 * time.Millisecond

func issueUserTokenPair(ctx context.Context, jwtService jwtkit.Service, user *domain.User, op string) (*usecase.TokenPair, error) {
	pair, err := jwtService.GenerateTokenPair(ctx, user.ID, string(user.Role))
	if err != nil {
		return nil, fmt.Errorf("%s - GenerateTokenPair: %w", op, err)
	}

	if shouldRetryDirectBanToken(ctx, jwtService, user, pair) {
		if err := waitForNextJWTIssueSecond(ctx); err != nil {
			return nil, fmt.Errorf("%s - waitForNextJWTIssueSecond: %w", op, err)
		}

		pair, err = jwtService.GenerateTokenPair(ctx, user.ID, string(user.Role))
		if err != nil {
			return nil, fmt.Errorf("%s - GenerateTokenPair retry: %w", op, err)
		}

		if _, err := jwtService.ValidateAccessToken(ctx, pair.AccessToken); errors.Is(err, jwtkit.ErrTokenRevoked) {
			return nil, fmt.Errorf("%s - ValidateAccessToken retry: %w", op, err)
		}
	}

	return tokenPairFromJWT(pair), nil
}

func issueUserTokenPairAfterRevocation(ctx context.Context, jwtService jwtkit.Service, user *domain.User, op string) (*usecase.TokenPair, error) {
	pair, err := issueUserTokenPair(ctx, jwtService, user, op)
	if err != nil {
		return nil, err
	}

	if !isAccessTokenRevoked(ctx, jwtService, pair.AccessToken) {
		return pair, nil
	}

	if err := waitForNextJWTIssueSecond(ctx); err != nil {
		return nil, fmt.Errorf("%s - waitForNextJWTIssueSecond: %w", op, err)
	}

	pair, err = issueUserTokenPair(ctx, jwtService, user, op+" retry")
	if err != nil {
		return nil, err
	}

	if isAccessTokenRevoked(ctx, jwtService, pair.AccessToken) {
		return nil, fmt.Errorf("%s - issued access token is still revoked", op)
	}

	return pair, nil
}

func shouldRetryDirectBanToken(ctx context.Context, jwtService jwtkit.Service, user *domain.User, pair *jwtkit.TokenPair) bool {
	if user == nil || !user.IsBanned || pair == nil {
		return false
	}

	return isAccessTokenRevoked(ctx, jwtService, pair.AccessToken)
}

func isAccessTokenRevoked(ctx context.Context, jwtService jwtkit.Service, accessToken string) bool {
	_, err := jwtService.ValidateAccessToken(ctx, accessToken)

	return errors.Is(err, jwtkit.ErrTokenRevoked)
}

func waitForNextJWTIssueSecond(ctx context.Context) error {
	now := time.Now()
	nextSecond := time.Unix(now.Unix()+1, 0).Add(jwtRevocationBoundarySlack)
	timer := time.NewTimer(time.Until(nextSecond))

	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
