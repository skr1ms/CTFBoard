//go:build wireinject

package wire

import (
	"context"

	"github.com/google/wire"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/TakuyaYagam1/AstroCTFb/config"
	"github.com/TakuyaYagam1/AstroCTFb/internal/storage"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/jwt"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/logger"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/mailer"
	pkgWS "github.com/TakuyaYagam1/AstroCTFb/pkg/websocket"
)

func InitializeApp(
	ctx context.Context,
	cfg *config.Config,
	l logger.Logger,
	pool *pgxpool.Pool,
	redisClient *redis.Client,
	storageProvider storage.Provider,
	jwtService *jwt.JWTService,
	wsHub *pkgWS.Hub,
	mailer mailer.Mailer,
) (*App, error) {
	wire.Build(RepoSet, UseCaseSet, InfraSet, HTTPSet)
	return nil, nil
}
