package test

import (
	"fmt"
	"sync"
	"testing"

	"cex_project/internal/core"
)

func TestExchangeServiceAddMarketCreatesOrderBook(t *testing.T) {
	service := core.NewExchangeService()
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
	buyer := createFundedUser(t, service, 1, core.USDT, 500)

	_, trades, err := service.PlaceLimitOrder(buyer.ID, market.Symbol, core.Buy, 100*wholeCoin, 2)
	if err != nil {
		t.Fatalf("unexpected submit error: %v", err)
	}
	if len(trades) != 0 {
		t.Fatalf("expected no trades, got %d", len(trades))
	}

	assertBalance(t, buyer, core.USDT, 300, 200)
	snapshot, err := service.GetOrderBookSnapshot(market.Symbol)
	if err != nil {
		t.Fatalf("unexpected snapshot error: %v", err)
	}
	assertSingleLevel(t, snapshot.Bids, 100*wholeCoin, 2)
}

func TestExchangeServiceUnmatchedSellReservesBase(t *testing.T) {
	service, market := newExchangeFixture(t)
	seller := createFundedUser(t, service, 1, core.BTC, 5)

	_, trades, err := service.PlaceLimitOrder(seller.ID, market.Symbol, core.Sell, 100*wholeCoin, 2)
	if err != nil {
		t.Fatalf("unexpected submit error: %v", err)
	}
	if len(trades) != 0 {
		t.Fatalf("expected no trades, got %d", len(trades))
	}

	assertBalance(t, seller, core.BTC, 3, 2)
	snapshot, err := service.GetOrderBookSnapshot(market.Symbol)
	if err != nil {
		t.Fatalf("unexpected snapshot error: %v", err)
	}
	assertSingleLevel(t, snapshot.Asks, 100*wholeCoin, 2)
}

