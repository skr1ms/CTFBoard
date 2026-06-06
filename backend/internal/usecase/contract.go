package usecase

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

const (
	DefaultPerPage    = 20
	DefaultMaxPerPage = 100

	DefaultScoreboardHistoryLimit = 10
	MaxScoreboardHistoryLimit     = 100
)

// =============================================================================
// Shared
// =============================================================================

// Paginated is a usecase-level wrapper for paginated list responses.
type Paginated[T any] struct {
	Data       []T
	Total      int64
	Page       int
	PerPage    int
	TotalPages int
}

// TokenPair carries issued access/refresh tokens without exposing JWT
// implementation types through usecase contracts.
type TokenPair struct {
	AccessToken      string
	RefreshToken     string
	AccessExpiresAt  int64
	RefreshExpiresAt int64
}

// SetupRequest carries all data required to complete the first-run setup wizard.
type SetupRequest struct {
	CTFName        string
	CTFDescription string

	Mode        string
	MaxTeamSize int

	ChallengeVisibility       string
	ScoreVisibility           string
	AccountVisibility         string
	RegistrationVisibility    string
	EmailVerificationRequired bool

	AdminUsername string
	AdminEmail    string
	AdminPassword string

	StartTime  *time.Time
	EndTime    *time.Time
	FreezeTime *time.Time
	Timezone   string

	ClientIP string
}

// SetupResult is returned by the first-run setup wizard on success.
type SetupResult struct {
	TokenPair *TokenPair
	User      *domain.User
}

// SetupUseCase orchestrates first-run setup completion.
type SetupUseCase interface {
	IsComplete(ctx context.Context) (bool, error)
	Complete(ctx context.Context, req *SetupRequest) (*SetupResult, error)
}

// =============================================================================
// Helpers
// =============================================================================

// NewPaginated constructs a Paginated response from a data slice, total count, and pagination parameters.
func NewPaginated[T any](data []T, total int64, page, perPage int) *Paginated[T] {
	if data == nil {
		data = make([]T, 0)
	}

	return &Paginated[T]{
		Data:       data,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: TotalPages(total, perPage),
	}
}

// TotalPages calculates the total number of pages for a given total and perPage.
func TotalPages(total int64, perPage int) int {
	if perPage <= 0 {
		return 0
	}

	n64 := (total + int64(perPage) - 1) / int64(perPage)
	if n64 > math.MaxInt {
		return math.MaxInt
	}

	return int(n64)
}

// FetchPage executes a paginated fetch using the provided fetch and count functions, returning a Paginated result.
func FetchPage[T any](
	ctx context.Context,
	page, perPage int,
	fetchFn func(ctx context.Context, limit, offset int) ([]T, error),
	countFn func(ctx context.Context) (int64, error),
) (*Paginated[T], error) {
	if fetchFn == nil || countFn == nil {
		return nil, errors.New("fetchFn and countFn must be non-nil")
	}

	if page < 1 {
		page = 1
	}

	if perPage < 1 {
		perPage = 1
	}

	offset64 := int64(page-1) * int64(perPage)
	if offset64 < 0 || offset64 > math.MaxInt {
		offset64 = math.MaxInt
	}

	items, err := fetchFn(ctx, perPage, int(offset64))
	if err != nil {
		return nil, err
	}

	total, err := countFn(ctx)
	if err != nil {
		return nil, err
	}

	return NewPaginated(items, total, page, perPage), nil
}
