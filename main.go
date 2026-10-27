package main

import (
	"cex_project/internal/HttpServer"
	"cex_project/internal/database"
	"cex_project/internal/handlers"
	"cex_project/internal/services"
	"cex_project/internal/utils"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"
)

func main() {
	if err := utils.LoadEnv(); err != nil {
		panic(err)
	}

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

	if err := createAdminUser(ctx, app); err != nil {
		panic(err)
	}

	if err := app.Recover(ctx); err != nil {
		panic(err)
	}

	newHandlers := handlers.NewHandlers(app)

    newServer := HttpServer.NewHttpServer("8000", newHandlers)

	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	go func(){
		fmt.Println("Server started on port: ", newServer.Port())
		if err := newServer.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
            panic(err)
		}
	}()
 
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-signalCtx.Done():
				return
			case <-ticker.C:
				log.Println("Sepolia watcher ticked...")
				
				if err := app.SepoliaWatcher(signalCtx); err != nil && signalCtx.Err() == nil {
					fmt.Println("sepolia watcher error: ", err)
				}
			}
		}
	}()

    <-signalCtx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := newServer.Shutdown(shutdownCtx); err != nil {
		panic(err)
	}
}

func createAdminUser(ctx context.Context, app *services.Services) error {
	username := os.Getenv("ADMIN_USERNAME")
	password := os.Getenv("ADMIN_PASSWORD")
	if username == "" || password == "" {
		return fmt.Errorf("ADMIN_USERNAME and ADMIN_PASSWORD are required")
	}

	user, err := app.DB.Models.User.GetByUsername(ctx, username)
	if err == nil {
		if err := utils.VerifyHash(password, user.PasswordHash); err != nil {
			return errors.New("admin password does not match")
		}
		
		return nil
	}
	
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	passwordHash, err := utils.HashPassword(password)
	if err != nil {
		return err
	}

	return app.DB.Models.User.CreateSuperUser(ctx, username, passwordHash)
}
