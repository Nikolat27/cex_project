package internal

import (
	"slices"
	"errors"
)

type OrderBook struct {
	Symbol string

	Bids []*Order // Buyers
	Asks []*Order // Sellers

	Orders map[uint64]*Order
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

func (ob *OrderBook) Add(order *Order) error {
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

	if order.Side == Buy {
		ob.Bids = sortByHighestPrice(order, ob.Bids)
	} else {
		ob.Asks = sortByLowestPrice(order, ob.Asks)
	}

	ob.Orders[order.ID] = order

	return nil
}

func (ob *OrderBook) Remove(orderID uint64) error {
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

func (ob *OrderBook) BestBid() *Order {
	if len(ob.Bids) == 0 {
		return nil
	}

	return ob.Bids[0]
}

func (ob *OrderBook) BestAsk() *Order {
	if len(ob.Asks) == 0 {
		return nil
	}

	return ob.Asks[0]
}

func (ob *OrderBook) Match(order *Order) []Trade {
	var trades []Trade

	for !order.IsFilled() {
		maker := ob.bestOppositeOrder(order)

		if !canMatch(order, maker) {
			break
		}
		
		trade := executeTrade(order, maker)
		trades = append(trades, trade)

		if maker.IsFilled() {
			ob.Remove(maker.ID)
		}
	}

	if order.Type == Limit && !order.IsFilled() {
		ob.Add(order)
	}

	return trades
}

func (ob *OrderBook) bestOppositeOrder(order *Order) *Order {
	if order.Side == Buy {
		return ob.BestAsk()
	}

	return ob.BestBid()
}

func (ob *OrderBook) idExists(orderID uint64) bool {
	_, exists := ob.Orders[orderID]
	return exists
}
