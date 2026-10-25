package database

import (
	"context"
	"database/sql"
)

type ChainDepositModel struct {
	db *sql.DB
}

type ChainDeposit struct {
	ID int64
	UserID int64
	Symbol string
	Network string
	Address string
	TxHash string
	Amount string
	Confirmations int64
	Credited bool
	BlockNumber int64
	BlockHash string
}

func NewChainDepositModel(db *sql.DB) *ChainDepositModel {
	return &ChainDepositModel{
		db: db,
	}
}

func (m *ChainDepositModel) Create(ctx context.Context, deposit *ChainDeposit) error {
	query := `
		INSERT INTO chain_deposits (user_id, symbol, network, address, tx_hash,
		 amount, confirmations, credited, block_number, block_hash)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (network, tx_hash) DO UPDATE
		SET confirmations = EXCLUDED.confirmations,
		    block_number = EXCLUDED.block_number,
		    block_hash = EXCLUDED.block_hash
		WHERE chain_deposits.credited = false
	`

	_, err := m.db.ExecContext(ctx, query, deposit.UserID, deposit.Symbol, deposit.Network,
		deposit.Address, deposit.TxHash, deposit.Amount,
		deposit.Confirmations, deposit.Credited, deposit.BlockNumber, deposit.BlockHash)
	return err
}

func (m *ChainDepositModel) ListReady(ctx context.Context,
	 finalizedNumber uint64) ([]ChainDeposit, error) {
	query := `
	    SELECT id, user_id, symbol, network, address, tx_hash,
		 amount, confirmations, credited, block_number, block_hash
		 FROM chain_deposits
		WHERE block_number <= $1 AND credited = false
	`

	rows, err := m.db.QueryContext(ctx, query, finalizedNumber)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	deposits := []ChainDeposit{}
	for rows.Next() {
		var deposit ChainDeposit
		if err := rows.Scan(&deposit.ID, &deposit.UserID, &deposit.Symbol, &deposit.Network,
			&deposit.Address, &deposit.TxHash, &deposit.Amount, &deposit.Confirmations,
			&deposit.Credited, &deposit.BlockNumber, &deposit.BlockHash); err != nil {
			return nil, err
		}
	
		deposits = append(deposits, deposit)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return deposits, nil
}

func (m *ChainDepositModel) MarkCredit(ctx context.Context, id int64) (bool, error) {
	result, err := m.db.ExecContext(ctx,
		`UPDATE chain_deposits SET credited = true WHERE id = $1 AND credited = false`, id)
	if err != nil {
		return false, err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	
	return rows == 1, nil
}

func (m *ChainDepositModel) UnmarkCredit(ctx context.Context, id int64) error {
	_, err := m.db.ExecContext(ctx,
		`UPDATE chain_deposits SET credited = false WHERE id = $1`, id)
	return err
}
