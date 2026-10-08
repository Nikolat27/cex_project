package test

import (
	"testing"

	"cex_project/internal"
)

func TestExchangeServiceAddMarketCreatesOrderBook(t *testing.T) {
	service := internal.NewExchangeService()
	market := btcUSDTMarket()

	if err := service.AddMarket(&market); err != nil {
		t.Fatalf("unexpected add market error: %v", err)
	}

	stored, err := service.GetMarket(market.Symbol)
	if err != nil {
		t.Fatalf("expected market to exist: %v", err)
	}
	if stored.Symbol != market.Symbol {
		t.Fatalf("expected symbol %q, got %q", market.Symbol, stored.Symbol)
	}
	if service.Engine.GetOrderBook(market.Symbol) == nil {
		t.Fatal("expected adding a market to create its order book")
	}
}

func TestExchangeServiceUnmatchedBuyReservesQuote(t *testing.T) {
	service, market := newExchangeFixture(t)
	buyer := createFundedUser(t, service, 1, internal.USDT, 500)

	_, trades, err := service.PlaceLimitOrder(buyer.ID, market.Symbol, internal.Buy, 100, 2)
	if err != nil {
		t.Fatalf("unexpected submit error: %v", err)
	}
	if len(trades) != 0 {
		t.Fatalf("expected no trades, got %d", len(trades))
	}

	assertBalance(t, buyer, internal.USDT, 300, 200)
	snapshot, err := service.GetOrderBookSnapshot(market.Symbol)
	if err != nil {
		t.Fatalf("unexpected snapshot error: %v", err)
	}
	assertSingleLevel(t, snapshot.Bids, 100, 2)
}

func TestExchangeServiceUnmatchedSellReservesBase(t *testing.T) {
	service, market := newExchangeFixture(t)
	seller := createFundedUser(t, service, 1, internal.BTC, 5)

	_, trades, err := service.PlaceLimitOrder(seller.ID, market.Symbol, internal.Sell, 100, 2)
	if err != nil {
		t.Fatalf("unexpected submit error: %v", err)
	}
	if len(trades) != 0 {
		t.Fatalf("expected no trades, got %d", len(trades))
	}

	assertBalance(t, seller, internal.BTC, 3, 2)
	snapshot, err := service.GetOrderBookSnapshot(market.Symbol)
	if err != nil {
		t.Fatalf("unexpected snapshot error: %v", err)
	}
	assertSingleLevel(t, snapshot.Asks, 100, 2)
}

func TestExchangeServiceRejectsInsufficientFunds(t *testing.T) {
	tests := []struct {
		name   string
		asset  internal.Asset
		amount int64
		side   internal.Side
	}{
		{
			name:   "buy",
			asset:  internal.USDT,
			amount: 199,
			side:   internal.Buy,
		},
		{
			name:   "sell",
			asset:  internal.BTC,
			amount: 1,
			side:   internal.Sell,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service, _ := newExchangeFixture(t)
			user := createFundedUser(t, service, 1, test.asset, test.amount)

			if _, _, err := service.PlaceLimitOrder(1, "BTCUSDT", test.side, 100, 2); err == nil {
				t.Fatal("expected insufficient funds error")
			}

			assertBalance(t, user, test.asset, test.amount, 0)
			snapshot, err := service.GetOrderBookSnapshot("BTCUSDT")
			if err != nil {
				t.Fatalf("unexpected snapshot error: %v", err)
			}
			if len(snapshot.Bids) != 0 || len(snapshot.Asks) != 0 {
				t.Fatal("expected rejected order not to enter the book")
			}
		})
	}
}

func TestExchangeServiceFullTradeSettlesBalances(t *testing.T) {
	service, market := newExchangeFixture(t)
	buyer := createFundedUser(t, service, 1, internal.USDT, 250)
	seller := createFundedUser(t, service, 2, internal.BTC, 2)

	if _, _, err := service.PlaceLimitOrder(seller.ID, market.Symbol, internal.Sell, 90, 2); err != nil {
		t.Fatalf("unexpected sell submit error: %v", err)
	}
	_, trades, err := service.PlaceLimitOrder(buyer.ID, market.Symbol, internal.Buy, 100, 2)
	if err != nil {
		t.Fatalf("unexpected buy submit error: %v", err)
	}

	if len(trades) != 1 {
		t.Fatalf("expected one trade, got %d", len(trades))
	}
	if trades[0].Price != 90 || trades[0].Quantity != 2 {
		t.Fatalf("unexpected trade: price=%d quantity=%d", trades[0].Price, trades[0].Quantity)
	}

	assertBalance(t, buyer, internal.USDT, 70, 0)
	assertBalance(t, buyer, internal.BTC, 2, 0)
	assertBalance(t, seller, internal.BTC, 0, 0)
	assertBalance(t, seller, internal.USDT, 180, 0)

	snapshot, err := service.GetOrderBookSnapshot(market.Symbol)
	if err != nil {
		t.Fatalf("unexpected snapshot error: %v", err)
	}
	if len(snapshot.Bids) != 0 || len(snapshot.Asks) != 0 {
		t.Fatal("expected fully matched book to be empty")
	}
}

