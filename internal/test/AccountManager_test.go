package test

import (
	"sync"
	"testing"

	"cex_project/internal"
)

func TestAccountManagerCreateGetAndDeleteUser(t *testing.T) {
	accounts := internal.NewAccountManager()

	user, err := accounts.CreateUser(1)
	if err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}
	if user.ID != 1 {
		t.Fatalf("expected user ID 1, got %d", user.ID)
	}
	if accounts.GetUser(1) != user {
		t.Fatal("expected GetUser to return the created user")
	}

	if err := accounts.DeleteUser(1); err != nil {
		t.Fatalf("unexpected delete error: %v", err)
	}
	if accounts.GetUser(1) != nil {
		t.Fatal("expected deleted user to be absent")
	}
}

func TestAccountManagerRejectsDuplicateUser(t *testing.T) {
	accounts := internal.NewAccountManager()

	if _, err := accounts.CreateUser(1); err != nil {
		t.Fatalf("unexpected first create error: %v", err)
	}
	if _, err := accounts.CreateUser(1); err == nil {
		t.Fatal("expected duplicate user error")
	}
}

func TestAccountManagerRejectsDeletingUnknownUser(t *testing.T) {
	accounts := internal.NewAccountManager()

	if err := accounts.DeleteUser(1); err == nil {
		t.Fatal("expected unknown user error")
	}
}

func TestSettleTradeChargesQuarterPercentFee(t *testing.T) {
	service, buyer, seller, market := newSettlementFixture(t, 1_000_000, 10_000)

	trade := &internal.Trade{
		Price:         100,
		Quantity:      10_000,
		BuyUserID:     buyer.ID,
		SellUserID:    seller.ID,
		BuyLimitPrice: 100,
	}

	if err := service.SettleTrade(market, trade); err != nil {
		t.Fatalf("unexpected settlement error: %v", err)
	}

	assertBalance(t, buyer, internal.BTC, 9_975, 0)
	assertBalance(t, buyer, internal.USDT, 0, 0)
	assertBalance(t, seller, internal.USDT, 997_500, 0)
	assertBalance(t, seller, internal.BTC, 0, 0)

	if fee := service.GetFeeBalance(internal.BTC); fee.Available != 25 {
		t.Fatalf("expected 25 BTC fee, got %d", fee.Available)
	}
	if fee := service.GetFeeBalance(internal.USDT); fee.Available != 2_500 {
		t.Fatalf("expected 2500 USDT fee, got %d", fee.Available)
	}
}

func TestSettleTradeWithPriceImprovement(t *testing.T) {
	service, buyer, seller, market := newSettlementFixture(t, 200, 2)

	trade := &internal.Trade{
		Price:         90,
		Quantity:      2,
		BuyUserID:     buyer.ID,
		SellUserID:    seller.ID,
		BuyLimitPrice: 100,
	}

	if err := service.SettleTrade(market, trade); err != nil {
		t.Fatalf("unexpected settlement error: %v", err)
	}

	assertBalance(t, buyer, internal.USDT, 20, 0)
	assertBalance(t, buyer, internal.BTC, 2, 0)
	assertBalance(t, seller, internal.BTC, 0, 0)
	assertBalance(t, seller, internal.USDT, 180, 0)
}

func TestSettleTradeRejectsInvalidTrades(t *testing.T) {
	service, buyer, seller, market := newSettlementFixture(t, 200, 2)

	tests := []struct {
		name  string
		trade *internal.Trade
	}{
		{
			name:  "nil trade",
			trade: nil,
		},
		{
			name: "self trade",
			trade: &internal.Trade{
				Price:         100,
				Quantity:      1,
				BuyUserID:     buyer.ID,
				SellUserID:    buyer.ID,
				BuyLimitPrice: 100,
			},
		},
		{
			name: "price exceeds buy limit",
			trade: &internal.Trade{
				Price:         101,
				Quantity:      1,
				BuyUserID:     buyer.ID,
				SellUserID:    seller.ID,
				BuyLimitPrice: 100,
			},
		},
		{
			name: "zero quantity",
			trade: &internal.Trade{
				Price:         100,
				Quantity:      0,
				BuyUserID:     buyer.ID,
				SellUserID:    seller.ID,
				BuyLimitPrice: 100,
			},
		},
		{
			name: "unknown buyer",
			trade: &internal.Trade{
				Price:         100,
				Quantity:      1,
				BuyUserID:     999,
				SellUserID:    seller.ID,
				BuyLimitPrice: 100,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := service.SettleTrade(market, test.trade); err == nil {
				t.Fatal("expected settlement error")
			}
		})
	}

	// Failed settlements must not alter the arranged balances.
	assertBalance(t, buyer, internal.USDT, 0, 200)
	assertBalance(t, buyer, internal.BTC, 0, 0)
	assertBalance(t, seller, internal.BTC, 0, 2)
	assertBalance(t, seller, internal.USDT, 0, 0)
}

