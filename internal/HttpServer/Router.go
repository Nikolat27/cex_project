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
	router.Get("/api/users/{id}/orders", handlers.ListUserOrders)
	router.Get("/api/users/{id}/trades", handlers.ListTradesByUser)
	router.Post("/api/markets", handlers.AddMarket)
	router.Get("/api/markets/{symbol}/book", handlers.GetOrderBook)
	router.Get("/api/markets/{symbol}/trades", handlers.ListTradesBySymbol)
	router.Post("/api/orders", handlers.PlaceOrder)
	router.Get("/api/orders/{id}", handlers.GetOrder)
	router.Delete("/api/orders/{id}", handlers.CancelOrder)

	return &CoreRouter{
		Router: router,
	}
}