func TestExchangeServicePartialFillLeavesRemainingFundsLocked(t *testing.T) {
	service, market := newExchangeFixture(t)
	buyer := createFundedUser(t, service, 1, internal.USDT, 300)
	seller := createFundedUser(t, service, 2, internal.BTC, 2)

	if _, _, err := service.PlaceLimitOrder(seller.ID, market.Symbol, internal.Sell, 90, 2); err != nil {
		t.Fatalf("unexpected sell submit error: %v", err)
	}
	buy, trades, err := service.PlaceLimitOrder(buyer.ID, market.Symbol, internal.Buy, 100, 3)
	if err != nil {
		t.Fatalf("unexpected buy submit error: %v", err)
	}

	if len(trades) != 1 || buy.Filled != 2 {
		t.Fatalf("expected one trade filling 2, got trades=%d filled=%d", len(trades), buy.Filled)
	}
	assertBalance(t, buyer, internal.USDT, 20, 100)
	assertBalance(t, buyer, internal.BTC, 2, 0)

	snapshot, err := service.GetOrderBookSnapshot(market.Symbol)
	if err != nil {
		t.Fatalf("unexpected snapshot error: %v", err)
	}
	assertSingleLevel(t, snapshot.Bids, 100, 1)
}

func TestExchangeServiceCancellationUnlocksRemainingFunds(t *testing.T) {
	service, market := newExchangeFixture(t)
	buyer := createFundedUser(t, service, 1, internal.USDT, 200)

	order, _, err := service.PlaceLimitOrder(buyer.ID, market.Symbol, internal.Buy, 100, 1)
	if err != nil {
		t.Fatalf("unexpected submit error: %v", err)
	}
	assertBalance(t, buyer, internal.USDT, 100, 100)

	cancelled, err := service.CancelOrder(buyer.ID, market.Symbol, order.ID)
	if err != nil {
		t.Fatalf("unexpected cancel error: %v", err)
	}
	if cancelled.Status != internal.Cancelled {
		t.Fatalf("expected cancelled status, got %q", cancelled.Status)
	}
	assertBalance(t, buyer, internal.USDT, 200, 0)
}

func TestExchangeServiceRejectsCancellationByAnotherUser(t *testing.T) {
	service, market := newExchangeFixture(t)
	owner := createFundedUser(t, service, 1, internal.USDT, 100)
	other, err := service.CreateUser(2)
	if err != nil {
		t.Fatalf("failed to create other user: %v", err)
	}

	order, _, err := service.PlaceLimitOrder(owner.ID, market.Symbol, internal.Buy, 100, 1)
	if err != nil {
		t.Fatalf("unexpected submit error: %v", err)
	}
	if _, err := service.CancelOrder(other.ID, market.Symbol, order.ID); err == nil {
		t.Fatal("expected ownership error")
	}

	assertBalance(t, owner, internal.USDT, 0, 100)
	snapshot, err := service.GetOrderBookSnapshot(market.Symbol)
	if err != nil {
		t.Fatalf("unexpected snapshot error: %v", err)
	}
	assertSingleLevel(t, snapshot.Bids, 100, 1)
}

func TestExchangeServiceFailedSubmissionReleasesReservation(t *testing.T) {
	service, market := newExchangeFixture(t)
	buyer := createFundedUser(t, service, 1, internal.USDT, 300)

	// Keep the market registered while removing its engine book so submission
	// fails after reservation and exercises the refund path.
	service.Engine.DeleteOrderBook(market.Symbol)
	if _, _, err := service.PlaceLimitOrder(buyer.ID, market.Symbol, internal.Buy, 100, 1); err == nil {
		t.Fatal("expected missing order book error")
	}

	assertBalance(t, buyer, internal.USDT, 300, 0)
}

func TestExchangeServicePlaceLimitOrderGeneratesIDs(t *testing.T) {
	service, market := newExchangeFixture(t)
	buyer := createFundedUser(t, service, 1, internal.USDT, 500)

	first, _, err := service.PlaceLimitOrder(buyer.ID, market.Symbol, internal.Buy, 100, 1)
	if err != nil {
		t.Fatalf("unexpected first order error: %v", err)
	}
	second, _, err := service.PlaceLimitOrder(buyer.ID, market.Symbol, internal.Buy, 90, 1)
	if err != nil {
		t.Fatalf("unexpected second order error: %v", err)
	}

	if first.ID == 0 || second.ID != first.ID+1 {
		t.Fatalf("expected sequential generated IDs, got %d and %d", first.ID, second.ID)
	}
	if first.Status != internal.Pending || second.Status != internal.Pending {
		t.Fatalf("expected pending orders, got %q and %q", first.Status, second.Status)
	}
}

func newExchangeFixture(t *testing.T) (*internal.ExchangeService, internal.Market) {
	t.Helper()

	service := internal.NewExchangeService()
	market := btcUSDTMarket()
	if err := service.AddMarket(&market); err != nil {
		t.Fatalf("failed to add market: %v", err)
	}

	return service, market
}

func btcUSDTMarket() internal.Market {
	return internal.Market{
		Symbol:     "BTCUSDT",
		BaseAsset:  internal.BTC,
		QuoteAsset: internal.USDT,
	}
}

func createFundedUser(
	t *testing.T,
	service *internal.ExchangeService,
	id uint64,
	asset internal.Asset,
	amount int64,
) *internal.User {
	t.Helper()

	user, err := service.CreateUser(id)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	if err := service.Deposit(id, asset, amount); err != nil {
		t.Fatalf("failed to fund user: %v", err)
	}

	return user
}

func assertSingleLevel(t *testing.T, levels []internal.PriceLevel, price, quantity int64) {
	t.Helper()

	if len(levels) != 1 {
		t.Fatalf("expected one price level, got %d", len(levels))
	}
	if levels[0].Price != price || levels[0].Quantity != quantity {
		t.Fatalf(
			"expected level price=%d quantity=%d, got price=%d quantity=%d",
			price,
			quantity,
			levels[0].Price,
			levels[0].Quantity,
		)
	}
}
