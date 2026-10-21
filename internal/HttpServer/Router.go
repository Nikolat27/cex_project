package HttpServer

import (
	"cex_project/internal/handlers"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type CoreRouter struct {
	Router *chi.Mux
}

func newCoreRouter(handlers *handlers.Handlers) *CoreRouter {
    router := chi.NewRouter()

	router.Use(middleware.Logger)

	// Public routes
	router.Post("/api/users/register", handlers.Register)
	router.Post("/api/users/login", handlers.Login)

	// Everything below requires authentication
	router.Group(func(r chi.Router) {
		r.Use(handlers.RequireAuth)

		r.Delete("/api/users/logout", handlers.Logout)

		r.Post("/api/users/{id}/deposit", handlers.Deposit)
		r.Post("/api/users/{id}/withdraw", handlers.Withdraw)
		r.Get("/api/users/{id}/balance/{asset}", handlers.GetBalance)
		r.Get("/api/users/{id}/orders", handlers.ListUserOrders)
		r.Get("/api/users/{id}/trades", handlers.ListTradesByUser)

		r.Post("/api/markets", handlers.AddMarket)
		r.Get("/api/markets/{symbol}/book", handlers.GetOrderBook)
		r.Get("/api/markets/{symbol}/trades", handlers.ListTradesBySymbol)

		r.Post("/api/orders", handlers.PlaceOrder)
		r.Get("/api/orders/{id}", handlers.GetOrder)
		r.Delete("/api/orders/{id}", handlers.CancelOrder)
	})

	return &CoreRouter{
		Router: router,
	}
}
