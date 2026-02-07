package challenge

import (
	"github.com/redis/go-redis/v9"
	"github.com/skr1ms/CTFBoard/internal/repo"
	"github.com/skr1ms/CTFBoard/pkg/cache"
	"github.com/skr1ms/CTFBoard/pkg/crypto"
	pkgWS "github.com/skr1ms/CTFBoard/pkg/websocket"
)

type ChallengeUCOption func(*ChallengeUseCase)

func WithTagRepo(r repo.TagRepository) ChallengeUCOption {
	return func(uc *ChallengeUseCase) { uc.tagRepo = r }
}

func WithSolveRepo(r repo.SolveRepository) ChallengeUCOption {
	return func(uc *ChallengeUseCase) { uc.solveRepo = r }
}

func WithTxRepo(r repo.TxRepository) ChallengeUCOption {
	return func(uc *ChallengeUseCase) { uc.txRepo = r }
}

func WithCompetitionRepo(r repo.CompetitionRepository) ChallengeUCOption {
	return func(uc *ChallengeUseCase) { uc.compRepo = r }
}

func WithTeamRepo(r repo.TeamRepository) ChallengeUCOption {
	return func(uc *ChallengeUseCase) { uc.teamRepo = r }
}

func WithRedis(r *redis.Client) ChallengeUCOption {
	return func(uc *ChallengeUseCase) { uc.redis = r }
}

func WithBroadcaster(b pkgWS.SolveBroadcaster) ChallengeUCOption {
	return func(uc *ChallengeUseCase) { uc.broadcaster = b }
}

func WithAuditLogRepo(r repo.AuditLogRepository) ChallengeUCOption {
	return func(uc *ChallengeUseCase) { uc.auditLogRepo = r }
}

func WithCrypto(c crypto.Service) ChallengeUCOption {
	return func(uc *ChallengeUseCase) { uc.crypto = c }
}

func WithScoreboardCache(inv cache.ScoreboardCacheInvalidator) ChallengeUCOption {
	return func(uc *ChallengeUseCase) { uc.scoreboardCache = inv }
}
