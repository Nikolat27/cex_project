package services

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"cex_project/internal/core"
	"cex_project/internal/database"
	"cex_project/internal/utils"
)

func (s *Services) PlaceMarketOrder(ctx context.Context, order *core.Order) error {
    s.Mu.Lock()
	defer s.Mu.Unlock()

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
	if err != nil {
		return err
	}

	if balance.Available < amount {
		return errors.New("insufficient balance")
	}

	newOrder, trades, err := s.Exchange.PlaceMarketOrder(order.UserID, order.Symbol, order.Side,
		order.Quantity, order.Price)
	if err != nil {
		return err
	}

	tx, txErr := s.DB.BeginTx(ctx)
	if txErr != nil {
		if err := s.recoverWhenFailed(ctx); err != nil {
			return err
		}

		log.Println("Recovered from failed transaction")
		return txErr
	}

	defer tx.Rollback()

	if lockErr := s.DB.Models.Balance.LockTx(ctx, tx, int64(order.UserID), asset, amount);
	 lockErr != nil {
		if err := s.recoverWhenFailed(ctx); err != nil {
			return err
		}

		log.Println("Recovered from failed transaction")
		return lockErr
	}

	if createOrderErr := s.CreateOrderTx(ctx, tx, &newOrder); createOrderErr != nil {
		if err := s.recoverWhenFailed(ctx); err != nil {
			return err
		}

		log.Println("Recovered from failed transaction")
		return createOrderErr
	}

	if saveTradesErr := s.saveTrades(ctx, tx, trades); saveTradesErr != nil {
		if err := s.recoverWhenFailed(ctx); err != nil {
			return err
		}

		log.Println("Recovered from failed transaction")
		return saveTradesErr
	}

	if newOrder.Remaining() > 0 {
		if cancelErr := s.DB.Models.Order.CancelTx(ctx, tx, int64(newOrder.ID));
		 cancelErr != nil {

			if err := s.recoverWhenFailed(ctx); err != nil {
				return err
			}
			
			log.Println("Recovered from failed transaction")
			return cancelErr
		}

		if releaseOrderFundsErr := s.releaseOrderFunds(ctx, tx, &newOrder);
		 releaseOrderFundsErr != nil {

			if err := s.recoverWhenFailed(ctx); err != nil {
				return err
			}

			log.Println("Recovered from failed transaction")
			return releaseOrderFundsErr
		}
	}

	if commitErr := tx.Commit(); commitErr != nil {
		if err := s.recoverWhenFailed(ctx); err != nil {
			return err
		}

		log.Println("Recovered from failed transaction")
		return commitErr
	}

	order.ID = newOrder.ID

	return nil
}

func (s *Services) PlaceLimitOrder(ctx context.Context, order *core.Order) error {
	s.Mu.Lock()
	defer s.Mu.Unlock()

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
	if err != nil {
		return err
	}

	if balance.Available < amount {
		return errors.New("insufficient balance")
	}

	newOrder, trades, err := s.Exchange.PlaceLimitOrder(order.UserID, order.Symbol, order.Side,
		order.Price, order.Quantity)
	if err != nil {
		return err
	}

	tx, txErr := s.DB.BeginTx(ctx)
	if txErr != nil {
		if err := s.recoverWhenFailed(ctx); err != nil {
			return err
		}

		return txErr
	}

	defer tx.Rollback()

	if lockErr := s.DB.Models.Balance.LockTx(ctx, tx, int64(order.UserID), asset, amount); lockErr != nil {
		if err := s.recoverWhenFailed(ctx); err != nil {
			return err
		}	

		log.Println("Recovered from failed transaction")
		return lockErr
	}

	if createOrderErr := s.CreateOrderTx(ctx, tx, &newOrder); createOrderErr != nil {
		if err := s.recoverWhenFailed(ctx); err != nil {
			return err
		}

		log.Println("Recovered from failed transaction")
		return createOrderErr
	}

	if saveTradesErr := s.saveTrades(ctx, tx, trades); saveTradesErr != nil {
		if err := s.recoverWhenFailed(ctx); err != nil {
			return err
		}

		return saveTradesErr
	}

	if commitErr := tx.Commit(); commitErr != nil {
		if err := s.recoverWhenFailed(ctx); err != nil {
			return err
		}

		return commitErr
	}

	order.ID = newOrder.ID

	return nil
}

func (s *Services) CreateOrderTx(ctx context.Context, tx *sql.Tx, newOrder *core.Order) error {
	// We do this because because the order side and type fields are integers (iota)
	side := "buy"
	if newOrder.Side == core.Sell {
		side = "sell"
	}

	orderType := "limit"
	if newOrder.Type == core.MarketType {
		orderType = "market"
	}

	_, err := s.DB.Models.Order.CreateTx(ctx, tx, &database.Order{
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

	return nil
}

func (s *Services) saveTrades(ctx context.Context, tx *sql.Tx, trades []core.Trade) error {
	for _, trade := range trades {
		settlement, err := trade.Settlement()
		if err != nil {
			return err
		}

		if err := s.DB.SaveTrade(ctx, tx, &trade, settlement); err != nil {
			return err
		}
	}

	return nil
}

// reservedFunds -> Amount of funds that you wanna Lock in the database
func reservedFunds(order *core.Order, market *core.Market) (string, int64, error) {
	if order.Side == core.Buy {
		amount, err := utils.QuoteAmount(order.Price, order.Quantity, string(market.BaseAsset))
		if err != nil {
			return "", 0, err
		}

		return string(market.QuoteAsset), amount, nil
	}

	return string(market.BaseAsset), order.Quantity, nil
}

func (s *Services) CancelOrder(ctx context.Context, order *core.Order) error {
	s.Mu.Lock()
	defer s.Mu.Unlock()

	doesUserExist, err := s.DoesUserExist(ctx, int64(order.UserID))
	if err != nil {
		return err
	}

	if !doesUserExist {
		return errors.New("user does not exist")
	}

	cancelledOrder, err := s.Exchange.CancelOrder(order.UserID, order.Symbol, order.ID)
	if err != nil {
		return err
	}

	tx, err := s.DB.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	if err := s.DB.Models.Order.CancelTx(ctx, tx, int64(cancelledOrder.ID)); err != nil {
		return err
	}

	if err := s.releaseOrderFunds(ctx, tx, cancelledOrder); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Services) releaseOrderFunds(ctx context.Context, tx *sql.Tx, order *core.Order) error {
	market, err := s.Exchange.GetMarket(order.Symbol)
	if err != nil {
		return err
	}

	remaining := order.Remaining()

	if order.Side == core.Buy {
		required, err := utils.QuoteAmount(order.Price, remaining, string(market.BaseAsset))
		if err != nil {
			return err
		}

		if err := s.DB.Models.Balance.UnlockTx(ctx, tx, int64(order.UserID),
			string(market.QuoteAsset), required); err != nil {
			return err
		}
	} else {
		if err := s.DB.Models.Balance.UnlockTx(ctx, tx, int64(order.UserID),
			string(market.BaseAsset), remaining); err != nil {
			return err
		}
	}

	return nil
}
