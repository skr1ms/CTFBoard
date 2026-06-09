package setup

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func TestSetupUseCaseIsComplete(t *testing.T) {
	t.Parallel()

	deps := newSetupTestDeps()
	deps.compParam.setupComplete = true

	got, err := deps.useCase().IsComplete(context.Background())

	require.NoError(t, err)
	assert.True(t, got)
	assert.Equal(t, 1, deps.compParam.getBoolCalls)
}

func TestSetupUseCaseCompleteSuccess(t *testing.T) {
	t.Parallel()

	deps := newSetupTestDeps()
	req := validSetupUseCaseRequest()

	got, err := deps.useCase().Complete(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, got)
	require.NotNil(t, got.TokenPair)
	assert.Equal(t, "access", got.TokenPair.AccessToken)
	assert.Equal(t, "refresh", got.TokenPair.RefreshToken)
	assert.Equal(t, deps.user.adminUser, got.User)

	assert.Equal(t, 1, deps.user.calls)
	assert.Equal(t, 1, deps.comp.getCalls)
	assert.Equal(t, 1, deps.comp.updateCalls)
	assert.Equal(t, req.CTFName, deps.comp.updateComp.Name)
	assert.Equal(t, domain.ModeTeamsOnly, deps.comp.updateComp.Mode)
	assert.Equal(t, req.MaxTeamSize, deps.comp.updateComp.MaxTeamSize)
	assert.Equal(t, deps.user.adminUser.ID, deps.comp.actorID)
	assert.Equal(t, req.ClientIP, deps.comp.clientIP)

	require.Len(t, deps.compParam.batchParams, 9)

	paramsByKey := make(map[string]*domain.CompetitionParam, len(deps.compParam.batchParams))
	for _, param := range deps.compParam.batchParams {
		paramsByKey[param.Key] = param
	}

	assert.Equal(t, req.CTFName, paramsByKey["ctf_name"].Value)
	assert.Equal(t, req.ScoreVisibility, paramsByKey["score_visibility"].Value)
	assert.Equal(t, "true", paramsByKey["email_verification_required"].Value)
	assert.Equal(t, req.Timezone, paramsByKey["timezone"].Value)
	assert.Equal(t, deps.user.adminUser.ID, deps.compParam.actorID)
	assert.Equal(t, req.ClientIP, deps.compParam.clientIP)

	assert.Equal(t, 1, deps.settings.getCalls)
	assert.Equal(t, 1, deps.settings.updateCalls)
	assert.Equal(t, req.CTFName, deps.settings.updateValue.AppName)
	assert.True(t, deps.settings.updateValue.VerifyEmails)
	assert.Equal(t, deps.user.adminUser.ID, deps.settings.actorID)
	assert.Equal(t, req.ClientIP, deps.settings.clientIP)

	assert.Equal(t, 1, deps.compParam.setCalls)
	assert.Equal(t, "setup_complete", deps.compParam.setParam.Key)
	assert.Equal(t, "true", deps.compParam.setParam.Value)
	assert.Equal(t, domain.CompetitionParamTypeBool, deps.compParam.setParam.ValueType)
	assert.Equal(t, deps.user.adminUser.ID, deps.compParam.setParam.ActorID)
	assert.Equal(t, req.ClientIP, deps.compParam.setParam.ClientIP)
	assert.Equal(t, 1, deps.jwt.calls)
	assert.Equal(t, 1, deps.tm.calls)
	assert.Equal(t, 1, deps.compParam.lockCalls)
}

func TestSetupUseCaseCompleteRejectsFlexibleModeBeforeSideEffects(t *testing.T) {
	t.Parallel()

	deps := newSetupTestDeps()
	req := validSetupUseCaseRequest()
	req.Mode = "flexible"

	got, err := deps.useCase().Complete(context.Background(), req)

	require.Error(t, err)
	assert.Nil(t, got)
	assert.Contains(t, err.Error(), "must be solo_only or teams_only")
	assert.Equal(t, 0, deps.user.calls)
	assert.Equal(t, 0, deps.comp.updateCalls)
	assert.Equal(t, 0, deps.compParam.setBatchCalls)
	assert.Equal(t, 0, deps.settings.updateCalls)
	assert.Equal(t, 0, deps.jwt.calls)
}

