package services

import (
	"cex_project/internal/core"
	"cex_project/internal/database"
	"sync"
)

type Services struct {
	Exchange *core.ExchangeService
	DB       *database.DB

	Mu sync.RWMutex
}

func NewServices(db *database.DB) (*Services, error) {
	exchange := core.NewExchangeService()

	return &Services{
		Exchange: exchange,
		DB:       db,
	}, nil
}
