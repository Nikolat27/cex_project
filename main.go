package main

import (
	"context"
	"time"
	"cex_project/internal/database"
	"fmt"
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

	if err := app.Recover(ctx); err != nil {
		panic(err)
	}

	fmt.Println("Recovery completed successfully")
}
