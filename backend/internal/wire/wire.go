//go:build wireinject

package wire

import (
	"context"

	"github.com/google/wire"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/wahrwelt-kit/go-jwtkit"
	"github.com/wahrwelt-kit/go-logkit"
	"github.com/wahrwelt-kit/go-wskit"

	"github.com/TakuyaYagam1/AstroCTFb/config"
	"github.com/TakuyaYagam1/AstroCTFb/internal/storage"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/mailer"
)

func InitializeApp(
	ctx context.Context,
	cfg *config.Config,
	l logkit.Logger,
	pool *pgxpool.Pool,
	redisClient *redis.Client,
	storageProvider storage.Provider,
	jwtService *jwtkit.JWTService,
	wsHub *wskit.Hub,
	mailer mailer.Mailer,
) (*App, error) {
	wire.Build(RepoSet, UseCaseSet, InfraSet, HTTPSet)
	return nil, nil
}

func InitializeCleanup(pool *pgxpool.Pool, storageProvider storage.Provider) (usecase.Cleaner, error) {
	wire.Build(CleanupSet)
	return nil, nil
}
