package internal

import (
	"cex_project/internal/utils"
	"errors"
	"sync"
)

type ExchangeService struct {
	Engine   *Engine
	Accounts *AccountManager
	Markets  map[string]*Market

	mu sync.RWMutex
}

func NewExchangeService() *ExchangeService {
	engine := NewEngine()
	accounts := NewAccountManager()
	markets := make(map[string]*Market)

	return &ExchangeService{
		Engine:   engine,
		Accounts: accounts,
		Markets:  markets,
	}
}

func (es *ExchangeService) AddMarket(market *Market) error {
	es.mu.Lock()
	defer es.mu.Unlock()

	if market == nil {
		return errors.New("market is nil")
	}

	if _, exists := es.Markets[market.Symbol]; exists {
		return errors.New("market with this symbol already exists")
	}

	es.Markets[market.Symbol] = market

	es.Engine.AddOrderBook(market.Symbol)

	return nil
}

func (es *ExchangeService) GetMarket(symbol string) (*Market, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()

	market, exists := es.Markets[symbol]
	if !exists {
		return nil, errors.New("market not found")
	}

	return market, nil
}

func (es *ExchangeService) CreateUser(id uint64) (*User, error) {
	return es.Accounts.CreateUser(id)
}

func (es *ExchangeService) Deposit(userID uint64, asset Asset, amount int64) error {
	user := es.Accounts.GetUser(userID)
	if user == nil {
		return errors.New("user not found")
	}

	return user.Deposit(asset, amount)
}

func (es *ExchangeService) Withdraw(userID uint64, asset Asset, amount int64) error {
	user := es.Accounts.GetUser(userID)
	if user == nil {
		return errors.New("user not found")
	}

	return user.Withdraw(asset, amount)
}

func (es *ExchangeService) GetBalance(userID uint64, asset Asset) (Balance, error) {
	user := es.Accounts.GetUser(userID)
	if user == nil {
		return Balance{}, errors.New("user not found")
	}

	return user.GetBalance(asset), nil
}

func (es *ExchangeService) SubmitOrder(order *Order) ([]Trade, error) {
	if order == nil {
		return nil, errors.New("order is nil")
	}

	market, err := es.GetMarket(order.Symbol)
	if err != nil {
		return nil, err
	}

	user := es.Accounts.GetUser(order.UserID)
	if user == nil {
		return nil, errors.New("user not found")
	}

	if err := es.reserveOrderFunds(user, *market, order); err != nil {
		return nil, err
	}

	trades, err := es.Engine.SubmitOrder(order)
	if err != nil {
		if releaseErr := es.releaseOrderFunds(user, *market, order); releaseErr != nil {
			return nil, releaseErr
		}

		return nil, err
	}

	if err := es.SettleTrades(*market, trades); err != nil {
		return nil, err
	}

	return trades, nil
}

func (es *ExchangeService) CancelOrder(userID uint64, symbol string, orderID uint64) (*Order, error) {
	user := es.Accounts.GetUser(userID)
	if user == nil {
		return nil, errors.New("user not found")
	}

	market, err := es.GetMarket(symbol);
	if err != nil {
		return nil, err
	}

	cancelledOrder, err := es.Engine.CancelOrder(symbol, userID, orderID)
	if err != nil {
		return nil, err
	}

	if err := es.releaseOrderFunds(user, *market, cancelledOrder); err != nil {
		return nil, err
	}

	return cancelledOrder, nil
}

func (es *ExchangeService) GetOrderBookSnapshot(symbol string) (BookSnapshot, error) {
	book := es.Engine.GetOrderBook(symbol)
	if book == nil {
		return BookSnapshot{}, errors.New("order book not found")
	}

	return book.Snapshot(), nil
}

func (es *ExchangeService) reserveOrderFunds(user *User, market Market, order *Order) error {
	if order.Type == MarketType {
		return errors.New("market orders are not supported yet")
	}

	if order.Side == Buy {
		required, err := utils.SafeMultiply(order.Price, order.Quantity)
		if err != nil {
			return err
		}

		return user.Lock(market.QuoteAsset, required)
	}

	return user.Lock(market.BaseAsset, order.Quantity)
}

func (es *ExchangeService) releaseOrderFunds(user *User, market Market, order *Order) error {
	if order.Type == MarketType {
		return errors.New("market orders are not supported yet")
	}

	remaining := order.Remaining()
	if remaining <= 0 {
		return nil
	}

	if order.Side == Buy {
		required, err := utils.SafeMultiply(order.Price, remaining)
		if err != nil {
			return err
		}

		return user.Unlock(market.QuoteAsset, required)
	}

	return user.Unlock(market.BaseAsset, remaining)
}

func (es *ExchangeService) SettleTrades(market Market, trades []Trade) error {
	for _, trade := range trades {
		if err := es.SettleTrade(market, &trade); err != nil {
			return err
		}
	}

	return nil
}

func (es *ExchangeService) SettleTrade(market Market, trade *Trade) error {
	return es.Accounts.settleTrade(market, trade)
}
