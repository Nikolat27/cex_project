package internal

import (
	"slices"
)

type OrderBook struct {
	Symbol string

	Bids []*Order // Buyers
	Asks []*Order // Sellers
}

func NewOrderBook(symbol string) *OrderBook {
	return &OrderBook{
		Symbol: symbol,
		Bids: []*Order{},
		Asks: []*Order{},
	}
}

func (ob *OrderBook) Add(order *Order) {
	if order.Side == Buy {
		ob.Bids = sortByHighestPrice(order, ob.Bids)
	} else {
		ob.Asks = sortByLowestPrice(order, ob.Asks)
	}
}

func (ob *OrderBook) Remove(orderID uint64) {
	for i, order := range ob.Bids {
		if order.ID == orderID {
			ob.Bids = slices.Delete(ob.Bids, i, i+1)
			return
		}
	}

	for i, order := range ob.Asks {
		if order.ID == orderID {
			ob.Asks = slices.Delete(ob.Asks, i, i+1)
			return
		}
	}
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
