package internal

import (
	"slices"
	"errors"
	"sync"
)

type OrderBook struct {
	Symbol string

	Bids []*Order // Buyers
	Asks []*Order // Sellers

	Orders map[uint64]*Order

	mu sync.RWMutex
}

type PriceLevel struct {
	Price int64
	Quantity int64
}

type BookSnapshot struct {
	Symbol string
	Bids []PriceLevel
	Asks []PriceLevel
}

func NewOrderBook(symbol string) *OrderBook {
	return &OrderBook{
		Symbol: symbol,
		Bids: []*Order{},
		Asks: []*Order{},

		Orders: make(map[uint64]*Order),
	}
}

func (ob *OrderBook) Snapshot() BookSnapshot {
	snapshot := BookSnapshot{
		Symbol: ob.Symbol,
		Bids: []PriceLevel{},
		Asks: []PriceLevel{},
	}

	ob.mu.RLock()
	defer ob.mu.RUnlock()

	for _, order := range ob.Bids {
		snapshot.Bids = addPriceLevel(snapshot.Bids, order)
	}

	for _, order := range ob.Asks {
		snapshot.Asks = addPriceLevel(snapshot.Asks, order)
	}

	return snapshot
}

func addPriceLevel(levels []PriceLevel, order *Order) []PriceLevel {
	quantity := order.Remaining()
	if quantity <= 0 {
		return levels
	}

	last := len(levels) - 1
	if last >= 0 && levels[last].Price == order.Price {
		levels[last].Quantity += quantity
		return levels
	}

	return append(levels, PriceLevel{
		Price: order.Price,
		Quantity: quantity,
	})
}

func (ob *OrderBook) GetOrder(orderID uint64) *Order {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

    return ob.getOrderUnlocked(orderID)
}

func (ob *OrderBook) getOrderUnlocked(orderID uint64) *Order {
    return ob.Orders[orderID]
}

// Add -> With lock (public method)
func (ob *OrderBook) Add(order *Order) error {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	return ob.addUnlocked(order)
}

// Remove -> With lock (public method)
func (ob *OrderBook) RemoveOrder(orderID uint64) error {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	return ob.removeOrderUnlocked(orderID)
}

func (ob *OrderBook) Match(order *Order) ([]Trade, error) {
	var trades []Trade

    ob.mu.Lock()
	defer ob.mu.Unlock()

	for !order.IsFilled() {
		maker := ob.bestOppositeOrderUnlocked(order)

		if !canMatch(order, maker) {
			break
		}

		trade := executeTrade(order, maker)

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
		if err := ob.addUnlocked(order); err != nil {
			return nil, err
		}
	}

	return trades, nil
}

func (ob *OrderBook) bestOppositeOrderUnlocked(order *Order) *Order {
	if order.Side == Buy {
		return ob.bestAskUnlocked()
	}

	return ob.bestBidUnlocked()
}

func (ob *OrderBook) idExists(orderID uint64) bool {
	_, exists := ob.Orders[orderID]
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

// addUnlocked -> Without lock (Private method)
func (ob *OrderBook) addUnlocked(order *Order) error {
	// Validation
	if order.Price <= 0 {
		return errors.New("price must be greater than 0")
	}

	if order.Quantity <= 0 {
		return errors.New("quantity must be greater than 0")
	}

	if ob.idExists(order.ID) {
		return errors.New("order with this ID already exists")
	}

	order.UpdateStatus()

	if order.Side == Buy {
		ob.Bids = sortByHighestPrice(order, ob.Bids)
	} else {
		ob.Asks = sortByLowestPrice(order, ob.Asks)
	}

	ob.Orders[order.ID] = order

	return nil
}

// removeUnlocked -> Without lock (Private method)
func (ob *OrderBook) removeOrderUnlocked(orderID uint64) error {
	if !ob.idExists(orderID) {
		return errors.New("order with this ID does not exist")
	}

	for i, order := range ob.Bids {
		if order.ID == orderID {
			ob.Bids = slices.Delete(ob.Bids, i, i+1)
			delete(ob.Orders, orderID)
			return nil
		}
	}

	for i, order := range ob.Asks {
		if order.ID == orderID {
			ob.Asks = slices.Delete(ob.Asks, i, i+1)
			delete(ob.Orders, orderID)
			return nil
		}
	}

	return nil
}

func (ob *OrderBook) CancelOrder(userID, orderID uint64) (*Order, error) {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	order := ob.getOrderUnlocked(orderID)
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
