package services

import (
	"cex_project/internal/core"
	"cex_project/internal/database"
	"context"
	"errors"
)

func (s *Services) Recover(ctx context.Context) error {
	if err := s.recoverUsers(ctx); err != nil {
		return err
	}

	if err := s.recoverMarkets(ctx); err != nil {
		return err
	}

	if err := s.recoverBalances(ctx); err != nil {
		return err
	}

	if err := s.recoverOrders(ctx); err != nil {
		return err
	}

	if err := s.recoverTrades(ctx); err != nil {
		return err
	}

	if err := s.recoverFeeBalances(ctx); err != nil {
		return err
	}

	return nil
}

func (s *Services) recoverUsers(ctx context.Context) error {
	allUsers, err := s.DB.Models.User.GetAll(ctx)
	if err != nil {
		return err
	}

	for _, user := range allUsers {
		if err := s.recoverUser(&user); err != nil {
			return err
		}
	}

	return nil
}

func (s *Services) recoverMarkets(ctx context.Context) error {
	allMarkets, err := s.DB.Models.Market.GetAll(ctx)
	if err != nil {
		return err
	}

	for _, market := range allMarkets {
		if err := s.recoverMarket(&market); err != nil {
			return err
		}
	}

	return nil
}

func (s *Services) recoverBalances(ctx context.Context) error {
	allBalances, err := s.DB.Models.Balance.GetAll(ctx)
	if err != nil {
		return err
	}

	for _, balance := range allBalances {
		if err := s.recoverBalance(&balance); err != nil {
			return err
		}
	}

	return nil
}

func (s *Services) recoverOrders(ctx context.Context) error {
	allOrders, err := s.DB.Models.Order.GetAll(ctx)
	if err != nil {
		return err
	}

	var maxID int64
	for _, order := range allOrders {
		if order.ID > maxID {
			maxID = order.ID
		}

		if err := s.recoverOrder(&order); err != nil {
			return err
		}
	}

	s.Exchange.SetNextOrderID(uint64(maxID))
	return nil
}

func (s *Services) recoverTrades(ctx context.Context) error {
	allTrades, err := s.DB.Models.Trade.GetAll(ctx)
	if err != nil {
		return err
	}

	var maxID int64
	for _, trade := range allTrades {
		if trade.ID > maxID {
			maxID = trade.ID
		}
		
		if err := s.recoverTrade(&trade); err != nil {
			return err
		}
	}

	s.Exchange.SetNextTradeID(uint64(maxID))
	return nil
}

func (s *Services) recoverFeeBalances(ctx context.Context) error {
	allFeeBalances, err := s.DB.Models.ExchangeFee.GetAll(ctx)
	if err != nil {
		return err
	}

	for _, feeBalance := range allFeeBalances {
		if err := s.recoverFeeBalance(&feeBalance); err != nil {
			return err
		}
	}

	return nil
}

func (s *Services) recoverUser(user *database.User) error {
	if user == nil {
		return errors.New("user is nil")
	}

	if _, err := s.Exchange.CreateUser(uint64(user.ID)); err != nil {
		return err
	}

	return nil
}	

func (s *Services) recoverMarket(market *database.Market) error {
	newMarket := core.Market{
		Symbol: market.Symbol,
		BaseAsset: core.Asset(market.BaseAsset),
		QuoteAsset: core.Asset(market.QuoteAsset),
	}

	if err := s.Exchange.AddMarket(&newMarket); err != nil {
		return err
	}

	return nil
}	

func (s *Services) recoverBalance(balance *database.Balance) error {
	user := s.Exchange.Accounts.GetUser(uint64(balance.UserID))
	if user == nil {
		return errors.New("user not found")
	}

	newBalance := core.Balance{
		Available: balance.Available,
		Locked: balance.Locked,
	}

	if err := user.SetBalance(core.Asset(balance.Asset), &newBalance); err != nil {
		return err
	}

	return nil
}	

func (s *Services) recoverOrder(order *database.Order) error {
	orderBook := s.Exchange.Engine.GetOrderBook(order.Symbol)
	if orderBook == nil {
		return errors.New("order book not found")
	}

	var orderSide core.Side
	switch order.Side {
	case "buy":
		orderSide = core.Buy
	case "sell":
		orderSide = core.Sell
	default:
		return errors.New("unknown order side")
	}

	var orderType core.OrderType
	switch order.Type {
	case "limit":
		orderType = core.Limit
	case "market":
		orderType = core.MarketType
	default:
		return errors.New("unknown order type")
	}

	newOrder := core.Order{
		ID: uint64(order.ID),
		UserID: uint64(order.UserID),
		Symbol: order.Symbol,
		Side: orderSide,
		Type: orderType,
		Price: order.Price,
		Quantity: order.Quantity,
		Filled: order.Filled,
		Status: core.Status(order.Status),
		CreatedAt: order.CreatedAt,
	}

	resting := order.Type == "limit" &&
		(order.Status == "pending" || order.Status == "partially_filled")
	if resting {
		return orderBook.AddOrder(&newOrder)
	}

	return orderBook.AddHistoryOrder(&newOrder)
}	

func (s *Services) recoverTrade(trade *database.Trade) error {
	market, err := s.Exchange.GetMarket(trade.Symbol)
	if err != nil {
		return err
	}

	newTrade := core.Trade{
		ID: uint64(trade.ID),
		Symbol: trade.Symbol,
		BaseAsset: market.BaseAsset,
		QuoteAsset: market.QuoteAsset,
		Price: trade.Price,
		Quantity: trade.Quantity,
		BuyOrderID: uint64(trade.BuyOrderID),
		SellOrderID: uint64(trade.SellOrderID),
		BuyUserID: uint64(trade.BuyUserID),
		SellUserID: uint64(trade.SellUserID),
		BuyLimitPrice: trade.BuyLimitPrice,
		ExecutedAt: trade.ExecutedAt,	
	}

	if err := s.Exchange.AddTrade(&newTrade); err != nil {
		return err
	}

	return nil
}

func (s *Services) recoverFeeBalance(feeBalance *database.ExchangeFee) error {
	if feeBalance == nil {
		return errors.New("fee balance is nil")
	}
	
	if err := s.Exchange.AddFeeBalance(core.Asset(feeBalance.Asset), feeBalance.Available);
	 err != nil {
		return err
	}

	return nil
}

// Use mutex locks when u wanna use this method
func (s *Services) recoverWhenFailed(ctx context.Context) error {
	s.Exchange = core.NewExchangeService()
	return s.Recover(ctx)
}
