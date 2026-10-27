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
