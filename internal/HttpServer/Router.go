package HttpServer

import (
	"cex_project/internal/handlers"

	"github.com/go-chi/chi/v5"
)

type CoreRouter struct {
	Router *chi.Mux
}

func newCoreRouter(handlers *handlers.Handlers) *CoreRouter {
    router := chi.NewRouter()

	router.Post("/api/create_user", handlers.CreateUser)

	return &CoreRouter{
		Router: router,
	}
}
