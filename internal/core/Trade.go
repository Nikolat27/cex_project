package core

import (
	"cex_project/internal/utils"
	"errors"
	"sort"
	"time"
)

type Trade struct {
	ID            uint64
	Symbol        string
	BaseAsset     Asset
	QuoteAsset    Asset
	Price         int64
	Quantity      int64
	BuyOrderID    uint64
	SellOrderID   uint64
	BuyUserID     uint64
	SellUserID    uint64
	BuyLimitPrice int64
	ExecutedAt    time.Time
}

type TradeSettlement struct {
	BaseAsset           Asset
	QuoteAsset          Asset
	BuyerQuoteLocked    int64
	BuyerQuoteRefund    int64
	BuyerBaseReceived   int64
	SellerBaseLocked    int64
	SellerQuoteReceived int64
	BuyerFee            int64
	SellerFee           int64
}

func (t *Trade) Settlement() (TradeSettlement, error) {
	if t.Price > t.BuyLimitPrice {
		return TradeSettlement{}, errors.New("trade price exceeds buy limit")
	}

	reserved, err := utils.SafeMultiply(t.BuyLimitPrice, t.Quantity)
	if err != nil {
		return TradeSettlement{}, err
	}

	actual, err := utils.SafeMultiply(t.Price, t.Quantity)
	if err != nil {
		return TradeSettlement{}, err
	}

	buyerFee, err := feeAmount(t.Quantity)
	if err != nil {
		return TradeSettlement{}, err
	}

	sellerFee, err := feeAmount(actual)
	if err != nil {
		return TradeSettlement{}, err
	}

	return TradeSettlement{
		BaseAsset:           t.BaseAsset,
		QuoteAsset:          t.QuoteAsset,
		BuyerQuoteLocked:    reserved,
		BuyerQuoteRefund:    reserved - actual,
		BuyerBaseReceived:   t.Quantity - buyerFee,
		SellerBaseLocked:    t.Quantity,
		SellerQuoteReceived: actual - sellerFee,
		BuyerFee:            buyerFee,
		SellerFee:           sellerFee,
	}, nil
}

func canMatch(taker, maker *Order) bool {
	if maker == nil || taker == nil {
		return false
	}

	if taker.Side == Buy {
		return maker.Price <= taker.Price
	}

	return maker.Price >= taker.Price
}

// executeTrade -> Create the trade
func executeTrade(taker, maker *Order, symbol string, baseAsset, quoteAsset Asset) (Trade, error) {
	quantity := taker.Remaining()

	if maker.Remaining() < quantity {
		quantity = maker.Remaining()
	}

	newTakerFilled, err := utils.SafeAdd(taker.Filled, quantity)
	if err != nil {
		return Trade{}, err
	}

	newMakerFilled, err := utils.SafeAdd(maker.Filled, quantity)
	if err != nil {
		return Trade{}, err
	}

	taker.Filled = newTakerFilled
	maker.Filled = newMakerFilled

	trade := Trade{
		Symbol:     symbol,
		Price:      maker.Price,
		Quantity:   quantity,
		BaseAsset:  baseAsset,
		QuoteAsset: quoteAsset,
	}

	if taker.Side == Buy {
		trade.BuyOrderID = taker.ID
		trade.SellOrderID = maker.ID

		trade.BuyUserID = taker.UserID
		trade.SellUserID = maker.UserID

		trade.BuyLimitPrice = taker.Price
	} else {
		trade.BuyOrderID = maker.ID
		trade.SellOrderID = taker.ID

		trade.BuyUserID = maker.UserID
		trade.SellUserID = taker.UserID

		trade.BuyLimitPrice = maker.Price
	}

	return trade, nil
}

func sortByHighestPrice(order *Order, orders []*Order) []*Order {
	for i, existing := range orders {
		if order.Price > existing.Price ||
			(order.Price == existing.Price && order.ID < existing.ID) {

			orders = append(orders, nil)
			copy(orders[i+1:], orders[i:])
			orders[i] = order

			return orders
		}
	}

	return append(orders, order)
}

func sortByLowestPrice(order *Order, orders []*Order) []*Order {
	for i, existing := range orders {
		if order.Price < existing.Price ||
			(order.Price == existing.Price && order.ID < existing.ID) {

			orders = append(orders, nil)
			copy(orders[i+1:], orders[i:])
			orders[i] = order

			return orders
		}
	}

	return append(orders, order)
}

func sortByID(trades []Trade) []Trade {
	sortedtrades := append([]Trade(nil), trades...)

	sort.Slice(sortedtrades, func(i, j int) bool {
		return sortedtrades[i].ID < sortedtrades[j].ID
	})

	return sortedtrades
}
