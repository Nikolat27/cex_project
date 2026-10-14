package services

import (
	"cex_project/internal/core"
	"cex_project/internal/database"
)

type Services struct {
	Exchange *core.ExchangeService
	DB       *database.DB
}

func NewServices() (*Services, error) {
	exchange := core.NewExchangeService()
	db, err := database.New()

	if err != nil {
		return nil, err
	}

	return &Services{
		Exchange: exchange,
		DB:       db,
	}, nil
}
