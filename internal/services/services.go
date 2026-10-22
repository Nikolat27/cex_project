package services

import (
	"cex_project/internal/core"
	"cex_project/internal/database"
	"cex_project/internal/paseto"
	"sync"
)

type Services struct {
	Exchange *core.ExchangeService
	DB       *database.DB
	Paseto   *paseto.Maker

	Mu sync.RWMutex
}

func NewServices(db *database.DB) (*Services, error) {
	exchange := core.NewExchangeService()

	paseto, err := paseto.New()
	if err != nil {
		return nil, err
	}

	return &Services{
		Exchange: exchange,
		DB:       db,
		Paseto:   paseto,
	}, nil
}
