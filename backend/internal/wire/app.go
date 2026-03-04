package wire

import (
	"net/http"

	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

type App struct {
	Server            *http.Server
	UserRepo          repo.UserRepository
	SubmissionBatcher usecase.SubmissionBatcher
}
