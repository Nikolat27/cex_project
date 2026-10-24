package database

import (
	"cex_project/internal/utils"
	"context"
	"database/sql"
)

type DepositAddressModel struct {
	db *sql.DB
}

type DepositAddress struct {
	ID int64
	UserID int64
	Symbol string
	Network string
	Address string
	DerivationIndex int64
}

func NewDepositAddressModel(db *sql.DB) *DepositAddressModel {
	return &DepositAddressModel{db: db}
}

func (d *DepositAddressModel) NextIndex(ctx context.Context) (int64, error) {
	var index int64

	if err := d.db.QueryRowContext(ctx,
		`SELECT nextval('deposit_addresses_index_seq')`).Scan(&index); err != nil {
		return 0, err
	}

	return index, nil
}

func (d *DepositAddressModel) Create(ctx context.Context, userID int64,
	 symbol, network, address string, index int64) error {

	query := ` INSERT INTO deposit_addresses (user_id, symbol, network, address, derivation_index)
	VALUES ($1, $2, $3, $4, $5)
	`

	result, err := d.db.ExecContext(ctx, query, userID, symbol, network, address, index)
	if err != nil {
		return err
	}

	return utils.RequireOneRowAffected(result)
}

func (d *DepositAddressModel) Get(ctx context.Context, userID int64,
	 symbol, network string) (*DepositAddress, error) {

	query := `SELECT id, user_id, symbol, network, address, derivation_index FROM deposit_addresses
	WHERE user_id = $1 AND symbol = $2 AND network = $3`

	var da DepositAddress

	row := d.db.QueryRowContext(ctx, query, userID, symbol, network)
	if err := row.Scan(&da.ID, &da.UserID, &da.Symbol, &da.Network,
		 &da.Address, &da.DerivationIndex); err != nil {
		return nil, err
	}
	
	return &da, nil
}
