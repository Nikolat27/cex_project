package services

import (
	"context"

	"cex_project/internal/core"
	"cex_project/internal/database"
	"cex_project/internal/utils"
)

func (s *Services) AddMarket(ctx context.Context, market *core.Market) error {
	s.Mu.Lock()
	defer s.Mu.Unlock()

	if err := s.Exchange.AddMarket(market); err != nil {
		return err
	}

	if err := s.DB.Models.Market.Create(ctx, &database.Market{
		Symbol: market.Symbol,
		BaseAsset: string(market.BaseAsset),
		QuoteAsset: string(market.QuoteAsset),
	}); err != nil {
		return utils.DatabaseError(err)
	}

	return nil
}
