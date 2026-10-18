package database

import (
	"cex_project/internal/utils"
	"context"
	"database/sql"
)

type ExchangeFeeModel struct {
	db *sql.DB
}

// ExchangeFees is the Balance of all fees of the exchange
type ExchangeFee struct {
	Asset string `json:"asset"`
	Available int64 `json:"available"`
}

func NewExchangeFeeModel(db *sql.DB) *ExchangeFeeModel {
	return &ExchangeFeeModel{db: db}
}

func (em *ExchangeFeeModel) Create(ctx context.Context, exchangeFee *ExchangeFee) error {
	query := `INSERT INTO exchange_fees (asset, available) VALUES ($1, $2) ON CONFLICT (asset)
	DO UPDATE SET available = exchange_fees.available + $2`

	result, err := em.db.ExecContext(ctx, query, exchangeFee.Asset, exchangeFee.Available)
	if err != nil {
		return err
	}

	return utils.RequireOneRowAffected(result)
}

func (em *ExchangeFeeModel) Get(ctx context.Context, asset string) (*ExchangeFee, error) {
	query := `SELECT asset, available FROM exchange_fees WHERE asset = $1`

	var exchangeFee ExchangeFee

	if err := em.db.QueryRowContext(ctx, query, asset).Scan(&exchangeFee.Asset,
		 &exchangeFee.Available); err != nil {

		return nil, err
	}

	return &exchangeFee, nil
}

func (em *ExchangeFeeModel) GetAll(ctx context.Context) ([]ExchangeFee, error) {
	query := `SELECT asset, available FROM exchange_fees`

	rows, err := em.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	var exchangeFees []ExchangeFee

	for rows.Next() {
		var exchangeFee ExchangeFee
		if err := rows.Scan(&exchangeFee.Asset, &exchangeFee.Available); err != nil {
			return nil, err
		}

		exchangeFees = append(exchangeFees, exchangeFee)
	}


	return exchangeFees, nil
}
