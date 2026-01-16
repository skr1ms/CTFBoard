package v1

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/skr1ms/CTFBoard/internal/usecase"
)

type solveRoutes struct {
	solveUC *usecase.SolveUseCase
}

func NewSolveRoutes(router chi.Router,
	solveUC *usecase.SolveUseCase,
) {
	routes := solveRoutes{solveUC: solveUC}

	router.Post("/solve", routes.Create)
}

func (h *solveRoutes) Create(w http.ResponseWriter, r *http.Request) {
}
