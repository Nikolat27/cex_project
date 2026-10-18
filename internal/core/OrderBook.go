package core

import (
	"cex_project/internal/utils"
	"errors"
	"slices"
	"sort"
	"sync"
)

type OrderBook struct {
	Symbol string

	Bids []*Order // Buyers
	Asks []*Order // Sellers

	ActiveOrders map[uint64]*Order
	AllOrders    map[uint64]*Order

	mu sync.RWMutex
}

type PriceLevel struct {
	Price    int64
	Quantity int64
}

type BookSnapshot struct {
	Symbol string
	Bids   []PriceLevel
	Asks   []PriceLevel
}

func NewOrderBook(symbol string) *OrderBook {
	return &OrderBook{
		Symbol: symbol,
		Bids:   []*Order{},
		Asks:   []*Order{},

		ActiveOrders: make(map[uint64]*Order),
		AllOrders:    make(map[uint64]*Order),
	}
}

func (ob *OrderBook) Snapshot() (BookSnapshot, error) {
	snapshot := BookSnapshot{
		Symbol: ob.Symbol,
		Bids:   []PriceLevel{},
		Asks:   []PriceLevel{},
	}

	ob.mu.RLock()
	defer ob.mu.RUnlock()

	for _, order := range ob.Bids {
		var err error
		snapshot.Bids, err = AddOrderPriceLevel(snapshot.Bids, order)
		if err != nil {
			return BookSnapshot{}, err
		}
	}

	for _, order := range ob.Asks {
		var err error
		snapshot.Asks, err = AddOrderPriceLevel(snapshot.Asks, order)
		if err != nil {
			return BookSnapshot{}, err
		}
	}

	return snapshot, nil
}

func AddOrderPriceLevel(levels []PriceLevel, order *Order) ([]PriceLevel, error) {
	quantity := order.Remaining()
	if quantity <= 0 {
		return levels, nil
	}

	last := len(levels) - 1
	if last >= 0 && levels[last].Price == order.Price {

		var err error
		levels[last].Quantity, err = utils.SafeAdd(levels[last].Quantity, quantity)
		if err != nil {
			return nil, err
		}

		return levels, nil
	}

	return append(levels, PriceLevel{
		Price:    order.Price,
		Quantity: quantity,
	}), nil
}

func (ob *OrderBook) ListUserOrders(userID uint64) ([]Order, error) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	return ob.getUserOrdersListUnlocked(userID)
}

func (ob *OrderBook) getUserOrdersListUnlocked(userID uint64) ([]Order, error) {
	if userID <= 0 {
		return nil, errors.New("user ID is invalid")
	}

	userOrders := []Order{}
	for _, order := range ob.getOrdersUnlocked() {
		if order.UserID == userID {
			userOrders = append(userOrders, order)
		}
	}

	return userOrders, nil
}

func (ob *OrderBook) GetOrders() []Order {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	return ob.getOrdersUnlocked()
}

func (ob *OrderBook) ListOrdersByUserID(userID uint64) ([]Order, error) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	return ob.getOrdersByUserIDUnlocked(userID)
}

func (ob *OrderBook) getOrdersByUserIDUnlocked(userID uint64) ([]Order, error) {
	if userID <= 0 {
		return nil, errors.New("user ID is invalid")
	}

	orders := []Order{}
	for _, order := range ob.getOrdersUnlocked() {
		if order.UserID == userID {
			orders = append(orders, order)
		}
	}

	return orders, nil
}

func (ob *OrderBook) getOrdersUnlocked() []Order {
	orders := make([]Order, 0, len(ob.AllOrders))
	for _, order := range ob.AllOrders {
		orders = append(orders, *order)
	}

	sort.Slice(orders, func(i, j int) bool {
		return orders[i].ID < orders[j].ID
	})

	return orders
}

func (ob *OrderBook) GetOrder(orderID uint64) (Order, error) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	if orderID == 0 {
		return Order{}, errors.New("order ID is invalid")
	}

	order, exists := ob.AllOrders[orderID]
	if !exists {
		return Order{}, errors.New("order not found")
	}

	return *order, nil
}

func (ob *OrderBook) getActiveOrderUnlocked(orderID uint64) *Order {
	return ob.ActiveOrders[orderID]
}

// AddOrder -> With lock (public method)
func (ob *OrderBook) AddOrder(order *Order) error {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	if err := ob.registerOrderUnlocked(order); err != nil {
		return err
	}

	if err := ob.AddOrderActiveOrderUnlocked(order); err != nil {
		delete(ob.AllOrders, order.ID)
		return err
	}

	return nil
}

