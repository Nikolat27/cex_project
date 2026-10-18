package handlers

import (
	"cex_project/internal/services"
)

type Handlers struct {
	Services *services.Services
}

func NewHandlers(services *services.Services) *Handlers {
	return &Handlers{
		Services: services,
	}
}
