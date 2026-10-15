package database

import (
	"cex_project/internal/utils"
	"context"
	"database/sql"
)

type ExchangeFeesModel struct {
	db *sql.DB
}

// ExchangeFees is the Balance of all fees of the exchange
type ExchangeFees struct {
	Asset string `json:"asset"`
	Available int64 `json:"available"`
}

func NewExchangeFeesModel(db *sql.DB) *ExchangeFeesModel {
	return &ExchangeFeesModel{db: db}
}

func (em *ExchangeFeesModel) Create(ctx context.Context, exchangeFees *ExchangeFees) error {
	query := `INSERT INTO exchange_fees (asset, available) VALUES ($1, $2) ON CONFLICT (asset)
	DO UPDATE SET available = exchange_fees.available + $2`

	result, err := em.db.ExecContext(ctx, query, exchangeFees.Asset, exchangeFees.Available)
	if err != nil {
		return err
	}

	return utils.RequireOneRowAffected(result)
}

func (em *ExchangeFeesModel) Get(ctx context.Context, asset string) (*ExchangeFees, error) {
	query := `SELECT asset, available FROM exchange_fees WHERE asset = $1`

	var exchangeFees ExchangeFees

	if err := em.db.QueryRowContext(ctx, query, asset).Scan(&exchangeFees.Asset,
		 &exchangeFees.Available); err != nil {

		return nil, err
	}

	return &exchangeFees, nil
}
