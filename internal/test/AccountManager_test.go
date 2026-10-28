package test

import (
	"math/big"
	"sync"
	"testing"

	"cex_project/internal/core"
)

const wholeCoin = 100_000_000

func TestAccountManagerCreateGetAndDeleteUser(t *testing.T) {
	accounts := core.NewAccountManager()

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
	accounts := core.NewAccountManager()

	if _, err := accounts.CreateUser(1); err != nil {
		t.Fatalf("unexpected first create error: %v", err)
	}
	if _, err := accounts.CreateUser(1); err == nil {
		t.Fatal("expected duplicate user error")
	}
}

func TestAccountManagerRejectsDeletingUnknownUser(t *testing.T) {
	accounts := core.NewAccountManager()

	if err := accounts.DeleteUser(1); err == nil {
		t.Fatal("expected unknown user error")
	}
}

func TestSettleTradeChargesQuarterPercentFee(t *testing.T) {
	service, buyer, seller, market := newSettlementFixture(t, 1_000_000, 10_000)

	trade := &core.Trade{
		BaseAsset:     core.BTC,
		Price:         100 * wholeCoin,
		Quantity:      10_000,
		BuyUserID:     buyer.ID,
		SellUserID:    seller.ID,
		BuyLimitPrice: 100 * wholeCoin,
	}

	if err := service.SettleTrade(market, trade); err != nil {
		t.Fatalf("unexpected settlement error: %v", err)
	}

	assertBalance(t, buyer, core.BTC, 9_975, 0)
	assertBalance(t, buyer, core.USDT, 0, 0)
	assertBalance(t, seller, core.USDT, 997_500, 0)
	assertBalance(t, seller, core.BTC, 0, 0)

	if fee := service.GetFeeBalance(core.BTC); fee.Available.Cmp(big.NewInt(25)) != 0 {
		t.Fatalf("expected 25 BTC fee, got %s", fee.Available)
	}
	if fee := service.GetFeeBalance(core.USDT); fee.Available.Cmp(big.NewInt(2_500)) != 0 {
		t.Fatalf("expected 2500 USDT fee, got %s", fee.Available)
	}
}

func TestSettleTradeWithPriceImprovement(t *testing.T) {
	service, buyer, seller, market := newSettlementFixture(t, 200, 2)

	trade := &core.Trade{
		BaseAsset:     core.BTC,
		Price:         90 * wholeCoin,
		Quantity:      2,
		BuyUserID:     buyer.ID,
		SellUserID:    seller.ID,
		BuyLimitPrice: 100 * wholeCoin,
	}

	if err := service.SettleTrade(market, trade); err != nil {
		t.Fatalf("unexpected settlement error: %v", err)
	}

	assertBalance(t, buyer, core.USDT, 20, 0)
	assertBalance(t, buyer, core.BTC, 2, 0)
	assertBalance(t, seller, core.BTC, 0, 0)
	assertBalance(t, seller, core.USDT, 180, 0)
}

