package challenge

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	challengeMock "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/challenge/mock"
)

type ratingTestDeps struct {
	challengeRepo *challengeMock.MockChallengeRepository
	solveRepo     *challengeMock.MockSolveRepository
	ratingRepo    *challengeMock.MockRatingRepository
	userRepo      *challengeMock.MockUserRepository
	teamRepo      *challengeMock.MockTeamRepository
	tm            *challengeMock.MockTransactionManager
}

func newRatingTestDeps(t *testing.T) *ratingTestDeps {
	t.Helper()

	return &ratingTestDeps{
		challengeRepo: challengeMock.NewMockChallengeRepository(t),
		solveRepo:     challengeMock.NewMockSolveRepository(t),
		ratingRepo:    challengeMock.NewMockRatingRepository(t),
		userRepo:      challengeMock.NewMockUserRepository(t),
		teamRepo:      challengeMock.NewMockTeamRepository(t),
		tm:            challengeMock.NewMockTransactionManager(t),
	}
}

func (d *ratingTestDeps) newUseCase() *RatingUseCase {
	return NewRatingUseCase(RatingDeps{
		ChallengeRepo: d.challengeRepo,
		SolveRepo:     d.solveRepo,
		RatingRepo:    d.ratingRepo,
		UserRepo:      d.userRepo,
		TeamRepo:      d.teamRepo,
		TM:            d.tm,
	})
}

// executeTx is a helper that makes TM.Run actually call the provided function.
func (d *ratingTestDeps) expectTxRun(returnErr error) {
	d.tm.On("Run", mock.Anything, mock.Anything).
		Return(returnErr).
		Run(func(args mock.Arguments) {
			if returnErr != nil {
				return
			}

			fn, ok := args.Get(1).(func(context.Context) error)
			if !ok {
				return
			}

			_ = fn(args.Get(0).(context.Context))
		}).
		Once()
}

func TestRatingUseCase_PutRating_Success(t *testing.T) {
	t.Parallel()
	d := newRatingTestDeps(t)
	uc := d.newUseCase()

	challengeID := uuid.New()
	userID := uuid.New()
	teamID := uuid.New()

	d.challengeRepo.On("GetByID", mock.Anything, challengeID).
		Return(&domain.Challenge{ID: challengeID, State: domain.ChallengeStateVisible}, nil).Once()
	d.userRepo.On("GetByID", mock.Anything, userID).
		Return(&domain.User{ID: userID, IsBanned: false}, nil).Once()
	d.teamRepo.On("GetByID", mock.Anything, teamID).
		Return(&domain.Team{ID: teamID, IsBanned: false}, nil).Once()

	d.expectTxRun(nil)

	d.solveRepo.On("GetByTeamAndChallengeForUpdate", mock.Anything, teamID, challengeID).
		Return(&domain.Solve{}, nil).Once()
	d.ratingRepo.On("Upsert", mock.Anything, mock.MatchedBy(func(r *domain.Rating) bool {
		return r.ChallengeID == challengeID && r.UserID == userID && r.TeamID == teamID &&
			r.Value == 4 && r.Review == "good challenge"
	})).Return(nil).Once()

	rating, err := uc.PutRating(context.Background(), challengeID, userID, teamID, 4, "good challenge")

	assert.NoError(t, err)
	assert.NotNil(t, rating)
	assert.Equal(t, challengeID, rating.ChallengeID)
	assert.Equal(t, 4, rating.Value)
}

func TestRatingUseCase_PutRating_ChallengeNotFound(t *testing.T) {
	t.Parallel()
	d := newRatingTestDeps(t)
	uc := d.newUseCase()

	d.challengeRepo.On("GetByID", mock.Anything, mock.Anything).
		Return(nil, apperr.ErrChallengeNotFound).Once()

	rating, err := uc.PutRating(context.Background(), uuid.New(), uuid.New(), uuid.New(), 3, "")

	assert.ErrorIs(t, err, apperr.ErrChallengeNotFound)
	assert.Nil(t, rating)
}

