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
	query := `SELECT id, user_id, symbol, network, destination, amount, status, created_at
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
		&withdrawal.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &withdrawal, nil
}

func (wm *WithdrawalModel) SetStatus(ctx context.Context, id int64, from, to string) error {
	query := `UPDATE withdrawals SET status = $1 WHERE id = $2 AND status = $3`

	result, err := wm.db.ExecContext(ctx, query, to, id, from)
	if err != nil {
		return err
	}

	return utils.RequireOneRowAffected(result)
}