func TestSettleTradeRejectsInvalidTrades(t *testing.T) {
	service, buyer, seller, market := newSettlementFixture(t, 200, 2)

	tests := []struct {
		name  string
		trade *core.Trade
	}{
		{
			name:  "nil trade",
			trade: nil,
		},
		{
			name: "self trade",
			trade: &core.Trade{
				BaseAsset:     core.BTC,
				Price:         100 * wholeCoin,
				Quantity:      1,
				BuyUserID:     buyer.ID,
				SellUserID:    buyer.ID,
				BuyLimitPrice: 100 * wholeCoin,
			},
		},
		{
			name: "price exceeds buy limit",
			trade: &core.Trade{
				BaseAsset:     core.BTC,
				Price:         101 * wholeCoin,
				Quantity:      1,
				BuyUserID:     buyer.ID,
				SellUserID:    seller.ID,
				BuyLimitPrice: 100 * wholeCoin,
			},
		},
		{
			name: "zero quantity",
			trade: &core.Trade{
				BaseAsset:     core.BTC,
				Price:         100 * wholeCoin,
				Quantity:      0,
				BuyUserID:     buyer.ID,
				SellUserID:    seller.ID,
				BuyLimitPrice: 100 * wholeCoin,
			},
		},
		{
			name: "unknown buyer",
			trade: &core.Trade{
				BaseAsset:     core.BTC,
				Price:         100 * wholeCoin,
				Quantity:      1,
				BuyUserID:     999,
				SellUserID:    seller.ID,
				BuyLimitPrice: 100 * wholeCoin,
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
	assertBalance(t, buyer, core.USDT, 0, 200)
	assertBalance(t, buyer, core.BTC, 0, 0)
	assertBalance(t, seller, core.BTC, 0, 2)
	assertBalance(t, seller, core.USDT, 0, 0)
}

func TestSettleTradeRejectsInsufficientLockedFunds(t *testing.T) {
	t.Run("buyer", func(t *testing.T) {
		service, buyer, seller, market := newSettlementFixture(t, 50, 1)

		trade := &core.Trade{
			BaseAsset:     core.BTC,
			Price:         100 * wholeCoin,
			Quantity:      1,
			BuyUserID:     buyer.ID,
			SellUserID:    seller.ID,
			BuyLimitPrice: 100 * wholeCoin,
		}

		if err := service.SettleTrade(market, trade); err == nil {
			t.Fatal("expected insufficient buyer locked balance error")
		}

		assertBalance(t, buyer, core.USDT, 0, 50)
		assertBalance(t, seller, core.BTC, 0, 1)
	})

	t.Run("seller", func(t *testing.T) {
		service, buyer, seller, market := newSettlementFixture(t, 200, 1)

		trade := &core.Trade{
			BaseAsset:     core.BTC,
			Price:         100 * wholeCoin,
			Quantity:      2,
			BuyUserID:     buyer.ID,
			SellUserID:    seller.ID,
			BuyLimitPrice: 100 * wholeCoin,
		}

		if err := service.SettleTrade(market, trade); err == nil {
			t.Fatal("expected insufficient seller locked balance error")
		}

		assertBalance(t, buyer, core.USDT, 0, 200)
		assertBalance(t, seller, core.BTC, 0, 1)
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
			errs <- service.SettleTrade(market, &core.Trade{
				BaseAsset:     core.BTC,
				Price:         90 * wholeCoin,
				Quantity:      1,
				BuyUserID:     buyer.ID,
				SellUserID:    seller.ID,
				BuyLimitPrice: 100 * wholeCoin,
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

	assertBalance(t, buyer, core.USDT, tradeCount*10, 0)
	assertBalance(t, buyer, core.BTC, tradeCount, 0)
	assertBalance(t, seller, core.BTC, 0, 0)
	assertBalance(t, seller, core.USDT, tradeCount*90, 0)
}

func newSettlementFixture(
	t *testing.T,
	buyerQuote int64,
	sellerBase int64,
) (*core.ExchangeService, *core.User, *core.User, core.Market) {
	t.Helper()

	service := core.NewExchangeService()
	buyer, err := service.CreateUser(1)
	if err != nil {
		t.Fatalf("failed to create buyer: %v", err)
	}
	seller, err := service.CreateUser(2)
	if err != nil {
		t.Fatalf("failed to create seller: %v", err)
	}

	if err := buyer.Deposit(core.USDT, buyerQuote); err != nil {
		t.Fatalf("failed to fund buyer: %v", err)
	}
	if err := buyer.Lock(core.USDT, buyerQuote); err != nil {
		t.Fatalf("failed to lock buyer funds: %v", err)
	}
	if err := seller.Deposit(core.BTC, sellerBase); err != nil {
		t.Fatalf("failed to fund seller: %v", err)
	}
	if err := seller.Lock(core.BTC, sellerBase); err != nil {
		t.Fatalf("failed to lock seller funds: %v", err)
	}

	market := core.Market{
		Symbol:     "BTCUSDT",
		BaseAsset:  core.BTC,
		QuoteAsset: core.USDT,
	}

	return service, buyer, seller, market
}
