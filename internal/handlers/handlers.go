package handlers

import (
	"cex_project/internal/core"
)

type Handlers struct {
	Exchange *core.ExchangeService
}

func NewHandlers(exchange *core.ExchangeService) *Handlers {
	return &Handlers{
		Exchange: exchange,
	}
}