func TestExchangeServiceRejectsInsufficientFunds(t *testing.T) {
	tests := []struct {
		name   string
		asset  core.Asset
		amount int64
		side   core.Side
	}{
		{
			name:   "buy",
			asset:  core.USDT,
			amount: 199,
			side:   core.Buy,
		},
		{
			name:   "sell",
			asset:  core.BTC,
			amount: 1,
			side:   core.Sell,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service, _ := newExchangeFixture(t)
			user := createFundedUser(t, service, 1, test.asset, test.amount)

			if _, _, err := service.PlaceLimitOrder(1, "BTCUSDT", test.side, 100*wholeCoin, 2); err == nil {
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
	buyer := createFundedUser(t, service, 1, core.USDT, 250)
	seller := createFundedUser(t, service, 2, core.BTC, 2)

	if _, _, err := service.PlaceLimitOrder(seller.ID, market.Symbol, core.Sell, 90*wholeCoin, 2); err != nil {
		t.Fatalf("unexpected sell submit error: %v", err)
	}
	_, trades, err := service.PlaceLimitOrder(buyer.ID, market.Symbol, core.Buy, 100*wholeCoin, 2)
	if err != nil {
		t.Fatalf("unexpected buy submit error: %v", err)
	}

	if len(trades) != 1 {
		t.Fatalf("expected one trade, got %d", len(trades))
	}
	if trades[0].Price != 90*wholeCoin || trades[0].Quantity != 2 {
		t.Fatalf("unexpected trade: price=%d quantity=%d", trades[0].Price, trades[0].Quantity)
	}
	if trades[0].ID == 0 {
		t.Fatal("expected generated trade ID")
	}
	if trades[0].ExecutedAt.IsZero() {
		t.Fatal("expected trade execution timestamp")
	}
	if trades[0].ExecutedAt.Location().String() != "UTC" {
		t.Fatalf("expected UTC execution timestamp, got %s", trades[0].ExecutedAt.Location())
	}

	assertBalance(t, buyer, core.USDT, 70, 0)
	assertBalance(t, buyer, core.BTC, 2, 0)
	assertBalance(t, seller, core.BTC, 0, 0)
	assertBalance(t, seller, core.USDT, 180, 0)

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
	buyer := createFundedUser(t, service, 1, core.USDT, 300)
	seller := createFundedUser(t, service, 2, core.BTC, 2)

	if _, _, err := service.PlaceLimitOrder(seller.ID, market.Symbol, core.Sell, 90*wholeCoin, 2); err != nil {
		t.Fatalf("unexpected sell submit error: %v", err)
	}
	buy, trades, err := service.PlaceLimitOrder(buyer.ID, market.Symbol, core.Buy, 100*wholeCoin, 3)
	if err != nil {
		t.Fatalf("unexpected buy submit error: %v", err)
	}

	if len(trades) != 1 || buy.Filled != 2 {
		t.Fatalf("expected one trade filling 2, got trades=%d filled=%d", len(trades), buy.Filled)
	}
	assertBalance(t, buyer, core.USDT, 20, 100)
	assertBalance(t, buyer, core.BTC, 2, 0)

	snapshot, err := service.GetOrderBookSnapshot(market.Symbol)
	if err != nil {
		t.Fatalf("unexpected snapshot error: %v", err)
	}
	assertSingleLevel(t, snapshot.Bids, 100*wholeCoin, 1)
}

func TestExchangeServiceCancellationUnlocksRemainingFunds(t *testing.T) {
	service, market := newExchangeFixture(t)
	buyer := createFundedUser(t, service, 1, core.USDT, 200)

	order, _, err := service.PlaceLimitOrder(buyer.ID, market.Symbol, core.Buy, 100*wholeCoin, 1)
	if err != nil {
		t.Fatalf("unexpected submit error: %v", err)
	}
	assertBalance(t, buyer, core.USDT, 100, 100)

	cancelled, err := service.CancelOrder(buyer.ID, market.Symbol, order.ID)
	if err != nil {
		t.Fatalf("unexpected cancel error: %v", err)
	}
	if cancelled.Status != core.Cancelled {
		t.Fatalf("expected cancelled status, got %q", cancelled.Status)
	}
	assertBalance(t, buyer, core.USDT, 200, 0)
}

func TestExchangeServiceRejectsCancellationByAnotherUser(t *testing.T) {
	service, market := newExchangeFixture(t)
	owner := createFundedUser(t, service, 1, core.USDT, 100)
	other, err := service.CreateUser(2)
	if err != nil {
		t.Fatalf("failed to create other user: %v", err)
	}

	order, _, err := service.PlaceLimitOrder(owner.ID, market.Symbol, core.Buy, 100*wholeCoin, 1)
	if err != nil {
		t.Fatalf("unexpected submit error: %v", err)
	}
	if _, err := service.CancelOrder(other.ID, market.Symbol, order.ID); err == nil {
		t.Fatal("expected ownership error")
	}

	assertBalance(t, owner, core.USDT, 0, 100)
	snapshot, err := service.GetOrderBookSnapshot(market.Symbol)
	if err != nil {
		t.Fatalf("unexpected snapshot error: %v", err)
	}
	assertSingleLevel(t, snapshot.Bids, 100*wholeCoin, 1)
}

func TestExchangeServiceFailedSubmissionReleasesReservation(t *testing.T) {
	service, market := newExchangeFixture(t)
	buyer := createFundedUser(t, service, 1, core.USDT, 300)

	// Keep the market registered while removing its engine book so submission
	// fails after reservation and exercises the refund path.
	service.Engine.DeleteOrderBook(market.Symbol)
	if _, _, err := service.PlaceLimitOrder(buyer.ID, market.Symbol, core.Buy, 100*wholeCoin, 1); err == nil {
		t.Fatal("expected missing order book error")
	}

	assertBalance(t, buyer, core.USDT, 300, 0)
}

func TestExchangeServicePlaceLimitOrderGeneratesIDs(t *testing.T) {
	service, market := newExchangeFixture(t)
	buyer := createFundedUser(t, service, 1, core.USDT, 500)

	first, _, err := service.PlaceLimitOrder(buyer.ID, market.Symbol, core.Buy, 100*wholeCoin, 1)
	if err != nil {
		t.Fatalf("unexpected first order error: %v", err)
	}
	second, _, err := service.PlaceLimitOrder(buyer.ID, market.Symbol, core.Buy, 90*wholeCoin, 1)
	if err != nil {
		t.Fatalf("unexpected second order error: %v", err)
	}

	if first.ID == 0 || second.ID != first.ID+1 {
		t.Fatalf("expected sequential generated IDs, got %d and %d", first.ID, second.ID)
	}
	if first.Status != core.Pending || second.Status != core.Pending {
		t.Fatalf("expected pending orders, got %q and %q", first.Status, second.Status)
	}
}

func TestExchangeServiceConcurrentOrderIDsAreUnique(t *testing.T) {
	const orderCount = 100

	service, market := newExchangeFixture(t)
	buyer := createFundedUser(t, service, 1, core.USDT, orderCount*100)

	var wg sync.WaitGroup
	ids := make(chan uint64, orderCount)
	errs := make(chan error, orderCount)

	for i := 0; i < orderCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			order, _, err := service.PlaceLimitOrder(
				buyer.ID,
				market.Symbol,
				core.Buy,
				100*wholeCoin,
				1,
			)
			if err != nil {
				errs <- err
				return
			}
			ids <- order.ID
		}()
	}

	wg.Wait()
	close(ids)
	close(errs)

	for err := range errs {
		t.Errorf("unexpected order placement error: %v", err)
	}

	seen := make(map[uint64]struct{}, orderCount)
	for id := range ids {
		if id == 0 {
			t.Error("expected nonzero order ID")
		}
		if _, exists := seen[id]; exists {
			t.Errorf("duplicate order ID %d", id)
		}
		seen[id] = struct{}{}
	}
	if len(seen) != orderCount {
		t.Fatalf("expected %d unique order IDs, got %d", orderCount, len(seen))
	}
}

func TestExchangeServiceConcurrentTradeIDsAreUnique(t *testing.T) {
	const tradeCount = 25

	type pendingTrade struct {
		buyerID uint64
		symbol  string
	}

	service := core.NewExchangeService()
	pending := make([]pendingTrade, 0, tradeCount)

	for i := 0; i < tradeCount; i++ {
		base := core.BTC
		symbol := fmt.Sprintf("BTCUSDT%d", i)
		market := core.Market{
			Symbol:     symbol,
			BaseAsset:  base,
			QuoteAsset: core.USDT,
		}
		if err := service.AddMarket(&market); err != nil {
			t.Fatalf("failed to add market %s: %v", symbol, err)
		}

		buyerID := uint64(i*2 + 1)
		sellerID := uint64(i*2 + 2)
		createFundedUser(t, service, buyerID, core.USDT, 100)
		seller := createFundedUser(t, service, sellerID, base, 1)

		if _, _, err := service.PlaceLimitOrder(
			seller.ID,
			symbol,
			core.Sell,
			90*wholeCoin,
			1,
		); err != nil {
			t.Fatalf("failed to place maker for %s: %v", symbol, err)
		}

		pending = append(pending, pendingTrade{
			buyerID: buyerID,
			symbol:  symbol,
		})
	}

	var wg sync.WaitGroup
	tradeIDs := make(chan uint64, tradeCount)
	errs := make(chan error, tradeCount)

	for _, item := range pending {
		wg.Add(1)
		go func(item pendingTrade) {
			defer wg.Done()

			_, trades, err := service.PlaceLimitOrder(
				item.buyerID,
				item.symbol,
				core.Buy,
				100*wholeCoin,
				1,
			)
			if err != nil {
				errs <- err
				return
			}
			if len(trades) != 1 {
				errs <- fmt.Errorf("%s: expected one trade, got %d", item.symbol, len(trades))
				return
			}
			if trades[0].ExecutedAt.IsZero() {
				errs <- fmt.Errorf("%s: execution timestamp is zero", item.symbol)
				return
			}
			tradeIDs <- trades[0].ID
		}(item)
	}

	wg.Wait()
	close(tradeIDs)
	close(errs)

	for err := range errs {
		t.Errorf("unexpected concurrent trade error: %v", err)
	}

	seen := make(map[uint64]struct{}, tradeCount)
	for id := range tradeIDs {
		if id == 0 {
			t.Error("expected nonzero trade ID")
		}
		if _, exists := seen[id]; exists {
			t.Errorf("duplicate trade ID %d", id)
		}
		seen[id] = struct{}{}
	}
	if len(seen) != tradeCount {
		t.Fatalf("expected %d unique trade IDs, got %d", tradeCount, len(seen))
	}
}

func TestExchangeServiceListTradesByUser(t *testing.T) {
	service, market := newExchangeFixture(t)
	seller := createFundedUser(t, service, 1, core.BTC, 2)
	buyerOne := createFundedUser(t, service, 2, core.USDT, 100)
	buyerTwo := createFundedUser(t, service, 3, core.USDT, 100)

	if _, _, err := service.PlaceLimitOrder(
		seller.ID, market.Symbol, core.Sell, 100*wholeCoin, 2,
	); err != nil {
		t.Fatalf("unexpected maker submit error: %v", err)
	}
	if _, _, err := service.PlaceLimitOrder(
		buyerOne.ID, market.Symbol, core.Buy, 100*wholeCoin, 1,
	); err != nil {
		t.Fatalf("unexpected first buyer submit error: %v", err)
	}
	if _, _, err := service.PlaceLimitOrder(
		buyerTwo.ID, market.Symbol, core.Buy, 100*wholeCoin, 1,
	); err != nil {
		t.Fatalf("unexpected second buyer submit error: %v", err)
	}

	sellerTrades, err := service.ListTradesByUser(seller.ID)
	if err != nil {
		t.Fatalf("unexpected seller history error: %v", err)
	}
	if len(sellerTrades) != 2 {
		t.Fatalf("expected two seller trades, got %d", len(sellerTrades))
	}
	if sellerTrades[0].ID >= sellerTrades[1].ID {
		t.Fatalf(
			"expected ascending trade IDs, got [%d %d]",
			sellerTrades[0].ID,
			sellerTrades[1].ID,
		)
	}
	for _, trade := range sellerTrades {
		if trade.SellUserID != seller.ID {
			t.Fatalf("expected seller ID %d, got %d", seller.ID, trade.SellUserID)
		}
	}

	buyerTrades, err := service.ListTradesByUser(buyerOne.ID)
	if err != nil {
		t.Fatalf("unexpected buyer history error: %v", err)
	}
	if len(buyerTrades) != 1 || buyerTrades[0].BuyUserID != buyerOne.ID {
		t.Fatalf("unexpected buyer history: %+v", buyerTrades)
	}

	unrelatedTrades, err := service.ListTradesByUser(4)
	if err != nil {
		t.Fatalf("unexpected unrelated-user history error: %v", err)
	}
	if len(unrelatedTrades) != 0 {
		t.Fatalf("expected empty unrelated-user history, got %d", len(unrelatedTrades))
	}
}

func TestExchangeServiceListTradesByUserRejectsInvalidID(t *testing.T) {
	service := core.NewExchangeService()

	if _, err := service.ListTradesByUser(0); err == nil {
		t.Fatal("expected an error for an invalid user ID")
	}
}

func TestExchangeServiceMarketBuyFullyFills(t *testing.T) {
	service, market := newExchangeFixture(t)
	seller := createFundedUser(t, service, 1, core.BTC, 1)
	buyer := createFundedUser(t, service, 2, core.USDT, 110)

	if _, _, err := service.PlaceLimitOrder(
		seller.ID, market.Symbol, core.Sell, 100*wholeCoin, 1,
	); err != nil {
		t.Fatalf("unexpected maker submit error: %v", err)
	}

	order, trades, err := service.PlaceMarketOrder(
		buyer.ID, market.Symbol, core.Buy, 1, 110*wholeCoin,
	)
	if err != nil {
		t.Fatalf("unexpected market order error: %v", err)
	}

	if order.Status != core.Filled {
		t.Fatalf("expected filled status, got %q", order.Status)
	}
	if len(trades) != 1 || trades[0].Price != 100*wholeCoin || trades[0].Quantity != 1 {
		t.Fatalf("unexpected trades: %+v", trades)
	}
	assertBalance(t, buyer, core.USDT, 10, 0)
	assertBalance(t, buyer, core.BTC, 1, 0)
	assertBalance(t, seller, core.BTC, 0, 0)
	assertBalance(t, seller, core.USDT, 100, 0)
}

func TestExchangeServiceMarketBuyPartiallyFillsAndUnlocksRemainder(t *testing.T) {
	service, market := newExchangeFixture(t)
	seller := createFundedUser(t, service, 1, core.BTC, 1)
	buyer := createFundedUser(t, service, 2, core.USDT, 220)

	if _, _, err := service.PlaceLimitOrder(
		seller.ID, market.Symbol, core.Sell, 100*wholeCoin, 1,
	); err != nil {
		t.Fatalf("unexpected maker submit error: %v", err)
	}

	order, trades, err := service.PlaceMarketOrder(
		buyer.ID, market.Symbol, core.Buy, 2, 110*wholeCoin,
	)
	if err != nil {
		t.Fatalf("unexpected market order error: %v", err)
	}

	if order.Status != core.Cancelled || order.Filled != 1 {
		t.Fatalf("expected partially filled and cancelled order, got %+v", order)
	}
	if len(trades) != 1 || trades[0].Quantity != 1 {
		t.Fatalf("unexpected trades: %+v", trades)
	}
	assertBalance(t, buyer, core.USDT, 120, 0)
	assertBalance(t, buyer, core.BTC, 1, 0)

	book := service.Engine.GetOrderBook(market.Symbol)
	if _, exists := book.ActiveOrders[order.ID]; exists {
		t.Fatal("expected market order not to rest in active orders")
	}
}

func TestExchangeServiceMarketBuyWithNoLiquidityUnlocksAllFunds(t *testing.T) {
	service, market := newExchangeFixture(t)
	buyer := createFundedUser(t, service, 1, core.USDT, 220)

	order, trades, err := service.PlaceMarketOrder(
		buyer.ID, market.Symbol, core.Buy, 2, 110*wholeCoin,
	)
	if err != nil {
		t.Fatalf("unexpected market order error: %v", err)
	}

	if order.Status != core.Cancelled || order.Filled != 0 {
		t.Fatalf("expected unfilled cancelled order, got %+v", order)
	}
	if len(trades) != 0 {
		t.Fatalf("expected no trades, got %d", len(trades))
	}
	assertBalance(t, buyer, core.USDT, 220, 0)
}

func TestExchangeServiceMarketBuyRespectsProtectionPrice(t *testing.T) {
	service, market := newExchangeFixture(t)
	seller := createFundedUser(t, service, 1, core.BTC, 1)
	buyer := createFundedUser(t, service, 2, core.USDT, 110)

	sellOrder, _, err := service.PlaceLimitOrder(
		seller.ID, market.Symbol, core.Sell, 120*wholeCoin, 1,
	)
	if err != nil {
		t.Fatalf("unexpected maker submit error: %v", err)
	}

	order, trades, err := service.PlaceMarketOrder(
		buyer.ID, market.Symbol, core.Buy, 1, 110*wholeCoin,
	)
	if err != nil {
		t.Fatalf("unexpected market order error: %v", err)
	}

	if order.Status != core.Cancelled || len(trades) != 0 {
		t.Fatalf("expected protected market order not to trade, order=%+v trades=%+v", order, trades)
	}
	assertBalance(t, buyer, core.USDT, 110, 0)

	book := service.Engine.GetOrderBook(market.Symbol)
	if book.BestAsk() == nil || book.BestAsk().ID != sellOrder.ID {
		t.Fatal("expected ask above the protection price to remain active")
	}
}

func TestExchangeServiceMarketSellFullyFills(t *testing.T) {
	service, market := newExchangeFixture(t)
	buyer := createFundedUser(t, service, 1, core.USDT, 100)
	seller := createFundedUser(t, service, 2, core.BTC, 1)

	if _, _, err := service.PlaceLimitOrder(
		buyer.ID, market.Symbol, core.Buy, 100*wholeCoin, 1,
	); err != nil {
		t.Fatalf("unexpected maker submit error: %v", err)
	}

	order, trades, err := service.PlaceMarketOrder(
		seller.ID, market.Symbol, core.Sell, 1, 90*wholeCoin,
	)
	if err != nil {
		t.Fatalf("unexpected market order error: %v", err)
	}

	if order.Status != core.Filled {
		t.Fatalf("expected filled status, got %q", order.Status)
	}
	if len(trades) != 1 || trades[0].Price != 100*wholeCoin {
		t.Fatalf("unexpected trades: %+v", trades)
	}
	assertBalance(t, seller, core.BTC, 0, 0)
	assertBalance(t, seller, core.USDT, 100, 0)
}

func TestExchangeServicePreventsSelfTradeAndReleasesTakerFunds(t *testing.T) {
	service, market := newExchangeFixture(t)
	user := createFundedUser(t, service, 1, core.BTC, 1)
	if err := service.Deposit(user.ID, core.USDT, 100); err != nil {
		t.Fatalf("failed to fund quote balance: %v", err)
	}

	maker, _, err := service.PlaceLimitOrder(
		user.ID, market.Symbol, core.Sell, 100*wholeCoin, 1,
	)
	if err != nil {
		t.Fatalf("unexpected maker submit error: %v", err)
	}

	if _, _, err := service.PlaceLimitOrder(
		user.ID, market.Symbol, core.Buy, 100*wholeCoin, 1,
	); err == nil {
		t.Fatal("expected self-trade prevention error")
	}

	book := service.Engine.GetOrderBook(market.Symbol)
	if book.BestAsk() == nil || book.BestAsk().ID != maker.ID {
		t.Fatal("expected maker to remain active")
	}
	storedMaker, err := service.GetOrder(market.Symbol, maker.ID)
	if err != nil {
		t.Fatalf("expected maker in order history: %v", err)
	}
	if storedMaker.Status != core.Pending || storedMaker.Filled != 0 {
		t.Fatalf("expected unchanged maker, got %+v", storedMaker)
	}
	if len(book.AllOrders) != 1 {
		t.Fatalf("expected only the maker in order history, got %d orders", len(book.AllOrders))
	}

	assertBalance(t, user, core.BTC, 0, 1)
	assertBalance(t, user, core.USDT, 100, 0)

	trades, err := service.ListTradesByUser(user.ID)
	if err != nil {
		t.Fatalf("unexpected trade history error: %v", err)
	}
	if len(trades) != 0 {
		t.Fatalf("expected no self-trades, got %d", len(trades))
	}
}

func TestExchangeServiceAllowsNonCrossingOrdersFromSameUser(t *testing.T) {
	service, market := newExchangeFixture(t)
	user := createFundedUser(t, service, 1, core.BTC, 1)
	if err := service.Deposit(user.ID, core.USDT, 100); err != nil {
		t.Fatalf("failed to fund quote balance: %v", err)
	}

	sell, _, err := service.PlaceLimitOrder(
		user.ID, market.Symbol, core.Sell, 200*wholeCoin, 1,
	)
	if err != nil {
		t.Fatalf("unexpected sell submit error: %v", err)
	}
	buy, trades, err := service.PlaceLimitOrder(
		user.ID, market.Symbol, core.Buy, 100*wholeCoin, 1,
	)
	if err != nil {
		t.Fatalf("expected non-crossing order to be accepted: %v", err)
	}
	if len(trades) != 0 {
		t.Fatalf("expected no trades, got %d", len(trades))
	}

	book := service.Engine.GetOrderBook(market.Symbol)
	if book.BestAsk() == nil || book.BestAsk().ID != sell.ID {
		t.Fatal("expected sell order to remain active")
	}
	if book.BestBid() == nil || book.BestBid().ID != buy.ID {
		t.Fatal("expected buy order to remain active")
	}
	if len(book.AllOrders) != 2 {
		t.Fatalf("expected both orders in history, got %d", len(book.AllOrders))
	}
}

func TestExchangeServiceGetOrderKeepsOrderHistory(t *testing.T) {
	service, market := newExchangeFixture(t)
	seller := createFundedUser(t, service, 1, core.BTC, 1)
	buyer := createFundedUser(t, service, 2, core.USDT, 100)

	sellOrder, _, err := service.PlaceLimitOrder(
		seller.ID, market.Symbol, core.Sell, 100*wholeCoin, 1,
	)
	if err != nil {
		t.Fatalf("unexpected seller submit error: %v", err)
	}

	activeOrder, err := service.GetOrder(market.Symbol, sellOrder.ID)
	if err != nil {
		t.Fatalf("unexpected active order lookup error: %v", err)
	}
	if activeOrder.Status != core.Pending {
		t.Fatalf("expected pending order, got %q", activeOrder.Status)
	}

	if _, _, err := service.PlaceLimitOrder(
		buyer.ID, market.Symbol, core.Buy, 100*wholeCoin, 1,
	); err != nil {
		t.Fatalf("unexpected buyer submit error: %v", err)
	}

	filledOrder, err := service.GetOrder(market.Symbol, sellOrder.ID)
	if err != nil {
		t.Fatalf("expected filled order to remain in history: %v", err)
	}
	if filledOrder.Status != core.Filled {
		t.Fatalf("expected filled order, got %q", filledOrder.Status)
	}

	filledOrder.Price = 1
	storedOrder, err := service.GetOrder(market.Symbol, sellOrder.ID)
	if err != nil {
		t.Fatalf("unexpected second lookup error: %v", err)
	}
	if storedOrder.Price != 100*wholeCoin {
		t.Fatal("expected GetOrder to return a copy")
	}
}

func TestExchangeServiceListOrdersByUserID(t *testing.T) {
	service, market := newExchangeFixture(t)
	user := createFundedUser(t, service, 1, core.USDT, 300)
	createFundedUser(t, service, 2, core.USDT, 100)

	first, _, err := service.PlaceLimitOrder(
		user.ID, market.Symbol, core.Buy, 100*wholeCoin, 1,
	)
	if err != nil {
		t.Fatalf("unexpected first submit error: %v", err)
	}
	second, _, err := service.PlaceLimitOrder(
		user.ID, market.Symbol, core.Buy, 100*wholeCoin, 1,
	)
	if err != nil {
		t.Fatalf("unexpected second submit error: %v", err)
	}
	if _, _, err := service.PlaceLimitOrder(
		2, market.Symbol, core.Buy, 100*wholeCoin, 1,
	); err != nil {
		t.Fatalf("unexpected unrelated submit error: %v", err)
	}
	if _, err := service.CancelOrder(user.ID, market.Symbol, first.ID); err != nil {
		t.Fatalf("unexpected cancel error: %v", err)
	}

	orders, err := service.ListOrdersByUserID(market.Symbol, user.ID)
	if err != nil {
		t.Fatalf("unexpected user order history error: %v", err)
	}
	if len(orders) != 2 {
		t.Fatalf("expected two user orders, got %d", len(orders))
	}
	if orders[0].ID != first.ID || orders[1].ID != second.ID {
		t.Fatalf("expected sorted IDs [%d %d], got [%d %d]",
			first.ID, second.ID, orders[0].ID, orders[1].ID)
	}
	if orders[0].Status != core.Cancelled {
		t.Fatalf("expected cancelled order in history, got %q", orders[0].Status)
	}

	orders[0].Price = 1
	storedOrder, err := service.GetOrder(market.Symbol, first.ID)
	if err != nil {
		t.Fatalf("unexpected order lookup error: %v", err)
	}
	if storedOrder.Price != 100*wholeCoin {
		t.Fatal("expected order list to contain copies")
	}
}

func TestExchangeServiceOrderHistoryRejectsUnknownValues(t *testing.T) {
	service, market := newExchangeFixture(t)

	if _, err := service.GetOrder("UNKNOWN", 1); err == nil {
		t.Fatal("expected GetOrder to reject an unknown symbol")
	}
	if _, err := service.GetOrder(market.Symbol, 1); err == nil {
		t.Fatal("expected GetOrder to reject an unknown order")
	}
	if _, err := service.ListOrdersByUserID("UNKNOWN", 1); err == nil {
		t.Fatal("expected ListOrdersByUserID to reject an unknown symbol")
	}
	if _, err := service.ListOrdersByUserID(market.Symbol, 0); err == nil {
		t.Fatal("expected ListOrdersByUserID to reject an invalid user ID")
	}
}

func newExchangeFixture(t *testing.T) (*core.ExchangeService, core.Market) {
	t.Helper()

	service := core.NewExchangeService()
	market := btcUSDTMarket()
	if err := service.AddMarket(&market); err != nil {
		t.Fatalf("failed to add market: %v", err)
	}

	return service, market
}

func btcUSDTMarket() core.Market {
	return core.Market{
		Symbol:     "BTCUSDT",
		BaseAsset:  core.BTC,
		QuoteAsset: core.USDT,
	}
}

func createFundedUser(
	t *testing.T,
	service *core.ExchangeService,
	id uint64,
	asset core.Asset,
	amount int64,
) *core.User {
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

func assertSingleLevel(t *testing.T, levels []core.PriceLevel, price, quantity int64) {
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
