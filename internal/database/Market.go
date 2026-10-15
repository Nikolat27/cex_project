package database

import (
	"cex_project/internal/utils"
	"context"
	"database/sql"
)

type MarketModel struct {
	db *sql.DB
}

func NewMarketModel(db *sql.DB) *MarketModel {
	return &MarketModel{db: db}
}

type Market struct {
	Symbol     string `json:"symbol"`
	BaseAsset  string `json:"base_asset"`
	QuoteAsset string `json:"quote_asset"`
}

func (mm *MarketModel) Create(ctx context.Context, market *Market) error {
	query := `INSERT INTO markets (symbol, base_asset, quote_asset)
	 VALUES ($1, $2, $3)
	 RETURNING symbol, base_asset, quote_asset`

	err := mm.db.QueryRowContext(ctx, query, market.Symbol, market.BaseAsset,
		market.QuoteAsset).Scan(&market.Symbol, &market.BaseAsset, &market.QuoteAsset)
	if err != nil {
		return err
	}

	return nil
}

func (mm *MarketModel) GetAll(ctx context.Context) ([]Market, error) {
	query := `SELECT symbol, base_asset, quote_asset FROM markets ORDER BY symbol`

	rows, err := mm.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	markets := []Market{}

	for rows.Next() {
		var market Market
		if err := rows.Scan(&market.Symbol, &market.BaseAsset,
			&market.QuoteAsset); err != nil {

			return nil, err
		}

		markets = append(markets, market)
	}

	return markets, rows.Err()
}

func (mm *MarketModel) GetBySymbol(ctx context.Context, symbol string) (*Market, error) {
	query := `SELECT symbol, base_asset, quote_asset FROM markets WHERE symbol = $1`

	var market Market
	err := mm.db.QueryRowContext(ctx, query, symbol).Scan(&market.Symbol,
		&market.BaseAsset, &market.QuoteAsset)

	if err != nil {
		return nil, err
	}

	return &market, nil
}

func (mm *MarketModel) Delete(ctx context.Context, symbol string) error {
	query := `DELETE FROM markets WHERE symbol = $1`

	result, err := mm.db.ExecContext(ctx, query, symbol)
	if err != nil {
		return err
	}

	return utils.RequireOneRowAffected(result)
}
