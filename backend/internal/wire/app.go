package wire

import (
	"net/http"
	"sync"

	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/avatar"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/competition"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/websocket"
)

type App struct {
	Server            *http.Server
	UserRepo          repo.UserRepository
	SubmissionBatcher usecase.SubmissionBatcher
	SolveUseCase      *competition.SolveUseCase
	AvatarUC          *avatar.AvatarUseCase
	RatelimitAuditWG  *sync.WaitGroup
	Broadcaster       *websocket.Broadcaster
}