func TestSetupUseCaseCompleteAlreadyCompleteFastPath(t *testing.T) {
	t.Parallel()

	deps := newSetupTestDeps()
	deps.compParam.setupComplete = true

	got, err := deps.useCase().Complete(context.Background(), validSetupUseCaseRequest())

	require.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrSetupAlreadyComplete)
	assert.Nil(t, got)
	assert.Equal(t, 0, deps.user.calls)
	assert.Equal(t, 0, deps.comp.updateCalls)
}

func TestSetupUseCaseCompleteSerializesConcurrentCalls(t *testing.T) {
	deps := newSetupTestDeps()
	uc := deps.useCase()
	req := validSetupUseCaseRequest()

	var wg sync.WaitGroup
	wg.Add(2)

	results := make(chan error, 2)

	for range 2 {
		go func() {
			defer wg.Done()

			_, err := uc.Complete(context.Background(), req)
			results <- err
		}()
	}

	wg.Wait()
	close(results)

	successes := 0
	alreadyComplete := 0

	for err := range results {
		if err == nil {
			successes++

			continue
		}

		if errors.Is(err, apperr.ErrSetupAlreadyComplete) {
			alreadyComplete++
		}
	}

	assert.Equal(t, 1, successes)
	assert.Equal(t, 1, alreadyComplete)
	assert.Equal(t, 1, deps.user.calls)
	assert.Equal(t, 1, deps.compParam.setCalls)
}

func TestSetupUseCaseCompleteRollsBackWhenTokenGenerationFails(t *testing.T) {
	t.Parallel()

	deps := newSetupTestDeps()
	deps.jwt.err = errors.New("token generation failed")
	deps.tm.rollback = func() {
		deps.compParam.mu.Lock()
		defer deps.compParam.mu.Unlock()

		deps.compParam.setupComplete = false
	}

	got, err := deps.useCase().Complete(context.Background(), validSetupUseCaseRequest())

	require.Error(t, err)
	assert.ErrorIs(t, err, deps.jwt.err)
	assert.Nil(t, got)
	assert.Equal(t, 1, deps.tm.calls)
	assert.False(t, deps.compParam.setupComplete)
}

func TestSetupUseCaseCompleteWrapsDependencyErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*setupTestDeps, error)
	}{
		{
			name: "admin create",
			mutate: func(d *setupTestDeps, err error) {
				d.user.err = err
			},
		},
		{
			name: "competition get",
			mutate: func(d *setupTestDeps, err error) {
				d.comp.getErr = err
			},
		},
		{
			name: "competition update",
			mutate: func(d *setupTestDeps, err error) {
				d.comp.updateErr = err
			},
		},
		{
			name: "config set batch",
			mutate: func(d *setupTestDeps, err error) {
				d.compParam.setBatchErr = err
			},
		},
		{
			name: "settings get",
			mutate: func(d *setupTestDeps, err error) {
				d.settings.getErr = err
			},
		},
		{
			name: "settings update",
			mutate: func(d *setupTestDeps, err error) {
				d.settings.updateErr = err
			},
		},
		{
			name: "setup complete set",
			mutate: func(d *setupTestDeps, err error) {
				d.compParam.setErr = err
			},
		},
		{
			name: "jwt generation",
			mutate: func(d *setupTestDeps, err error) {
				d.jwt.err = err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			expectedErr := errors.New(tt.name + " failed")
			deps := newSetupTestDeps()
			tt.mutate(deps, expectedErr)

			got, err := deps.useCase().Complete(context.Background(), validSetupUseCaseRequest())

			require.Error(t, err)
			assert.ErrorIs(t, err, expectedErr)
			assert.Nil(t, got)
		})
	}
}