func TestRatingUseCase_PutRating_ChallengeRepoUnexpectedError(t *testing.T) {
	t.Parallel()
	d := newRatingTestDeps(t)
	uc := d.newUseCase()

	d.challengeRepo.On("GetByID", mock.Anything, mock.Anything).
		Return(nil, errors.New("db error")).Once()

	rating, err := uc.PutRating(context.Background(), uuid.New(), uuid.New(), uuid.New(), 3, "")

	assert.Error(t, err)
	assert.Nil(t, rating)
}

func TestRatingUseCase_PutRating_HiddenChallenge(t *testing.T) {
	t.Parallel()
	d := newRatingTestDeps(t)
	uc := d.newUseCase()

	d.challengeRepo.On("GetByID", mock.Anything, mock.Anything).
		Return(&domain.Challenge{State: domain.ChallengeStateHidden}, nil).Once()

	rating, err := uc.PutRating(context.Background(), uuid.New(), uuid.New(), uuid.New(), 3, "")

	assert.ErrorIs(t, err, apperr.ErrChallengeNotFound)
	assert.Nil(t, rating)
}

func TestRatingUseCase_PutRating_UserBanned(t *testing.T) {
	t.Parallel()
	d := newRatingTestDeps(t)
	uc := d.newUseCase()

	d.challengeRepo.On("GetByID", mock.Anything, mock.Anything).
		Return(&domain.Challenge{State: domain.ChallengeStateVisible}, nil).Once()
	d.userRepo.On("GetByID", mock.Anything, mock.Anything).
		Return(&domain.User{IsBanned: true}, nil).Once()

	rating, err := uc.PutRating(context.Background(), uuid.New(), uuid.New(), uuid.New(), 3, "")

	assert.ErrorIs(t, err, apperr.ErrUserBanned)
	assert.Nil(t, rating)
}

func TestRatingUseCase_PutRating_UserRepoError(t *testing.T) {
	t.Parallel()
	d := newRatingTestDeps(t)
	uc := d.newUseCase()

	d.challengeRepo.On("GetByID", mock.Anything, mock.Anything).
		Return(&domain.Challenge{State: domain.ChallengeStateVisible}, nil).Once()
	d.userRepo.On("GetByID", mock.Anything, mock.Anything).
		Return(nil, errors.New("user repo error")).Once()

	rating, err := uc.PutRating(context.Background(), uuid.New(), uuid.New(), uuid.New(), 3, "")

	assert.Error(t, err)
	assert.Nil(t, rating)
}

func TestRatingUseCase_PutRating_TeamBanned(t *testing.T) {
	t.Parallel()
	d := newRatingTestDeps(t)
	uc := d.newUseCase()

	d.challengeRepo.On("GetByID", mock.Anything, mock.Anything).
		Return(&domain.Challenge{State: domain.ChallengeStateVisible}, nil).Once()
	d.userRepo.On("GetByID", mock.Anything, mock.Anything).
		Return(&domain.User{IsBanned: false}, nil).Once()
	d.teamRepo.On("GetByID", mock.Anything, mock.Anything).
		Return(&domain.Team{IsBanned: true}, nil).Once()

	rating, err := uc.PutRating(context.Background(), uuid.New(), uuid.New(), uuid.New(), 3, "")

	assert.ErrorIs(t, err, apperr.ErrTeamBanned)
	assert.Nil(t, rating)
}

func TestRatingUseCase_PutRating_TeamRepoError(t *testing.T) {
	t.Parallel()
	d := newRatingTestDeps(t)
	uc := d.newUseCase()

	d.challengeRepo.On("GetByID", mock.Anything, mock.Anything).
		Return(&domain.Challenge{State: domain.ChallengeStateVisible}, nil).Once()
	d.userRepo.On("GetByID", mock.Anything, mock.Anything).
		Return(&domain.User{IsBanned: false}, nil).Once()
	d.teamRepo.On("GetByID", mock.Anything, mock.Anything).
		Return(nil, errors.New("team repo error")).Once()

	rating, err := uc.PutRating(context.Background(), uuid.New(), uuid.New(), uuid.New(), 3, "")

	assert.Error(t, err)
	assert.Nil(t, rating)
}

