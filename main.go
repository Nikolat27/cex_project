package main

import (
	"context"
	"fmt"
	"time"

	"cex_project/internal/core"
	"cex_project/internal/database"
	"cex_project/internal/services"
	"cex_project/internal/utils"
)

func main() {
	db, err := database.New()
	if err != nil {
		panic(err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := db.Ping(ctx); err != nil {
		panic(err)
	}

	app, err := services.NewServices(db)
	if err != nil {
		panic(err)
	}

	seller, err := app.CreateUser(ctx)
	if err != nil {
		panic(err)
	}
	buyer, err := app.CreateUser(ctx)
	if err != nil {
		panic(err)
	}

	oneBTC, err := utils.ToMinorUnits(1, "BTC")
	if err != nil {
		panic(err)
	}
	price := int64(100)
	quoteCost := price * oneBTC
	if err := app.Deposit(ctx, int64(seller.ID), "BTC", oneBTC); err != nil {
		panic(err)
	}
	if err := app.Deposit(ctx, int64(buyer.ID), "USDT", quoteCost); err != nil {
		panic(err)
	}

	if err := app.AddMarket(ctx, &core.Market{
		Symbol:     "BTCUSDT",
		BaseAsset:  core.BTC,
		QuoteAsset: core.USDT,
	}); err != nil {
		panic(err)
	}

	if err := app.PlaceLimitOrder(ctx, &core.Order{
		UserID:   seller.ID,
		Symbol:   "BTCUSDT",
		Side:     core.Sell,
		Price:    price,
		Quantity: oneBTC,
	}); err != nil {
		panic(err)
	}
	if err := app.PlaceLimitOrder(ctx, &core.Order{
		UserID:   buyer.ID,
		Symbol:   "BTCUSDT",
		Side:     core.Buy,
		Price:    price,
		Quantity: oneBTC,
	}); err != nil {
		panic(err)
	}

	printBalance(app, seller.ID, core.BTC)
	printBalance(app, seller.ID, core.USDT)
	printBalance(app, buyer.ID, core.BTC)
	printBalance(app, buyer.ID, core.USDT)

	trades, err := db.Models.Trade.ListBySymbol(ctx, "BTCUSDT")
	if err != nil {
		panic(err)
	}
	fmt.Printf("trades: %d\n", len(trades))
	for _, trade := range trades {
		fmt.Printf("trade %d price=%d quantity=%d buyer=%d seller=%d\n",
			trade.ID, trade.Price, trade.Quantity, trade.BuyUserID, trade.SellUserID)
	}
}

func printBalance(app *services.Services, userID uint64, asset core.Asset) {
	balance, err := app.Exchange.GetBalance(userID, asset)
	if err != nil {
		panic(err)
	}
	fmt.Printf("user %d %s available=%d locked=%d\n", userID, asset, balance.Available, balance.Locked)
}
