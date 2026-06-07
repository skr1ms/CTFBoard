package competition

import (
	"context"
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

type shareConfigStub struct {
	strings map[string]string
	bools   map[string]bool
}

func (s shareConfigStub) GetString(_ context.Context, key, defaultVal string) string {
	if s.strings == nil {
		return defaultVal
	}

	if v, ok := s.strings[key]; ok {
		return v
	}

	return defaultVal
}

func (s shareConfigStub) GetBool(_ context.Context, key string, defaultVal bool) bool {
	if s.bools == nil {
		return defaultVal
	}

	if v, ok := s.bools[key]; ok {
		return v
	}

	return defaultVal
}

func newShareUseCaseForTest(d *competitionTestDeps, cfg shareConfigStub) *ShareUseCase {
	return NewShareUseCase(ShareDeps{
		SolveRepo:     d.solveRepo,
		ChallengeRepo: d.challengeRepo,
		UserRepo:      d.userRepo,
		TeamRepo:      d.teamRepo,
		CompParamUC:   cfg,
		BaseURL:       "http://localhost:8080",
		FrontendURL:   "http://localhost:3000",
		ShareSecret:   "share-secret",
	})
}

func TestShareUseCase_CreateSolveShare_ReturnsSignedURL(t *testing.T) {
	ctx := context.Background()
	d := newCompetitionTestDeps(t)
	uc := newShareUseCaseForTest(d, shareConfigStub{})

	userID := uuid.New()
	teamID := uuid.New()
	challengeID := uuid.New()
	solveID := uuid.New()
	solve := &domain.Solve{ID: solveID, UserID: userID, TeamID: teamID, ChallengeID: challengeID}
	challenge := &domain.Challenge{ID: challengeID, State: domain.ChallengeStateVisible}

	d.challengeRepo.EXPECT().GetByID(mock.Anything, challengeID).Return(challenge, nil)
	d.solveRepo.EXPECT().GetByTeamAndChallenge(mock.Anything, teamID, challengeID).Return(solve, nil)

	link, err := uc.CreateSolveShare(ctx, usecase.CreateShareParams{
		Type:        usecase.ShareTypeSolve,
		UserID:      userID,
		TeamID:      teamID,
		ChallengeID: challengeID,
	})
	require.NoError(t, err)
	require.NotNil(t, link)

	assert.Equal(t, usecase.ShareTypeSolve, link.Type)
	assert.Equal(t, solveID, link.SolveID)

	parsed, err := url.Parse(link.URL)
	require.NoError(t, err)

	assert.Equal(t, "/api/v1/shares/solve", parsed.Path)
	assert.Equal(t, solveID.String(), parsed.Query().Get("solve_id"))
	assert.Len(t, parsed.Query().Get("mac"), 64)
	assert.True(t, uc.verifySolveMAC(solveID, parsed.Query().Get("mac")))
}

func TestShareUseCase_CreateSolveShare_Disabled(t *testing.T) {
	d := newCompetitionTestDeps(t)
	uc := newShareUseCaseForTest(d, shareConfigStub{bools: map[string]bool{shareConfigEnabled: false}})

	_, err := uc.CreateSolveShare(context.Background(), usecase.CreateShareParams{
		Type:        usecase.ShareTypeSolve,
		UserID:      uuid.New(),
		TeamID:      uuid.New(),
		ChallengeID: uuid.New(),
	})
	require.ErrorIs(t, err, apperr.ErrSharesDisabled)
}

func TestShareUseCase_CreateSolveShare_Unsolved(t *testing.T) {
	ctx := context.Background()
	d := newCompetitionTestDeps(t)
	uc := newShareUseCaseForTest(d, shareConfigStub{})

	teamID := uuid.New()
	challengeID := uuid.New()
	challenge := &domain.Challenge{ID: challengeID, State: domain.ChallengeStateVisible}
	d.challengeRepo.EXPECT().GetByID(mock.Anything, challengeID).Return(challenge, nil)
	d.solveRepo.EXPECT().GetByTeamAndChallenge(mock.Anything, teamID, challengeID).Return(nil, apperr.ErrSolveNotFound)

	_, err := uc.CreateSolveShare(ctx, usecase.CreateShareParams{
		Type:        usecase.ShareTypeSolve,
		UserID:      uuid.New(),
		TeamID:      teamID,
		ChallengeID: challengeID,
	})
	require.ErrorIs(t, err, apperr.ErrSolutionAccessDenied)
}

func TestShareUseCase_CreateSolveShare_HiddenChallenge(t *testing.T) {
	ctx := context.Background()
	d := newCompetitionTestDeps(t)
	uc := newShareUseCaseForTest(d, shareConfigStub{})

	teamID := uuid.New()
	challengeID := uuid.New()
	challenge := &domain.Challenge{ID: challengeID, State: domain.ChallengeStateHidden}
	d.challengeRepo.EXPECT().GetByID(mock.Anything, challengeID).Return(challenge, nil)

	_, err := uc.CreateSolveShare(ctx, usecase.CreateShareParams{
		Type:        usecase.ShareTypeSolve,
		UserID:      uuid.New(),
		TeamID:      teamID,
		ChallengeID: challengeID,
	})
	require.ErrorIs(t, err, apperr.ErrChallengeNotFound)
}

func TestShareUseCase_ResolveSolveShare_ReturnsPublicPayload(t *testing.T) {
	ctx := context.Background()
	d := newCompetitionTestDeps(t)
	uc := newShareUseCaseForTest(d, shareConfigStub{
		strings: map[string]string{
			shareConfigCTFName:        "Astro CTF",
			shareConfigCTFDescription: "Space hacking season",
			shareConfigCTFLogo:        "https://cdn.example/logo.png",
		},
	})

	solveID := uuid.New()
	userID := uuid.New()
	teamID := uuid.New()
	challengeID := uuid.New()
	solvedAt := time.Now().UTC()
	solve := &domain.Solve{ID: solveID, UserID: userID, TeamID: teamID, ChallengeID: challengeID, PointsAtSolve: 420, SolvedAt: solvedAt}
	challenge := &domain.Challenge{ID: challengeID, Title: "Orbit", Category: "web", State: domain.ChallengeStateVisible}
	team := &domain.Team{ID: teamID, Name: "cosmos"}
	user := &domain.User{ID: userID, Username: "alice"}

	d.solveRepo.EXPECT().GetByID(mock.Anything, solveID).Return(solve, nil)
	d.challengeRepo.EXPECT().GetByID(mock.Anything, challengeID).Return(challenge, nil)
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil)
	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil)

	share, err := uc.ResolveSolveShare(ctx, solveID, uc.signSolve(solveID))
	require.NoError(t, err)
	require.NotNil(t, share)

	assert.Equal(t, solveID, share.SolveID)
	assert.Equal(t, "Orbit", share.ChallengeTitle)
	assert.Equal(t, "cosmos", share.TeamName)
	assert.Equal(t, "alice", share.Username)
	assert.Equal(t, "Astro CTF", share.CTFName)
	assert.Equal(t, "Space hacking season", share.CTFDescription)
	assert.Equal(t, "https://cdn.example/logo.png", share.CTFLogo)
	assert.Equal(t, "http://localhost:3000/register", share.RegisterURL)
	assert.Equal(t, 420, share.PointsAtSolve)
	assert.Equal(t, solvedAt, share.SolvedAt)
}

