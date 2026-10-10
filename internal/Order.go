package internal

import (
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

type Status string

const (
	Pending         Status = "pending"
	PartiallyFilled Status = "partially_filled"
	Filled          Status = "filled"
	Cancelled       Status = "cancelled"
)

type Order struct {
	ID        uint64
	UserID    uint64
	Symbol    string
	Side      Side
	Type      OrderType
	Price     int64
	Quantity  int64
	Filled    int64
	Status    Status
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

	if o.Type != Limit && o.Type != MarketType {
		return errors.New("we only support limit & market orders")
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
