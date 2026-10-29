package database

import (
	"context"
	"database/sql"
	"time"
	"cex_project/internal/utils"
)

type WithdrawalModel struct {
	db *sql.DB
}

type Withdrawal struct {
	ID int64 `json:"id"`
	UserID int64 `json:"user_id"`
	Symbol string `json:"symbol"`
	Network string `json:"network"`
	Destination string `json:"destination"`
	Amount string `json:"amount"`
	Status string `json:"status"`
	TxHash *string `json:"tx_hash"`
	CreatedAt *time.Time `json:"created_at"`
}

func NewWithdrawalModel(db *sql.DB) *WithdrawalModel {
	return &WithdrawalModel{
		db: db,
	}
}

func (wm *WithdrawalModel) Create(ctx context.Context, withdrawal *Withdrawal) error {
	query := ` INSERT INTO withdrawals (user_id, symbol, network, destination, amount)
	VALUES ($1, $2, $3, $4, $5)
	`

	result, err := wm.db.ExecContext(ctx, query, withdrawal.UserID, withdrawal.Symbol,
		 withdrawal.Network, withdrawal.Destination, withdrawal.Amount)
		 
	if err != nil {
		return err
	}

	return utils.RequireOneRowAffected(result)
}

func (wm *WithdrawalModel) Get(ctx context.Context, id int64) (*Withdrawal, error) {
	query := `SELECT id, user_id, symbol, network, destination, amount, status, tx_hash, created_at
	FROM withdrawals WHERE id = $1`

	var withdrawal Withdrawal
	err := wm.db.QueryRowContext(ctx, query, id).Scan(
		&withdrawal.ID,
		&withdrawal.UserID,
		&withdrawal.Symbol,
		&withdrawal.Network,
		&withdrawal.Destination,
		&withdrawal.Amount,
		&withdrawal.Status,
		&withdrawal.TxHash,
		&withdrawal.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &withdrawal, nil
}

func (wm *WithdrawalModel) GetOldest(ctx context.Context, symbol, network, status string) (*Withdrawal, error) {
	query := `SELECT id, user_id, symbol, network, destination, amount, status, tx_hash, created_at
	FROM withdrawals
	WHERE symbol = $1 AND network = $2 AND status = $3
	ORDER BY id ASC LIMIT 1`

	var withdrawal Withdrawal
	if err := wm.db.QueryRowContext(ctx, query, symbol, network, status).Scan(
		&withdrawal.ID,
		&withdrawal.UserID,
		&withdrawal.Symbol,
		&withdrawal.Network,
		&withdrawal.Destination,
		&withdrawal.Amount,
		&withdrawal.Status,
		&withdrawal.TxHash,
		&withdrawal.CreatedAt,
	); err != nil {
		return nil, err
	}

	return &withdrawal, nil
}

func (wm *WithdrawalModel) ListByUserID(ctx context.Context, userID int64) ([]Withdrawal, error) {
	query := `SELECT id, user_id, symbol, network, destination, amount, status, tx_hash, created_at
	FROM withdrawals WHERE user_id = $1 ORDER BY id DESC`

	rows, err := wm.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	withdrawals := []Withdrawal{}
	for rows.Next() {
		var withdrawal Withdrawal
		err := rows.Scan(
			&withdrawal.ID,
			&withdrawal.UserID,
			&withdrawal.Symbol,
			&withdrawal.Network,
			&withdrawal.Destination,
			&withdrawal.Amount,
			&withdrawal.Status,
			&withdrawal.TxHash,
			&withdrawal.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		withdrawals = append(withdrawals, withdrawal)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return withdrawals, nil
}

func (wm *WithdrawalModel) SetBroadcast(ctx context.Context, id int64, txHash string) error {
	query := `UPDATE withdrawals SET status = 'broadcast', tx_hash = $1
	WHERE id = $2 AND status = 'requested'`

	result, err := wm.db.ExecContext(ctx, query, txHash, id)
	if err != nil {
		return err
	}

	return utils.RequireOneRowAffected(result)
}

func (wm *WithdrawalModel) SetStatus(ctx context.Context, id int64, from, to string) error {
	query := `UPDATE withdrawals SET status = $1 WHERE id = $2 AND status = $3`

	result, err := wm.db.ExecContext(ctx, query, to, id, from)
	if err != nil {
		return err
	}

	return utils.RequireOneRowAffected(result)
}
