package test

import (
	"testing"
	"cex_project/internal"
	"reflect"
)

func TestOrderBook_Add(t *testing.T) {
	newOrderBook := &internal.OrderBook{
		Symbol: "BTCUSDT",
		Bids: []*internal.Order{},
		Asks: []*internal.Order{},
	}

	newOrderBook.Add(&internal.Order{
		ID: 1,
		UserID: 1,
		Symbol: "BTCUSDT",
		Side: internal.Buy,
		Type: internal.Limit,
		Price: 10000,
		Quantity: 1,
	})

	expectedBids := []*internal.Order{
		{
			ID: 1,
			UserID: 1,
			Symbol: "BTCUSDT",
			Side: internal.Buy,
			Type: internal.Limit,
			Price: 10000,
			Quantity: 1,
		},
	}

	if !reflect.DeepEqual(newOrderBook.Bids, expectedBids) {
		t.Errorf("Expected bids: %v, got: %v", expectedBids, newOrderBook.Bids)
	}

	expectedAsks := []*internal.Order{}
	if !reflect.DeepEqual(newOrderBook.Asks, expectedAsks) {
		t.Errorf("Expected asks: %v, got: %v", expectedAsks, newOrderBook.Asks)
	}
}

func TestOrderBook_Remove(t *testing.T) {
	newOrderBook := &internal.OrderBook{
		Symbol: "BTCUSDT",
		Bids: []*internal.Order{},
		Asks: []*internal.Order{},
	}

	newOrderBook.Add(&internal.Order{
		ID: 1,
		UserID: 1,
		Symbol: "BTCUSDT",
		Side: internal.Buy,
		Type: internal.Limit,
		Price: 10000,
		Quantity: 1,
	})

	newOrderBook.Add(&internal.Order{
		ID: 2,
		UserID: 1,
		Symbol: "BTCUSDT",
		Side: internal.Sell,
		Type: internal.Limit,
		Price: 10000,
		Quantity: 1,
	})

	expectedBids := []*internal.Order{}

	expectedAsks := []*internal.Order{
		{
			ID: 2,
			UserID: 1,
			Symbol: "BTCUSDT",
			Side: internal.Sell,
			Type: internal.Limit,
			Price: 10000,
			Quantity: 1,
		},
	}
	
	newOrderBook.Remove(1)

	if !reflect.DeepEqual(newOrderBook.Bids, expectedBids) {
		t.Errorf("Expected bids: %v, got: %v", expectedBids, newOrderBook.Bids)
	}

	if !reflect.DeepEqual(newOrderBook.Asks, expectedAsks) {
		t.Errorf("Expected asks: %v, got: %v", expectedAsks, newOrderBook.Asks)
	}
}

func TestOrderBook_BestBid(t *testing.T) {
	newOrderBook := &internal.OrderBook{
		Symbol: "BTCUSDT",
		Bids: []*internal.Order{},
		Asks: []*internal.Order{},
	}

	newOrderBook.Add(&internal.Order{
		ID: 1,
		UserID: 1,
		Symbol: "BTCUSDT",
		Side: internal.Buy,
		Type: internal.Limit,
		Price: 10000,
		Quantity: 1,
	})

	newOrderBook.Add(&internal.Order{
		ID: 2,
		UserID: 1,
		Symbol: "BTCUSDT",
		Side: internal.Buy,
		Type: internal.Limit,
		Price: 10001,
		Quantity: 1,
	})

	newOrderBook.Add(&internal.Order{
		ID: 3,
		UserID: 1,
		Symbol: "BTCUSDT",
		Side: internal.Buy,
		Type: internal.Limit,
		Price: 10002,
		Quantity: 1,
	})

	bestBid := newOrderBook.BestBid()
	if bestBid.ID != 3 {
		t.Errorf("Expected best bid ID: 2, got: %d", bestBid.ID)
	}
	if bestBid.Price != 10002 {
		t.Errorf("Expected best bid price: 10001, got: %d", bestBid.Price)
	}
	if bestBid.Quantity != 1 {
		t.Errorf("Expected best bid quantity: 1, got: %d", bestBid.Quantity)
	}

}

func TestOrderBook_BestAsk(t *testing.T) {
}
