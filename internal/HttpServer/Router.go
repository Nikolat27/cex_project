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

	router.Post("/api/users/create_user", handlers.CreateUser)
	router.Post("/api/users/{id}/deposit", handlers.Deposit)
	router.Post("/api/users/{id}/withdraw", handlers.Withdraw)
	router.Get("/api/users/{id}/balance/{asset}", handlers.GetBalance)

	return &CoreRouter{
		Router: router,
	}
}