func TestSettleTradeRejectsInsufficientLockedFunds(t *testing.T) {
	t.Run("buyer", func(t *testing.T) {
		service, buyer, seller, market := newSettlementFixture(t, 50, 1)

		trade := &internal.Trade{
			Price:         100,
			Quantity:      1,
			BuyUserID:     buyer.ID,
			SellUserID:    seller.ID,
			BuyLimitPrice: 100,
		}

		if err := service.SettleTrade(market, trade); err == nil {
			t.Fatal("expected insufficient buyer locked balance error")
		}

		assertBalance(t, buyer, internal.USDT, 0, 50)
		assertBalance(t, seller, internal.BTC, 0, 1)
	})

	t.Run("seller", func(t *testing.T) {
		service, buyer, seller, market := newSettlementFixture(t, 200, 1)

		trade := &internal.Trade{
			Price:         100,
			Quantity:      2,
			BuyUserID:     buyer.ID,
			SellUserID:    seller.ID,
			BuyLimitPrice: 100,
		}

		if err := service.SettleTrade(market, trade); err == nil {
			t.Fatal("expected insufficient seller locked balance error")
		}

		assertBalance(t, buyer, internal.USDT, 0, 200)
		assertBalance(t, seller, internal.BTC, 0, 1)
	})
}

func TestConcurrentTradeSettlement(t *testing.T) {
	const tradeCount = 100

	service, buyer, seller, market := newSettlementFixture(
		t,
		tradeCount*100,
		tradeCount,
	)

	var wg sync.WaitGroup
	errs := make(chan error, tradeCount)

	for i := 0; i < tradeCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- service.SettleTrade(market, &internal.Trade{
				Price:         90,
				Quantity:      1,
				BuyUserID:     buyer.ID,
				SellUserID:    seller.ID,
				BuyLimitPrice: 100,
			})
		}()
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Errorf("unexpected settlement error: %v", err)
		}
	}

	assertBalance(t, buyer, internal.USDT, tradeCount*10, 0)
	assertBalance(t, buyer, internal.BTC, tradeCount, 0)
	assertBalance(t, seller, internal.BTC, 0, 0)
	assertBalance(t, seller, internal.USDT, tradeCount*90, 0)
}

func newSettlementFixture(
	t *testing.T,
	buyerQuote int64,
	sellerBase int64,
) (*internal.ExchangeService, *internal.User, *internal.User, internal.Market) {
	t.Helper()

	service := internal.NewExchangeService()
	buyer, err := service.CreateUser(1)
	if err != nil {
		t.Fatalf("failed to create buyer: %v", err)
	}
	seller, err := service.CreateUser(2)
	if err != nil {
		t.Fatalf("failed to create seller: %v", err)
	}

	if err := buyer.Deposit(internal.USDT, buyerQuote); err != nil {
		t.Fatalf("failed to fund buyer: %v", err)
	}
	if err := buyer.Lock(internal.USDT, buyerQuote); err != nil {
		t.Fatalf("failed to lock buyer funds: %v", err)
	}
	if err := seller.Deposit(internal.BTC, sellerBase); err != nil {
		t.Fatalf("failed to fund seller: %v", err)
	}
	if err := seller.Lock(internal.BTC, sellerBase); err != nil {
		t.Fatalf("failed to lock seller funds: %v", err)
	}

	market := internal.Market{
		Symbol:     "BTCUSDT",
		BaseAsset:  internal.BTC,
		QuoteAsset: internal.USDT,
	}

	return service, buyer, seller, market
}
