package internal

import "errors"

type Engine struct {
	Books map[string]*OrderBook
}

func NewEngine() *Engine {
	return &Engine{
		Books: make(map[string]*OrderBook),
	}
}

func (e *Engine) AddOrderBook(symbol string) *OrderBook {
	e.Books[symbol] = NewOrderBook(symbol)	
	return e.Books[symbol]
}

func (e *Engine) GetOrderBook(symbol string) *OrderBook {
	book, exists := e.Books[symbol]
	if !exists {
		return nil
	}

	return book
}

func (e *Engine) DeleteOrderBook(symbol string) {
	delete(e.Books, symbol)
}

func (e *Engine) SubmitOrder(order *Order) ([]Trade, error) {
	if order == nil {
		return nil, errors.New("order is nil")
	}

	book := e.GetOrderBook(order.Symbol)
	if book == nil {
		return nil, errors.New("order book does not exist")
	}

	trades := book.Match(order)
	return trades, nil
}

func (e *Engine) CancelOrder(symbol string, orderID uint64) error {
	book := e.GetOrderBook(symbol)
	if book == nil {
		return errors.New("order book does not exist")
	}

	order, exists := book.Orders[orderID]
	if !exists {
		return errors.New("order does not exist")
	}

	if err := book.Remove(orderID); err != nil {
		return err
	}

	order.Status = Cancelled
	return nil	
}
