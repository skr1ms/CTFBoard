//go:build wireinject

package wire

import (
	"github.com/google/wire"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/skr1ms/CTFBoard/config"
	"github.com/skr1ms/CTFBoard/internal/storage"
	"github.com/skr1ms/CTFBoard/pkg/jwt"
	"github.com/skr1ms/CTFBoard/pkg/logger"
	"github.com/skr1ms/CTFBoard/pkg/mailer"
	pkgWS "github.com/skr1ms/CTFBoard/pkg/websocket"
)

func InitializeApp(
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
