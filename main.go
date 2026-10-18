package main

import (
	"context"
	"time"
	"fmt"
	"cex_project/internal/database"
	"cex_project/internal/services"
	"cex_project/internal/HttpServer"
	"cex_project/internal/handlers"
)

func main() {
	db, err := database.New()
	if err != nil {
		panic(err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := db.Ping(ctx); err != nil {
		panic(err)
	}

	app, err := services.NewServices(db)
	if err != nil {
		panic(err)
	}

	if err := app.Recover(ctx); err != nil {
		panic(err)
	}

	newHandlers := handlers.NewHandlers(app)

    newServer := HttpServer.NewHttpServer("8000", newHandlers)
	if err := newServer.Start(); err != nil {
		panic(err)
	}
	defer newServer.Close()

	fmt.Println("Server started on port", newServer.Port())
}
