package internal

import (
	"errors"
	"sync"
)

type Engine struct {
	Books map[string]*OrderBook
	mu sync.RWMutex
}

func NewEngine() *Engine {
	return &Engine{
		Books: make(map[string]*OrderBook),
	}
}

func (e *Engine) AddOrderBook(symbol string) *OrderBook {
	e.mu.Lock()
	defer e.mu.Unlock()

	return e.addOrderBookUnlocked(symbol)
}

func (e *Engine) addOrderBookUnlocked(symbol string) *OrderBook{
	e.Books[symbol] = NewOrderBook(symbol)	
	return e.Books[symbol]
}

func (e *Engine) GetOrderBook(symbol string) *OrderBook {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.getOrderBookUnlocked(symbol)
}

func (e *Engine) getOrderBookUnlocked(symbol string) *OrderBook {
	book, exists := e.Books[symbol]
	if !exists {
		return nil
	}

	return book
}

func (e *Engine) DeleteOrderBook(symbol string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.deleteOrderBookUnlocked(symbol)
}

func (e *Engine) deleteOrderBookUnlocked(symbol string) {
	delete(e.Books, symbol)
}

func (e *Engine) SubmitOrder(order *Order) ([]Trade, error) {
	if order == nil {
		return nil, errors.New("order is nil")
	}

	if err := order.ValidateForSubmission(); err != nil {
		return nil, err
	}

	book := e.GetOrderBook(order.Symbol)
	if book == nil {
		return nil, errors.New("order book does not exist")
	}

	trades, err :=  book.Match(order)
	if err != nil {
		return nil, err
	}
	
	return trades, nil
}

func (e *Engine) CancelOrder(symbol string, userID, orderID uint64) (*Order, error) {
	book := e.GetOrderBook(symbol)
	if book == nil {
		return nil, errors.New("order book does not exist")
	}

	order, err := book.CancelOrder(userID, orderID)
	if err != nil {
		return nil, err
	}

	return order, nil
}
