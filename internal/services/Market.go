package services

import (
	"cex_project/internal/core"
	"cex_project/internal/database"
	"context"
)

func (s *Services) AddMarket(ctx context.Context, market *core.Market) error {
	if err := s.Exchange.AddMarket(market); err != nil {
		return err
	}

	if err := s.DB.Models.Market.Create(ctx, &database.Market{
		Symbol: market.Symbol,
		BaseAsset: string(market.BaseAsset),
		QuoteAsset: string(market.QuoteAsset),
	}); err != nil {
		return err
	}

	return nil
}
