package main

import (
	"cex_project/internal"
	"log"
)

func main() {
	engine := internal.NewEngine()

	orderBook := engine.AddOrderBook("BTCUSDT")

	firstOrder := &internal.Order{
		ID: 1,
		UserID: 1,
		Symbol: "BTCUSDT",
		Side: internal.Buy,
		Type: internal.Limit,
		Price: 10000,
		Quantity: 1,
		Filled: 0,
		Status: internal.Pending,
	}

	if err := orderBook.Add(firstOrder); err != nil {
		log.Fatal(err)
	}

	secondOrder := &internal.Order{
		ID: 2,
		UserID: 1,
		Symbol: "BTCUSDT",
		Side: internal.Sell,
		Type: internal.Limit,
		Price: 10000,
		Quantity: 1,
		Filled: 0,
		Status: internal.Pending,
	}

	if err := orderBook.Add(secondOrder); err != nil {
		log.Fatal(err)
	}

	thirdOrder := &internal.Order{
		ID: 3,
		UserID: 1,
		Symbol: "BTCUSDT",
		Side: internal.Buy,
		Type: internal.Limit,
		Price: 10000,
		Quantity: 1,
		Filled: 0,
		Status: internal.Pending,
	}

	if err := orderBook.Add(thirdOrder); err != nil {
		log.Fatal(err)
	}

	x := orderBook.Snapshot()
	log.Println(x)
}
