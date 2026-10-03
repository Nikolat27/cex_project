package internal

type OrderType uint8
const (
	Limit OrderType = iota
	Market
)

type Side uint8
const (
	Buy Side = iota
	Sell
)

type Trade struct {
	Price int64
	Quantity int64
	BuyOrderID uint64
	SellOrderID uint64
}

type Order struct {
	ID uint64
	UserID uint64
	Symbol string
	Side Side
	Type OrderType
	Price int64
	Quantity int64
	Filled int64
}

func (o *Order) Remaining() int64 {
	return o.Quantity - o.Filled
}

func (o *Order) IsFilled() bool {
	return o.Remaining() <= 0
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
	if maker == nil {
		return false
	}

	if taker.Type == Market {
		return true
	}

	if taker.Side == Buy {
		return maker.Price <= taker.Price
	}

	return maker.Price >= taker.Price
}

func executeTrade(taker, maker *Order) Trade {
	quantity := taker.Remaining()

	if maker.Remaining() < quantity {
		quantity = maker.Remaining()
	}

	taker.Filled += quantity
	maker.Filled += quantity

	trade := Trade{
		Price:    maker.Price,
		Quantity: quantity,
	}

	if taker.Side == Buy {
		trade.BuyOrderID = taker.ID
		trade.SellOrderID = maker.ID
	} else {
		trade.BuyOrderID = maker.ID
		trade.SellOrderID = taker.ID
	}

	return trade
}
