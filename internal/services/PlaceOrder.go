package services

import (
	"context"
	"database/sql"
	"errors"

	"cex_project/internal/core"
	"cex_project/internal/database"
	"cex_project/internal/utils"
)

func (s *Services) PlaceLimitOrder(ctx context.Context, order *core.Order) error {
	doesUserExist, err := s.DoesUserExist(ctx, int64(order.UserID))
	if err != nil {
		return err
	}

	if !doesUserExist {
		return errors.New("user does not exist")
	}

	market, err := s.Exchange.GetMarket(order.Symbol)
	if err != nil {
		return err
	}

	asset, amount, err := reservedFunds(order, market)
	if err != nil {
		return err
	}

	balance, err := s.DB.Models.Balance.Get(ctx, int64(order.UserID), asset)
	if errors.Is(err, sql.ErrNoRows) {
		return errors.New("balance does not exist")
	}

	if err != nil {
		return err
	}
	
	if balance.Available < amount {
		return errors.New("insufficient balance")
	}

	if err := s.DB.Models.Balance.Lock(ctx, int64(order.UserID), asset, amount); err != nil {
		return err
	}

	newOrder, trades, err := s.Exchange.PlaceLimitOrder(order.UserID, order.Symbol, order.Side,
		order.Price, order.Quantity)

	if err != nil {
		_ = s.DB.Models.Balance.Unlock(ctx, int64(order.UserID), asset, amount)
		return err
	}

	side := "buy"
	if newOrder.Side == core.Sell {
		side = "sell"
	}

	orderType := "limit"
	if newOrder.Type == core.MarketType {
		orderType = "market"
	}

	_, err = s.DB.Models.Order.Create(ctx, &database.Order{
		ID:        int64(newOrder.ID),
		UserID:    int64(newOrder.UserID),
		Symbol:    newOrder.Symbol,
		Side:      side,
		Type:      orderType,
		Price:     newOrder.Price,
		Quantity:  newOrder.Quantity,
		Filled:    0,
		Status:    "pending",
		CreatedAt: newOrder.CreatedAt,
	})

	if err != nil {
		return err
	}

	for _, trade := range trades {
		settlement, err := trade.Settlement()
		if err != nil {
			return err
		}

		if err := s.DB.SaveTrade(ctx, &trade, settlement); err != nil {
			return err
		}
	}

	return nil
}

func reservedFunds(order *core.Order, market *core.Market) (string, int64, error) {
	if order.Side == core.Buy {
		amount, err := utils.SafeMultiply(order.Price, order.Quantity)
		if err != nil {
			return "", 0, err
		}

		return string(market.QuoteAsset), amount, nil
	}

	return string(market.BaseAsset), order.Quantity, nil
}