func TestRatingUseCase_PutRating_SolveRequired(t *testing.T) {
	t.Parallel()
	d := newRatingTestDeps(t)
	uc := d.newUseCase()

	challengeID := uuid.New()
	teamID := uuid.New()

	d.challengeRepo.On("GetByID", mock.Anything, challengeID).
		Return(&domain.Challenge{ID: challengeID, State: domain.ChallengeStateVisible}, nil).Once()
	d.userRepo.On("GetByID", mock.Anything, mock.Anything).
		Return(&domain.User{IsBanned: false}, nil).Once()
	d.teamRepo.On("GetByID", mock.Anything, teamID).
		Return(&domain.Team{IsBanned: false}, nil).Once()

	// TM.Run executes fn; fn returns ErrSolveRequiredForRating; TM.Run propagates it.
	d.tm.On("Run", mock.Anything, mock.Anything).
		Return(func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		}).Once()

	d.solveRepo.On("GetByTeamAndChallengeForUpdate", mock.Anything, teamID, challengeID).
		Return(nil, apperr.ErrSolveNotFound).Once()

	rating, err := uc.PutRating(context.Background(), challengeID, uuid.New(), teamID, 3, "")

	assert.ErrorIs(t, err, apperr.ErrSolveRequiredForRating)
	assert.Nil(t, rating)
}

func TestRatingUseCase_PutRating_TxError(t *testing.T) {
	t.Parallel()
	d := newRatingTestDeps(t)
	uc := d.newUseCase()

	d.challengeRepo.On("GetByID", mock.Anything, mock.Anything).
		Return(&domain.Challenge{State: domain.ChallengeStateVisible}, nil).Once()
	d.userRepo.On("GetByID", mock.Anything, mock.Anything).
		Return(&domain.User{IsBanned: false}, nil).Once()
	d.teamRepo.On("GetByID", mock.Anything, mock.Anything).
		Return(&domain.Team{IsBanned: false}, nil).Once()

	d.tm.On("Run", mock.Anything, mock.Anything).Return(errors.New("tx error")).Once()

	rating, err := uc.PutRating(context.Background(), uuid.New(), uuid.New(), uuid.New(), 3, "")

	assert.Error(t, err)
	assert.Nil(t, rating)
}

func TestRatingUseCase_PutRating_RatingRepoUpsertError(t *testing.T) {
	t.Parallel()
	d := newRatingTestDeps(t)
	uc := d.newUseCase()

	challengeID := uuid.New()
	teamID := uuid.New()

	d.challengeRepo.On("GetByID", mock.Anything, challengeID).
		Return(&domain.Challenge{ID: challengeID, State: domain.ChallengeStateVisible}, nil).Once()
	d.userRepo.On("GetByID", mock.Anything, mock.Anything).
		Return(&domain.User{IsBanned: false}, nil).Once()
	d.teamRepo.On("GetByID", mock.Anything, teamID).
		Return(&domain.Team{IsBanned: false}, nil).Once()

	upsertErr := errors.New("upsert failed")
	d.tm.On("Run", mock.Anything, mock.Anything).
		Return(upsertErr).
		Run(func(args mock.Arguments) {
			fn, ok := args.Get(1).(func(context.Context) error)
			if !ok {
				return
			}

			_ = fn(args.Get(0).(context.Context))
		}).Once()

	d.solveRepo.On("GetByTeamAndChallengeForUpdate", mock.Anything, teamID, challengeID).
		Return(&domain.Solve{}, nil).Once()
	d.ratingRepo.On("Upsert", mock.Anything, mock.Anything).Return(upsertErr).Once()

	rating, err := uc.PutRating(context.Background(), challengeID, uuid.New(), teamID, 3, "")

	assert.Error(t, err)
	assert.Nil(t, rating)
}

