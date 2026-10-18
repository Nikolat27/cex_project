package main

import (
	"context"
	"time"
	"cex_project/internal/database"
	"cex_project/internal/services"
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

	if err := app.Withdraw(ctx, 9, "USDT", 1000000); err != nil {
		panic(err)
	}
}
