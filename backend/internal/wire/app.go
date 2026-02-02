package wire

import (
	"net/http"

	"github.com/skr1ms/CTFBoard/internal/repo"
)

type App struct {
	Server   *http.Server
	UserRepo repo.UserRepository
}
