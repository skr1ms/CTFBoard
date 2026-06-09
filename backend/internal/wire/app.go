package wire

import (
	"net/http"
	"sync"

	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/avatar"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/backup"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/competition"
	"github.com/TakuyaYagam1/AstroCTFb/internal/websocket"
)

type App struct {
	Server           *http.Server
	UserRepo         repo.UserRepository
	SolveUseCase     *competition.SolveUseCase
	AvatarUC         *avatar.AvatarUseCase
	BackupUC         *backup.BackupUseCase
	RatelimitAuditWG *sync.WaitGroup
	Broadcaster      *websocket.Broadcaster
}
