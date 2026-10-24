package asset

import "errors"

type Type string

const (
	Fiat   Type = "fiat"
	Crypto Type = "crypto"
)

// Info is the catalog record for one symbol. A deposit address is not part of it.
type Info struct {
	Symbol   string
	Name     string
	Decimals int
	Type     Type
}

// Network is one chain an asset can move on. USDT has more than one.
type Network struct {
	Symbol  string
	Network string
}

// DepositAddress belongs to one user on one network.
type DepositAddress struct {
	UserID  int64
	Symbol  string
	Network string
	Address string
}

var catalog = map[string]Info{
	"BTC":  {Symbol: "BTC", Name: "Bitcoin", Decimals: 8, Type: Crypto},
	"USDT": {Symbol: "USDT", Name: "Tether", Decimals: 6, Type: Crypto},
}

var Networks = []Network{
	{Symbol: "BTC", Network: "bitcoin"},
	{Symbol: "USDT", Network: "ethereum"},
	{Symbol: "USDT", Network: "tron"},
}

func Get(symbol string) (Info, error) {
	info, ok := catalog[symbol]
	if !ok {
		return Info{}, errors.New("unknown asset")
	}
	return info, nil
}

// Scale is 10^Decimals. BTC decimals 8 is 100_000_000.
func Scale(symbol string) (int64, error) {
	info, err := Get(symbol)
	if err != nil {
		return 0, err
	}

	scale := int64(1)
	for range info.Decimals {
		scale *= 10
	}
	return scale, nil
}
