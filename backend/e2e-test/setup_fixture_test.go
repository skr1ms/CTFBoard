package e2e_test

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	restapimiddleware "github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/middleware"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/competition"
)

// teamBracketGetter adapts TeamRepository for scoreboard cache (bracket by team).
type teamBracketGetter struct {
	r repo.TeamRepository
}

func (g *teamBracketGetter) GetTeamBracketID(ctx context.Context, teamID uuid.UUID) (*uuid.UUID, error) {
	team, err := g.r.GetByID(ctx, teamID)
	if err != nil || team == nil {
		return nil, err
	}

	return team.BracketID, nil
}

var (
	TestPool           *pgxpool.Pool
	TestRedis          *redis.Client
	testPort           string
	e2eConnStr         string
	testRateLimitCache *restapimiddleware.RateLimitConfigCache
	testCompetitionUC  *competition.CompetitionUseCase
)

// Mocks.
type noOpMailer struct{}

// Send is a no-op for e2e tests (no real email sent).
func (m *noOpMailer) Send(context.Context, usecase.EmailMessage) error {
	return nil
}