func TestRatingUseCase_GetRatingsByChallengeID_Success(t *testing.T) {
	t.Parallel()
	d := newRatingTestDeps(t)
	uc := d.newUseCase()

	challengeID := uuid.New()
	expected := []*domain.Rating{
		{ID: uuid.New(), ChallengeID: challengeID, Value: 4},
		{ID: uuid.New(), ChallengeID: challengeID, Value: 5},
	}

	d.challengeRepo.On("GetByID", mock.Anything, challengeID).
		Return(&domain.Challenge{ID: challengeID, State: domain.ChallengeStateVisible}, nil).Once()
	d.ratingRepo.On("GetByChallengeID", mock.Anything, challengeID).
		Return(expected, nil).Once()

	ratings, err := uc.GetRatingsByChallengeID(context.Background(), challengeID)

	assert.NoError(t, err)
	assert.Len(t, ratings, 2)
	assert.Equal(t, 4, ratings[0].Value)
}

func TestRatingUseCase_GetRatingsByChallengeID_EmptyList(t *testing.T) {
	t.Parallel()
	d := newRatingTestDeps(t)
	uc := d.newUseCase()

	challengeID := uuid.New()

	d.challengeRepo.On("GetByID", mock.Anything, challengeID).
		Return(&domain.Challenge{ID: challengeID, State: domain.ChallengeStateVisible}, nil).Once()
	d.ratingRepo.On("GetByChallengeID", mock.Anything, challengeID).
		Return([]*domain.Rating{}, nil).Once()

	ratings, err := uc.GetRatingsByChallengeID(context.Background(), challengeID)

	assert.NoError(t, err)
	assert.Empty(t, ratings)
}

func TestRatingUseCase_GetRatingsByChallengeID_ChallengeNotFound(t *testing.T) {
	t.Parallel()
	d := newRatingTestDeps(t)
	uc := d.newUseCase()

	d.challengeRepo.On("GetByID", mock.Anything, mock.Anything).
		Return(nil, apperr.ErrChallengeNotFound).Once()

	ratings, err := uc.GetRatingsByChallengeID(context.Background(), uuid.New())

	assert.ErrorIs(t, err, apperr.ErrChallengeNotFound)
	assert.Nil(t, ratings)
}

func TestRatingUseCase_GetRatingsByChallengeID_HiddenChallenge(t *testing.T) {
	t.Parallel()
	d := newRatingTestDeps(t)
	uc := d.newUseCase()

	d.challengeRepo.On("GetByID", mock.Anything, mock.Anything).
		Return(&domain.Challenge{State: domain.ChallengeStateHidden}, nil).Once()

	ratings, err := uc.GetRatingsByChallengeID(context.Background(), uuid.New())

	assert.ErrorIs(t, err, apperr.ErrChallengeNotFound)
	assert.Nil(t, ratings)
}

func TestRatingUseCase_GetRatingsByChallengeID_ChallengeRepoUnexpectedError(t *testing.T) {
	t.Parallel()
	d := newRatingTestDeps(t)
	uc := d.newUseCase()

	d.challengeRepo.On("GetByID", mock.Anything, mock.Anything).
		Return(nil, errors.New("db error")).Once()

	ratings, err := uc.GetRatingsByChallengeID(context.Background(), uuid.New())

	assert.Error(t, err)
	assert.Nil(t, ratings)
}

func TestRatingUseCase_GetRatingsByChallengeID_RatingRepoError(t *testing.T) {
	t.Parallel()
	d := newRatingTestDeps(t)
	uc := d.newUseCase()

	challengeID := uuid.New()

	d.challengeRepo.On("GetByID", mock.Anything, challengeID).
		Return(&domain.Challenge{ID: challengeID, State: domain.ChallengeStateVisible}, nil).Once()
	d.ratingRepo.On("GetByChallengeID", mock.Anything, challengeID).
		Return(nil, errors.New("rating repo error")).Once()

	ratings, err := uc.GetRatingsByChallengeID(context.Background(), challengeID)

	assert.Error(t, err)
	assert.Nil(t, ratings)
}