// Remove -> With lock (public method)
func (ob *OrderBook) RemoveOrder(orderID uint64) error {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	return ob.removeOrderUnlocked(orderID)
}

// Match -> Create the trades
func (ob *OrderBook) Match(order *Order, baseAsset, quoteAsset Asset) ([]Trade, error) {
	var trades []Trade

	ob.mu.Lock()
	defer ob.mu.Unlock()

	if ob.wouldPreventSelfTrade(order) {
		return nil, errors.New("self-trade prevented")
	}

	if err := ob.registerOrderUnlocked(order); err != nil {
		return nil, err
	}

	for !order.IsFilled() {
		maker := ob.bestOppositeOrderUnlocked(order)

		if !canMatch(order, maker) {
			break
		}

		trade, err := executeTrade(order, maker, ob.Symbol, baseAsset, quoteAsset)
		if err != nil {
			return nil, err
		}

		order.UpdateStatus()
		maker.UpdateStatus()

		trades = append(trades, trade)

		if maker.IsFilled() {
			if err := ob.removeOrderUnlocked(maker.ID); err != nil {
				return nil, err
			}
		}
	}

	if order.Type == Limit && !order.IsFilled() {
		if err := ob.AddOrderActiveOrderUnlocked(order); err != nil {
			return nil, err
		}
	}

	return trades, nil
}

func (ob *OrderBook) wouldPreventSelfTrade(taker *Order) bool {
	makers := ob.Bids
	if taker.Side == Buy {
		makers = ob.Asks
	}

	for _, maker := range makers {
		if !canMatch(taker, maker) {
			break
		}

		if maker.UserID == taker.UserID {
			return true
		}
	}

	return false
}

func (ob *OrderBook) bestOppositeOrderUnlocked(order *Order) *Order {
	if order.Side == Buy {
		return ob.bestAskUnlocked()
	}

	return ob.bestBidUnlocked()
}

func (ob *OrderBook) activeOrderExists(orderID uint64) bool {
	_, exists := ob.ActiveOrders[orderID]
	return exists
}

func (ob *OrderBook) BestBid() *Order {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	return ob.bestBidUnlocked()
}

func (ob *OrderBook) bestBidUnlocked() *Order {
	if len(ob.Bids) == 0 {
		return nil
	}

	return ob.Bids[0]
}

func (ob *OrderBook) BestAsk() *Order {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	return ob.bestAskUnlocked()
}

func (ob *OrderBook) bestAskUnlocked() *Order {
	if len(ob.Asks) == 0 {
		return nil
	}

	return ob.Asks[0]
}

func (ob *OrderBook) registerOrderUnlocked(order *Order) error {
	if order == nil {
		return errors.New("order is nil")
	}

	if order.Price <= 0 {
		return errors.New("price must be greater than 0")
	}

	if order.Quantity <= 0 {
		return errors.New("quantity must be greater than 0")
	}

	if _, exists := ob.AllOrders[order.ID]; exists {
		return errors.New("order with this ID already exists")
	}

	ob.AllOrders[order.ID] = order
	return nil
}

func (ob *OrderBook) AddOrderActiveOrderUnlocked(order *Order) error {
	if ob.activeOrderExists(order.ID) {
		return errors.New("order with this ID is already active")
	}

	order.UpdateStatus()

	if order.Side == Buy {
		ob.Bids = sortByHighestPrice(order, ob.Bids)
	} else {
		ob.Asks = sortByLowestPrice(order, ob.Asks)
	}

	ob.ActiveOrders[order.ID] = order

	return nil
}

// removeUnlocked -> Without lock (Private method)
func (ob *OrderBook) removeOrderUnlocked(orderID uint64) error {
	if !ob.activeOrderExists(orderID) {
		return errors.New("order with this ID does not exist")
	}

	delete(ob.ActiveOrders, orderID)

	for i, order := range ob.Bids {
		if order.ID == orderID {
			ob.Bids = slices.Delete(ob.Bids, i, i+1)
			return nil
		}
	}

	for i, order := range ob.Asks {
		if order.ID == orderID {
			ob.Asks = slices.Delete(ob.Asks, i, i+1)
			return nil
		}
	}

	return nil
}

func (ob *OrderBook) CancelOrder(userID, orderID uint64) (*Order, error) {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	order := ob.getActiveOrderUnlocked(orderID)
	if order == nil {
		return nil, errors.New("order with this ID does not exist")
	}

	if order.UserID != userID {
		return nil, errors.New("order does not belong to this user")
	}

	if err := ob.removeOrderUnlocked(orderID); err != nil {
		return nil, err
	}

	order.Cancel()

	return order, nil
}
