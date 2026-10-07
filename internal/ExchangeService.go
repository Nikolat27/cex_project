package internal

import (
	"sync"
	"errors"
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

	if _, exists := es.Markets[market.Symbol]; exists {
		return errors.New("market with this symbol already exists")
	}

	es.Markets[market.Symbol] = market

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

func (es *ExchangeService) GetBalance(userID uint64, asset Asset, amount int64) (Balance, error) {
	user := es.Accounts.GetUser(userID)
	if user == nil {
		return Balance{}, errors.New("user not found")
	}

	return user.GetBalance(asset), nil
}

func (es *ExchangeService) SubmitOrder(order *Order) ([]Trade, error) {
	return es.Engine.SubmitOrder(order)
}

func (es *ExchangeService) CancelOrder(userID uint64, symbol string, orderID uint64) error {
	user := es.Accounts.GetUser(userID)
	if user == nil {
		return errors.New("user not found")
	}

	return es.Engine.CancelOrder(symbol, orderID)
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
		required := order.Price * order.Quantity
		return user.Lock(market.QuoteAsset, required)
	}

	return user.Lock(market.BaseAsset, order.Quantity)
}

func (es *ExchangeService) releaseOrderFunds(user *User, market Market, order *Order) error {
	if order.Type == MarketType {
		return errors.New("market orders are not supported yet")
	}

	remaining := order.Remaining()

	if order.Side == Buy {
		return user.Unlock(market.QuoteAsset, order.Price * remaining)
	}

	return user.Unlock(market.BaseAsset, remaining)
}

func (es *ExchangeService) SettleTrade(market Market, trade *Trade) error {
	return es.Accounts.settleTrade(market, trade)
}
