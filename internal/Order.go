package internal

import (
	"cex_project/internal/utils"
	"errors"
	"time"
)

type OrderType uint8

const (
	Limit OrderType = iota
	MarketType
)

type Side uint8

const (
	Buy Side = iota
	Sell
)

type Trade struct {
	ID            uint64
	Symbol 		  string
	Price         int64
	Quantity      int64
	BuyOrderID    uint64
	SellOrderID   uint64
	BuyUserID     uint64
	SellUserID    uint64
	BuyLimitPrice int64
	ExecutedAt    time.Time
}

type Status string

const (
	Pending         Status = "pending"
	PartiallyFilled Status = "partially_filled"
	Filled          Status = "filled"
	Cancelled       Status = "cancelled"
)

type Order struct {
	ID       uint64
	UserID   uint64
	Symbol   string
	Side     Side
	Type     OrderType
	Price    int64
	Quantity int64
	Filled   int64
	Status   Status
	CreatedAt time.Time
}

func (o *Order) Remaining() int64 {
	return o.Quantity - o.Filled
}

func (o *Order) IsFilled() bool {
	return o.Remaining() <= 0
}

func (o *Order) UpdateStatus() {
	switch {
	case o.IsFilled():
		o.Status = Filled
	case o.Filled > 0:
		o.Status = PartiallyFilled
	default:
		o.Status = Pending
	}
}

func (o *Order) Cancel() {
	o.Status = Cancelled
}

func (o *Order) ValidateForSubmission() error {
	if o.ID <= 0 {
		return errors.New("order ID must be greater than 0")
	}

	if o.UserID <= 0 {
		return errors.New("user ID must be greater than 0")
	}

	if o.Symbol == "" {
		return errors.New("symbol is required")
	}

	if o.Quantity <= 0 {
		return errors.New("quantity must be greater than 0")
	}

	if o.Price <= 0 {
		return errors.New("price must be greater than 0")
	}

	if o.Type != Limit {
		return errors.New("currently we only support limit orders")
	}

	if o.Filled != 0 {
		return errors.New("filled quantity must be 0")
	}

	if o.Status != Pending {
		return errors.New("order status must be pending")
	}

	if o.Side != Buy && o.Side != Sell {
		return errors.New("side must be buy or sell")
	}

	return nil
}

func sortByHighestPrice(order *Order, orders []*Order) []*Order {
	for i, existing := range orders {
		if order.Price > existing.Price ||
			(order.Price == existing.Price && order.ID < existing.ID) {

			orders = append(orders, nil)
			copy(orders[i+1:], orders[i:])
			orders[i] = order

			return orders
		}
	}

	return append(orders, order)
}

func sortByLowestPrice(order *Order, orders []*Order) []*Order {
	for i, existing := range orders {
		if order.Price < existing.Price ||
			(order.Price == existing.Price && order.ID < existing.ID) {

			orders = append(orders, nil)
			copy(orders[i+1:], orders[i:])
			orders[i] = order

			return orders
		}
	}

	return append(orders, order)
}

func canMatch(taker, maker *Order) bool {
	if maker == nil || taker == nil {
		return false
	}

	if taker.UserID == maker.UserID {
		return false
	}

	if taker.Type == MarketType {
		return true
	}

	if taker.Side == Buy {
		return maker.Price <= taker.Price
	}

	return maker.Price >= taker.Price
}

func executeTrade(taker, maker *Order, symbol string) (Trade, error) {
	quantity := taker.Remaining()

	if maker.Remaining() < quantity {
		quantity = maker.Remaining()
	}

	newTakerFilled, err := utils.SafeAdd(taker.Filled, quantity)
	if err != nil {
		return Trade{}, err
	}

	newMakerFilled, err := utils.SafeAdd(maker.Filled, quantity)
	if err != nil {
		return Trade{}, err
	}

	taker.Filled = newTakerFilled
	maker.Filled = newMakerFilled

	trade := Trade{
		Symbol:   symbol,
		Price:    maker.Price,
		Quantity: quantity,
	}

	if taker.Side == Buy {
		trade.BuyOrderID = taker.ID
		trade.SellOrderID = maker.ID

		trade.BuyUserID = taker.UserID
		trade.SellUserID = maker.UserID

		trade.BuyLimitPrice = taker.Price
	} else {
		trade.BuyOrderID = maker.ID
		trade.SellOrderID = taker.ID

		trade.BuyUserID = maker.UserID
		trade.SellUserID = taker.UserID

		trade.BuyLimitPrice = maker.Price
	}

	return trade, nil
}
