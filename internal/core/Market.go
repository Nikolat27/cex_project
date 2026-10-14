package core

import "errors"

type Asset string

const (
	BTC  Asset = "BTC"
	USDT Asset = "USDT"
)

type Market struct {
	Symbol     string // BTCUSDT
	BaseAsset  Asset  // BTC (Base Asset is the asset that is being traded)
	QuoteAsset Asset  // USDT (Quote Asset is the asset that is used to pay for the base asset)
}

func (m *Market) Validate() error {
	if m.Symbol == "" {
		return errors.New("symbol is required")
	}

	if m.BaseAsset == "" {
		return errors.New("base asset is required")
	}

	if m.QuoteAsset == "" {
		return errors.New("quote asset is required")
	}

	if m.BaseAsset == m.QuoteAsset {
		return errors.New("base and quote assets cannot be the same")
	}

	return nil
}
