package test

import (
	"fmt"
	"sync"
	"testing"

	"cex_project/internal/core"
)

func TestConcurrentSubmissionsToSameBook(t *testing.T) {
	const orderCount = 200

	engine := core.NewEngine()
	book := engine.AddOrderBook("BTCUSDT")

	var wg sync.WaitGroup
	errs := make(chan error, orderCount)

	for i := 1; i <= orderCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			_, err := engine.SubmitOrder(&core.Order{
				ID:       uint64(id),
				UserID:   1,
				Symbol:   "BTCUSDT",
				Side:     core.Buy,
				Type:     core.Limit,
				Price:    100,
				Quantity: 1,
				Status:   core.Pending,
			}, core.BTC, core.USDT)
			if err != nil {
				errs <- err
			}
		}(i)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("unexpected submit error: %v", err)
	}

	snapshot, err := book.Snapshot()
	if err != nil {
		t.Fatalf("unexpected snapshot error: %v", err)
	}
	if len(snapshot.Bids) != 1 {
		t.Fatalf("expected one bid price level, got %d", len(snapshot.Bids))
	}
	if snapshot.Bids[0].Quantity != orderCount {
		t.Fatalf("expected bid quantity %d, got %d", orderCount, snapshot.Bids[0].Quantity)
	}
}

func TestConcurrentSubmissionsToDifferentBooks(t *testing.T) {
	const ordersPerBook = 100

	engine := core.NewEngine()
	btcBook := engine.AddOrderBook("BTCUSDT")
	ethBook := engine.AddOrderBook("ETHUSDT")

	var wg sync.WaitGroup
	errs := make(chan error, ordersPerBook*2)

	for i := 1; i <= ordersPerBook; i++ {
		for _, symbol := range []string{"BTCUSDT", "ETHUSDT"} {
			wg.Add(1)
			go func(id int, symbol string) {
				defer wg.Done()

				_, err := engine.SubmitOrder(&core.Order{
					ID:       uint64(id),
					UserID:   1,
					Symbol:   symbol,
					Side:     core.Buy,
					Type:     core.Limit,
					Price:    100,
					Quantity: 1,
					Status:   core.Pending,
				}, core.BTC, core.USDT)
				if err != nil {
					errs <- fmt.Errorf("%s: %w", symbol, err)
				}
			}(i, symbol)
		}
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("unexpected submit error: %v", err)
	}

	assertSnapshotBidQuantity(t, btcBook, ordersPerBook)
	assertSnapshotBidQuantity(t, ethBook, ordersPerBook)
}

func TestConcurrentSnapshotsAndSubmissions(t *testing.T) {
	const orderCount = 200

	engine := core.NewEngine()
	book := engine.AddOrderBook("BTCUSDT")

	var writers sync.WaitGroup
	errs := make(chan error, orderCount)

	for i := 1; i <= orderCount; i++ {
		writers.Add(1)
		go func(id int) {
			defer writers.Done()

			_, err := engine.SubmitOrder(&core.Order{
				ID:       uint64(id),
				UserID:   1,
				Symbol:   "BTCUSDT",
				Side:     core.Buy,
				Type:     core.Limit,
				Price:    100,
				Quantity: 1,
				Status:   core.Pending,
			}, core.BTC, core.USDT)
			if err != nil {
				errs <- err
			}
		}(i)
	}

	done := make(chan struct{})
	go func() {
		writers.Wait()
		close(done)
	}()

	for {
		select {
		case <-done:
			close(errs)
			for err := range errs {
				t.Errorf("unexpected submit error: %v", err)
			}
			assertSnapshotBidQuantity(t, book, orderCount)
			return
		default:
			_, err := book.Snapshot()
			if err != nil {
				t.Fatalf("unexpected snapshot error: %v", err)
			}
		}
	}
}

func TestConcurrentCancellations(t *testing.T) {
	const orderCount = 200

	engine := core.NewEngine()
	book := engine.AddOrderBook("BTCUSDT")
	orders := make([]*core.Order, 0, orderCount)

	for i := 1; i <= orderCount; i++ {
		order := &core.Order{
			ID:       uint64(i),
			UserID:   1,
			Symbol:   "BTCUSDT",
			Side:     core.Buy,
			Type:     core.Limit,
			Price:    100,
			Quantity: 1,
			Status:   core.Pending,
		}
		if _, err := engine.SubmitOrder(order, core.BTC, core.USDT); err != nil {
			t.Fatalf("failed to arrange order %d: %v", i, err)
		}
		orders = append(orders, order)
	}

	var wg sync.WaitGroup
	errs := make(chan error, orderCount)

	for _, order := range orders {
		wg.Add(1)
		go func(order *core.Order) {
			defer wg.Done()
			if _, err := engine.CancelOrder("BTCUSDT", order.UserID, order.ID); err != nil {
				errs <- err
			}
		}(order)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("unexpected cancellation error: %v", err)
	}

	snapshot, err := book.Snapshot()
	if err != nil {
		t.Fatalf("unexpected snapshot error: %v", err)
	}
	if len(snapshot.Bids) != 0 {
		t.Fatalf("expected no bids after cancellation, got %d levels", len(snapshot.Bids))
	}
	for _, order := range orders {
		if order.Status != core.Cancelled {
			t.Errorf("expected order %d to be cancelled, got %q", order.ID, order.Status)
		}
	}
}

func assertSnapshotBidQuantity(t *testing.T, book *core.OrderBook, expected int64) {
	t.Helper()

	snapshot, err := book.Snapshot()
	if err != nil {
		t.Fatalf("unexpected snapshot error: %v", err)
	}
	if len(snapshot.Bids) != 1 {
		t.Fatalf("expected one bid price level, got %d", len(snapshot.Bids))
	}
	if snapshot.Bids[0].Quantity != expected {
		t.Fatalf("expected bid quantity %d, got %d", expected, snapshot.Bids[0].Quantity)
	}
}
