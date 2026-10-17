package services

import (
	"cex_project/internal/core"
	"cex_project/internal/database"
)

type Services struct {
	Exchange *core.ExchangeService
	DB       *database.DB
}

func NewServices(db *database.DB) (*Services, error) {
	exchange := core.NewExchangeService()

	return &Services{
		Exchange: exchange,
		DB:       db,
	}, nil
}
