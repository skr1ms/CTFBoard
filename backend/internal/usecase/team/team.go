// Package team implements team use cases. Lock order for concurrent safety
// always lock user(s) first in canonical order (by UUID string), then team(s)
// in canonical order (orderTeamLockIDs). Never lock team before user when both are needed
package team

import (
	"context"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-cachekit"
	"github.com/wahrwelt-kit/go-logkit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/cacheutil"
)

type TeamUseCase struct {
	deps TeamDeps
}

type JWTRevoker interface {
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
}

type TeamDeps struct {
	TeamRepo           repo.TeamRepository
	UserRepo           repo.UserRepository
	SolveRepo          repo.SolveRepository
	SubmissionRepo     repo.SubmissionRepository
	AwardRepo          repo.AwardRepository
	CompRepo           repo.CompetitionRepository
	SettingsGetter     usecase.SettingsGetter
	ChallengeRepo      repo.ChallengeRepository
	CompParamUC        usecase.CompetitionParamUseCase
	TM                 repo.TransactionManager
	Guard              usecase.CompetitionGuard
	ScoreboardCache    cacheutil.ScoreboardCacheInvalidator
	ChallengeListCache cacheutil.ChallengeListCacheInvalidator
	UserCache          cacheutil.UserCacheInvalidator
	TeamCache          *cachekit.Cache
	HintRepo           repo.HintRepository
	RatingRepo         repo.RatingRepository
	FieldValueRepo     repo.FieldValueRepository
	JWTRevoker         JWTRevoker
	DefaultMaxTeamSize int
	Logger             logkit.Logger
}

const defaultMaxTeamSize = 10

var _ usecase.TeamUseCase = (*TeamUseCase)(nil)

func NewTeamUseCase(deps TeamDeps) *TeamUseCase {
	if deps.DefaultMaxTeamSize <= 0 {
		deps.DefaultMaxTeamSize = defaultMaxTeamSize
	}

	if deps.Logger == nil {
		deps.Logger = logkit.Noop()
	}

	return &TeamUseCase{deps: deps}
}
