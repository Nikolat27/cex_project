package main

import (
	"context"
	"time"
	"cex_project/internal/database"
)

func main() {
	db, err := database.New()
	if err != nil {
		panic(err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := db.Ping(ctx); err != nil {
		panic(err)
	}
	
}
