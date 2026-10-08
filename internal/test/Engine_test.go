package test

import (
	"testing"

	"cex_project/internal"
)

func TestEngineOrderBookLifecycle(t *testing.T) {
	engine := internal.NewEngine()

	book := engine.AddOrderBook("BTCUSDT")
	if book == nil {
		t.Fatal("expected AddOrderBook to return a book")
	}
	if book.Symbol != "BTCUSDT" {
		t.Fatalf("expected symbol BTCUSDT, got %q", book.Symbol)
	}
	if engine.GetOrderBook("BTCUSDT") != book {
		t.Fatal("expected GetOrderBook to return the added book")
	}

	engine.DeleteOrderBook("BTCUSDT")
	if engine.GetOrderBook("BTCUSDT") != nil {
		t.Fatal("expected deleted book to be absent")
	}
}

func TestEngineSubmitOrderRejectsNilOrder(t *testing.T) {
	engine := internal.NewEngine()

	if _, err := engine.SubmitOrder(nil); err == nil {
		t.Fatal("expected an error for a nil order")
	}
}

func TestEngineSubmitOrderRejectsUnknownBook(t *testing.T) {
	engine := internal.NewEngine()

	order := &internal.Order{
		ID:       1,
		UserID:   1,
		Symbol:   "BTCUSDT",
		Side:     internal.Buy,
		Type:     internal.Limit,
		Price:    100,
		Quantity: 1,
		Status:   internal.Pending,
	}

	if _, err := engine.SubmitOrder(order); err == nil {
		t.Fatal("expected an error for an unknown order book")
	}
}

func TestEngineSubmitOrderRestsUnmatchedLimitOrder(t *testing.T) {
	engine := internal.NewEngine()
	book := engine.AddOrderBook("BTCUSDT")
	order := &internal.Order{
		ID:       1,
		UserID:   1,
		Symbol:   "BTCUSDT",
		Side:     internal.Buy,
		Type:     internal.Limit,
		Price:    100,
		Quantity: 2,
		Status:   internal.Pending,
	}

	trades, err := engine.SubmitOrder(order)
	if err != nil {
		t.Fatalf("unexpected submit error: %v", err)
	}
	if len(trades) != 0 {
		t.Fatalf("expected no trades, got %d", len(trades))
	}
	if book.BestBid() != order {
		t.Fatal("expected unmatched limit order to rest as the best bid")
	}
	if book.AllOrders[order.ID] != order {
		t.Fatal("expected resting order in the order lookup")
	}
}

func TestEngineSubmitOrderMatchesExistingOrder(t *testing.T) {
	engine := internal.NewEngine()
	book := engine.AddOrderBook("BTCUSDT")
	sell := &internal.Order{
		UserID:   1,
		ID:       1,
		Symbol:   "BTCUSDT",
		Side:     internal.Sell,
		Type:     internal.Limit,
		Price:    100,
		Quantity: 2,
		Status:   internal.Pending,
	}
	buy := &internal.Order{
		UserID:   2,
		ID:       2,
		Symbol:   "BTCUSDT",
		Side:     internal.Buy,
		Type:     internal.Limit,
		Price:    100,
		Quantity: 2,
		Status:   internal.Pending,
	}

	if _, err := engine.SubmitOrder(sell); err != nil {
		t.Fatalf("unexpected maker submit error: %v", err)
	}
	trades, err := engine.SubmitOrder(buy)
	if err != nil {
		t.Fatalf("unexpected taker submit error: %v", err)
	}

	if len(trades) != 1 {
		t.Fatalf("expected one trade, got %d", len(trades))
	}
	trade := trades[0]
	if trade.BuyOrderID != buy.ID || trade.SellOrderID != sell.ID {
		t.Fatalf("unexpected trade order IDs: buy=%d sell=%d", trade.BuyOrderID, trade.SellOrderID)
	}
	if trade.Price != 100 || trade.Quantity != 2 {
		t.Fatalf("unexpected trade: price=%d quantity=%d", trade.Price, trade.Quantity)
	}
	if book.BestAsk() != nil {
		t.Fatal("expected filled ask to be removed")
	}
	if _, exists := book.ActiveOrders[sell.ID]; exists {
		t.Fatal("expected filled maker to be removed from active orders")
	}
	if historical, exists := book.AllOrders[sell.ID]; !exists || historical.Status != internal.Filled {
		t.Fatal("expected filled maker to remain in order history")
	}
	if historical, exists := book.AllOrders[buy.ID]; !exists || historical.Status != internal.Filled {
		t.Fatal("expected filled taker to remain in order history")
	}
}

func TestEngineCancelOrder(t *testing.T) {
	engine := internal.NewEngine()
	book := engine.AddOrderBook("BTCUSDT")
	order := &internal.Order{
		ID:       1,
		UserID:   1,
		Symbol:   "BTCUSDT",
		Side:     internal.Buy,
		Type:     internal.Limit,
		Price:    100,
		Quantity: 1,
		Status:   internal.Pending,
	}

	if _, err := engine.SubmitOrder(order); err != nil {
		t.Fatalf("unexpected submit error: %v", err)
	}
	if _, err := engine.CancelOrder("BTCUSDT", order.UserID, order.ID); err != nil {
		t.Fatalf("unexpected cancel error: %v", err)
	}

	if order.Status != internal.Cancelled {
		t.Fatalf("expected cancelled status, got %q", order.Status)
	}
	if book.BestBid() != nil {
		t.Fatal("expected cancelled bid to be removed")
	}
	if _, exists := book.ActiveOrders[order.ID]; exists {
		t.Fatal("expected cancelled order to be removed from active orders")
	}
	if historical, exists := book.AllOrders[order.ID]; !exists || historical.Status != internal.Cancelled {
		t.Fatal("expected cancelled order to remain in order history")
	}
}

func TestEngineCancelOrderRejectsUnknownBook(t *testing.T) {
	engine := internal.NewEngine()

	if _, err := engine.CancelOrder("BTCUSDT", 1, 1); err == nil {
		t.Fatal("expected an error for an unknown order book")
	}
}

func TestEngineCancelOrderRejectsUnknownOrder(t *testing.T) {
	engine := internal.NewEngine()
	engine.AddOrderBook("BTCUSDT")

	if _, err := engine.CancelOrder("BTCUSDT", 1, 1); err == nil {
		t.Fatal("expected an error for an unknown order")
	}
}
