package internal

import (
	"cex_project/internal/utils"
	"errors"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

type ExchangeService struct {
	Engine   *Engine
	Accounts *AccountManager
	Markets  map[string]*Market

	trades map[uint64]Trade

	nextOrderID atomic.Uint64
	nextTradeID atomic.Uint64

	mu sync.RWMutex
}

func NewExchangeService() *ExchangeService {
	engine := NewEngine()
	accounts := NewAccountManager()
	markets := make(map[string]*Market)
	trades := make(map[uint64]Trade)

	return &ExchangeService{
		Engine:   engine,
		Accounts: accounts,
		Markets:  markets,
		trades:   trades,
	}
}

func (es *ExchangeService) PlaceLimitOrder(userID uint64, symbol string, side Side,
	price int64, quantity int64) (Order, []Trade, error) {

	newOrder := &Order{
		ID:        es.nextOrderID.Add(1),
		UserID:    userID,
		Symbol:    symbol,
		Side:      side,
		Type:      Limit,
		Price:     price,
		Quantity:  quantity,
		Status:    Pending,
		CreatedAt: time.Now().UTC(),
	}

	trades, err := es.submitOrder(newOrder)
	if err != nil {
		return Order{}, nil, err
	}

	return *newOrder, trades, nil
}

func (es *ExchangeService) PlaceMarketOrder(userID uint64, symbol string, side Side,
	quantity int64, protectionPrice int64) (Order, []Trade, error) {

	newOrder := &Order{
		ID:        es.nextOrderID.Add(1),
		UserID:    userID,
		Symbol:    symbol,
		Side:      side,
		Type:      MarketType,
		Quantity:  quantity,
		Status:    Pending,
		Price:     protectionPrice,
		CreatedAt: time.Now().UTC(),
	}

	trades, err := es.submitOrder(newOrder)
	if err != nil {
		return Order{}, nil, err
	}

	return *newOrder, trades, nil
}

func (es *ExchangeService) AddMarket(market *Market) error {
	es.mu.Lock()
	defer es.mu.Unlock()

	if market == nil {
		return errors.New("market is nil")
	}

	if err := market.Validate(); err != nil {
		return err
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

func (es *ExchangeService) submitOrder(order *Order) ([]Trade, error) {
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

	if err := es.Settletrades(*market, trades); err != nil {
		return nil, err
	}

	if order.Type == MarketType && !order.IsFilled() {
		if err := es.releaseOrderFunds(user, *market, order); err != nil {
			return nil, err
		}

		order.Cancel()
	}

	return trades, nil
}

func (es *ExchangeService) CancelOrder(userID uint64, symbol string, orderID uint64) (*Order, error) {
	user := es.Accounts.GetUser(userID)
	if user == nil {
		return nil, errors.New("user not found")
	}

	market, err := es.GetMarket(symbol)
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

	snapshot, err := book.Snapshot()
	if err != nil {
		return BookSnapshot{}, err
	}

	return snapshot, nil
}

func (es *ExchangeService) reserveOrderFunds(user *User, market Market, order *Order) error {
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

func (es *ExchangeService) Settletrades(market Market, trades []Trade) error {
	for i := range trades {
		trades[i].ID = es.nextTradeID.Add(1)
		trades[i].ExecutedAt = time.Now().UTC()

		if err := es.SettleTrade(market, &trades[i]); err != nil {
			return err
		}

		es.mu.Lock()
		es.trades[trades[i].ID] = trades[i]
		es.mu.Unlock()
	}

	return nil
}

func (es *ExchangeService) SettleTrade(market Market, trade *Trade) error {
	return es.Accounts.settleTrade(market, trade)
}

func (es *ExchangeService) GetTrade(tradeID uint64) (*Trade, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()

	return es.getTradeUnlocked(tradeID)
}

func (es *ExchangeService) getTradeUnlocked(tradeID uint64) (*Trade, error) {
	if tradeID <= 0 {
		return nil, errors.New("trade ID is invalid")
	}

	trade, exists := es.trades[tradeID]
	if !exists {
		return nil, errors.New("trade not found")
	}

	return &trade, nil
}

func (es *ExchangeService) ListTradesBySymbol(symbol string) ([]Trade, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()

	return es.getTradesListBySymbolUnlocked(symbol)
}

func (es *ExchangeService) getTradesListBySymbolUnlocked(symbol string) ([]Trade, error) {
	trades := es.getTradesListUnlocked()
	filteredTrades := []Trade{}

	if symbol == "" {
		return trades, errors.New("symbol is empty")
	}

	for _, trade := range trades {
		if trade.Symbol == symbol {
			filteredTrades = append(filteredTrades, trade)
		}
	}

	return sortByID(filteredTrades), nil
}

func (es *ExchangeService) getTradesListUnlocked() []Trade {
	trades := []Trade{}

	for _, trade := range es.trades {
		trades = append(trades, trade)
	}

	return sortByID(trades)
}

func (es *ExchangeService) ListTradesByUser(userID uint64) ([]Trade, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()

	return es.getTradesListByUserUnlocked(userID)
}

func (es *ExchangeService) getTradesListByUserUnlocked(userID uint64) ([]Trade, error) {
	if userID <= 0 {
		return nil, errors.New("user ID is invalid")
	}

	trades := es.getTradesListUnlocked()
	filteredTrades := []Trade{}

	for _, trade := range trades {
		if trade.BuyUserID == userID || trade.SellUserID == userID {
			filteredTrades = append(filteredTrades, trade)
		}
	}

	return sortByID(filteredTrades), nil
}

func (es *ExchangeService) GetOrder(symbol string, orderID uint64) (Order, error) {
	book := es.Engine.GetOrderBook(symbol)
	if book == nil {
		return Order{}, errors.New("order book not found")
	}

	return book.GetOrder(orderID)
}

func (es *ExchangeService) ListOrdersByUserID(symbol string, userID uint64) ([]Order, error) {
	book := es.Engine.GetOrderBook(symbol)
	if book == nil {
		return nil, errors.New("order book not found")
	}

	return book.ListOrdersByUserID(userID)
}

func sortByID(trades []Trade) []Trade {
	sortedtrades := append([]Trade(nil), trades...)

	sort.Slice(sortedtrades, func(i, j int) bool {
		return sortedtrades[i].ID < sortedtrades[j].ID
	})

	return sortedtrades
}