func TestShareUseCase_ResolveSolveShare_InvalidMACDoesNotHitRepos(t *testing.T) {
	d := newCompetitionTestDeps(t)
	uc := newShareUseCaseForTest(d, shareConfigStub{})

	_, err := uc.ResolveSolveShare(context.Background(), uuid.New(), "not-hex")
	require.ErrorIs(t, err, apperr.ErrShareNotFound)
}

func TestShareUseCase_ResolveSolveShare_HidesInvisibleSubjects(t *testing.T) {
	tests := []struct {
		name       string
		challenge  *domain.Challenge
		team       *domain.Team
		user       *domain.User
		wantErr    error
		expectTeam bool
		expectUser bool
	}{
		{
			name:      "hidden challenge",
			challenge: &domain.Challenge{ID: uuid.New(), State: domain.ChallengeStateHidden},
			wantErr:   apperr.ErrShareNotFound,
		},
		{
			name:       "hidden team",
			challenge:  &domain.Challenge{ID: uuid.New(), State: domain.ChallengeStateVisible},
			team:       &domain.Team{ID: uuid.New(), IsHidden: true},
			wantErr:    apperr.ErrShareNotFound,
			expectTeam: true,
		},
		{
			name:       "banned user",
			challenge:  &domain.Challenge{ID: uuid.New(), State: domain.ChallengeStateVisible},
			team:       &domain.Team{ID: uuid.New()},
			user:       &domain.User{ID: uuid.New(), IsBanned: true},
			wantErr:    apperr.ErrShareNotFound,
			expectTeam: true,
			expectUser: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			d := newCompetitionTestDeps(t)
			uc := newShareUseCaseForTest(d, shareConfigStub{})

			solveID := uuid.New()
			userID := uuid.New()
			teamID := uuid.New()
			challengeID := tt.challenge.ID
			tt.challenge.ID = challengeID

			if tt.team != nil {
				tt.team.ID = teamID
			}

			if tt.user != nil {
				tt.user.ID = userID
			}

			solve := &domain.Solve{ID: solveID, UserID: userID, TeamID: teamID, ChallengeID: challengeID}

			d.solveRepo.EXPECT().GetByID(mock.Anything, solveID).Return(solve, nil)
			d.challengeRepo.EXPECT().GetByID(mock.Anything, challengeID).Return(tt.challenge, nil)

			if tt.expectTeam {
				d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(tt.team, nil)
			}

			if tt.expectUser {
				d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(tt.user, nil)
			}

			_, err := uc.ResolveSolveShare(ctx, solveID, uc.signSolve(solveID))
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestShareUseCase_ResolveSolveShare_MapsMissingSolveToShareNotFound(t *testing.T) {
	d := newCompetitionTestDeps(t)
	uc := newShareUseCaseForTest(d, shareConfigStub{})
	solveID := uuid.New()

	d.solveRepo.EXPECT().GetByID(mock.Anything, solveID).Return(nil, apperr.ErrSolveNotFound)

	_, err := uc.ResolveSolveShare(context.Background(), solveID, uc.signSolve(solveID))
	require.True(t, errors.Is(err, apperr.ErrShareNotFound))
}
