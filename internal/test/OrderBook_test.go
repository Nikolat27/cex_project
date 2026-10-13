package test

import (
	"cex_project/internal/core"
	"reflect"
	"testing"
)

func TestOrderBook_Add(t *testing.T) {
	newOrderBook := internal.NewOrderBook("BTCUSDT")

	newOrderBook.Add(&internal.Order{
		ID:       1,
		UserID:   1,
		Symbol:   "BTCUSDT",
		Side:     internal.Buy,
		Type:     internal.Limit,
		Price:    10000,
		Quantity: 1,
		Status:   internal.Pending,
	})

	expectedBids := []*internal.Order{
		{
			ID:       1,
			UserID:   1,
			Symbol:   "BTCUSDT",
			Side:     internal.Buy,
			Type:     internal.Limit,
			Price:    10000,
			Quantity: 1,
			Status:   internal.Pending,
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
	newOrderBook := internal.NewOrderBook("BTCUSDT")

	newOrderBook.Add(&internal.Order{
		ID:       1,
		UserID:   1,
		Symbol:   "BTCUSDT",
		Side:     internal.Buy,
		Type:     internal.Limit,
		Price:    10000,
		Quantity: 1,
		Status:   internal.Pending,
	})

	newOrderBook.Add(&internal.Order{
		ID:       2,
		UserID:   1,
		Symbol:   "BTCUSDT",
		Side:     internal.Sell,
		Type:     internal.Limit,
		Price:    10000,
		Quantity: 1,
		Status:   internal.Pending,
	})

	expectedBids := []*internal.Order{}

	expectedAsks := []*internal.Order{
		{
			ID:       2,
			UserID:   1,
			Symbol:   "BTCUSDT",
			Side:     internal.Sell,
			Type:     internal.Limit,
			Price:    10000,
			Quantity: 1,
			Status:   internal.Pending,
		},
	}

	newOrderBook.RemoveOrder(1)

	if !reflect.DeepEqual(newOrderBook.Bids, expectedBids) {
		t.Errorf("Expected bids: %v, got: %v", expectedBids, newOrderBook.Bids)
	}

	if !reflect.DeepEqual(newOrderBook.Asks, expectedAsks) {
		t.Errorf("Expected asks: %v, got: %v", expectedAsks, newOrderBook.Asks)
	}
}

func TestOrderBook_BestBid(t *testing.T) {
	newOrderBook := internal.NewOrderBook("BTCUSDT")

	newOrderBook.Add(&internal.Order{
		ID:       1,
		UserID:   1,
		Symbol:   "BTCUSDT",
		Side:     internal.Buy,
		Type:     internal.Limit,
		Price:    10000,
		Quantity: 1,
		Status:   internal.Pending,
	})

	newOrderBook.Add(&internal.Order{
		ID:       2,
		UserID:   1,
		Symbol:   "BTCUSDT",
		Side:     internal.Buy,
		Type:     internal.Limit,
		Price:    10001,
		Quantity: 1,
		Status:   internal.Pending,
	})

	newOrderBook.Add(&internal.Order{
		ID:       3,
		UserID:   1,
		Symbol:   "BTCUSDT",
		Side:     internal.Buy,
		Type:     internal.Limit,
		Price:    10002,
		Quantity: 1,
		Status:   internal.Pending,
	})

	bestBid := newOrderBook.BestBid()
	if bestBid.ID != 3 {
		t.Errorf("Expected best bid ID: 3, got: %d", bestBid.ID)
	}
	if bestBid.Price != 10002 {
		t.Errorf("Expected best bid price: 10002, got: %d", bestBid.Price)
	}
	if bestBid.Quantity != 1 {
		t.Errorf("Expected best bid quantity: 1, got: %d", bestBid.Quantity)
	}
}

func TestOrderBook_BestAsk(t *testing.T) {
	newOrderBook := internal.NewOrderBook("BTCUSDT")

	newOrderBook.Add(&internal.Order{
		ID:       1,
		UserID:   1,
		Symbol:   "BTCUSDT",
		Side:     internal.Sell,
		Type:     internal.Limit,
		Price:    10000,
		Quantity: 1,
		Status:   internal.Pending,
	})

	newOrderBook.Add(&internal.Order{
		ID:       2,
		UserID:   1,
		Symbol:   "BTCUSDT",
		Side:     internal.Sell,
		Type:     internal.Limit,
		Price:    10001,
		Quantity: 1,
		Status:   internal.Pending,
	})

	newOrderBook.Add(&internal.Order{
		ID:       3,
		UserID:   1,
		Symbol:   "BTCUSDT",
		Side:     internal.Sell,
		Type:     internal.Limit,
		Price:    10002,
		Quantity: 1,
		Status:   internal.Pending,
	})

	bestAsk := newOrderBook.BestAsk()
	if bestAsk.ID != 1 {
		t.Errorf("Expected best ask ID: 1, got: %d", bestAsk.ID)
	}
	if bestAsk.Price != 10000 {
		t.Errorf("Expected best ask price: 10000, got: %d", bestAsk.Price)
	}
	if bestAsk.Quantity != 1 {
		t.Errorf("Expected best ask quantity: 1, got: %d", bestAsk.Quantity)
	}
}

func TestOrderBook_Match(t *testing.T) {
	newOrderBook := internal.NewOrderBook("BTCUSDT")

	newOrderBook.Add(&internal.Order{
		ID:       1,
		UserID:   1,
		Symbol:   "BTCUSDT",
		Side:     internal.Buy,
		Type:     internal.Limit,
		Price:    10000,
		Quantity: 1,
		Status:   internal.Pending,
	})

	newOrderBook.Add(&internal.Order{
		ID:       2,
		UserID:   2,
		Symbol:   "BTCUSDT",
		Side:     internal.Sell,
		Type:     internal.Limit,
		Price:    10000,
		Quantity: 1,
		Status:   internal.Pending,
	})

	trades, err := newOrderBook.Match(&internal.Order{
		ID:       3,
		UserID:   3,
		Symbol:   "BTCUSDT",
		Side:     internal.Buy,
		Type:     internal.Limit,
		Price:    10000,
		Quantity: 1,
		Status:   internal.Pending,
	}, internal.BTC, internal.USDT)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(trades) != 1 {
		t.Fatalf("Expected 1 trade, got: %d", len(trades))
	}

	if trades[0].BuyOrderID != 3 {
		t.Fatalf("Expected buy order ID: 3, got: %d", trades[0].BuyOrderID)
	}

	if trades[0].SellOrderID != 2 {
		t.Fatalf("Expected sell order ID: 2, got: %d", trades[0].SellOrderID)
	}

	if trades[0].Price != 10000 {
		t.Fatalf("Expected trade price: 10000, got: %d", trades[0].Price)
	}

	if trades[0].Quantity != 1 {
		t.Fatalf("Expected trade quantity: 1, got: %d", trades[0].Quantity)
	}
}

func TestOrderBook_HistoryKeepsFilledOrders(t *testing.T) {
	book := internal.NewOrderBook("BTCUSDT")
	maker := &internal.Order{
		ID: 1, UserID: 1, Symbol: "BTCUSDT", Side: internal.Sell,
		Type: internal.Limit, Price: 100, Quantity: 1, Status: internal.Pending,
	}
	taker := &internal.Order{
		ID: 2, UserID: 2, Symbol: "BTCUSDT", Side: internal.Buy,
		Type: internal.Limit, Price: 100, Quantity: 1, Status: internal.Pending,
	}

	if err := book.Add(maker); err != nil {
		t.Fatalf("unexpected add error: %v", err)
	}
	if _, err := book.Match(taker, internal.BTC, internal.USDT); err != nil {
		t.Fatalf("unexpected match error: %v", err)
	}

	if len(book.ActiveOrders) != 0 {
		t.Fatalf("expected no active orders, got %d", len(book.ActiveOrders))
	}
	if len(book.AllOrders) != 2 {
		t.Fatalf("expected two historical orders, got %d", len(book.AllOrders))
	}
	if book.AllOrders[maker.ID].Status != internal.Filled ||
		book.AllOrders[taker.ID].Status != internal.Filled {
		t.Fatal("expected both historical orders to be filled")
	}
}

func TestOrderBook_HistoryReturnsSortedCopies(t *testing.T) {
	book := internal.NewOrderBook("BTCUSDT")
	for _, order := range []*internal.Order{
		{ID: 3, UserID: 1, Symbol: "BTCUSDT", Side: internal.Buy, Type: internal.Limit, Price: 90, Quantity: 1, Status: internal.Pending},
		{ID: 1, UserID: 2, Symbol: "BTCUSDT", Side: internal.Buy, Type: internal.Limit, Price: 80, Quantity: 1, Status: internal.Pending},
		{ID: 2, UserID: 1, Symbol: "BTCUSDT", Side: internal.Buy, Type: internal.Limit, Price: 85, Quantity: 1, Status: internal.Pending},
	} {
		if err := book.Add(order); err != nil {
			t.Fatalf("unexpected add error: %v", err)
		}
	}

	orders := book.GetOrders()
	if orders[0].ID != 1 || orders[1].ID != 2 || orders[2].ID != 3 {
		t.Fatalf("expected sorted IDs [1 2 3], got [%d %d %d]", orders[0].ID, orders[1].ID, orders[2].ID)
	}

	order, err := book.GetOrder(1)
	if err != nil {
		t.Fatalf("unexpected get error: %v", err)
	}
	order.Price = 1
	stored, err := book.GetOrder(1)
	if err != nil {
		t.Fatalf("unexpected second get error: %v", err)
	}
	if stored.Price != 80 {
		t.Fatal("expected GetOrder to return a copy")
	}

	userOrders, err := book.ListUserOrders(1)
	if err != nil {
		t.Fatalf("unexpected list error: %v", err)
	}
	if len(userOrders) != 2 || userOrders[0].ID != 2 || userOrders[1].ID != 3 {
		t.Fatalf("expected sorted user IDs [2 3], got %+v", userOrders)
	}
}

func TestOrderBook_HistoryKeepsCancelledOrder(t *testing.T) {
	book := internal.NewOrderBook("BTCUSDT")
	order := &internal.Order{
		ID: 1, UserID: 1, Symbol: "BTCUSDT", Side: internal.Buy,
		Type: internal.Limit, Price: 100, Quantity: 1, Status: internal.Pending,
	}

	if err := book.Add(order); err != nil {
		t.Fatalf("unexpected add error: %v", err)
	}
	if _, err := book.CancelOrder(order.UserID, order.ID); err != nil {
		t.Fatalf("unexpected cancel error: %v", err)
	}

	if _, exists := book.ActiveOrders[order.ID]; exists {
		t.Fatal("expected cancelled order to be inactive")
	}
	historical, err := book.GetOrder(order.ID)
	if err != nil {
		t.Fatalf("expected cancelled order in history: %v", err)
	}
	if historical.Status != internal.Cancelled {
		t.Fatalf("expected cancelled status, got %q", historical.Status)
	}
}
