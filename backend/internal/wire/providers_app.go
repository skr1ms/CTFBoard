package wire

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/TakuyaYagam1/AstroCTFb/config"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/avatar"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/backup"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/competition"
	iws "github.com/TakuyaYagam1/AstroCTFb/internal/websocket"
)

func ProvideServer(router chi.Router, cfg *config.Config) *http.Server {
	return &http.Server{
		Addr:              ":" + cfg.HTTP.Port,
		Handler:           router,
		ReadHeaderTimeout: httpReadHeaderTimeout,
		WriteTimeout:      httpWriteTimeout,
		IdleTimeout:       httpIdleTimeout,
	}
}

func ProvideApp(server *http.Server, userRepo repo.UserRepository, solveUC *competition.SolveUseCase, avatarUC *avatar.AvatarUseCase, backupUC *backup.BackupUseCase, serverDeps *helper.ServerDeps, broadcaster *iws.Broadcaster) *App {
	return &App{
		Server:           server,
		UserRepo:         userRepo,
		SolveUseCase:     solveUC,
		AvatarUC:         avatarUC,
		BackupUC:         backupUC,
		RatelimitAuditWG: serverDeps.Infra.RatelimitAuditWG,
		Broadcaster:      broadcaster,
	}
}
