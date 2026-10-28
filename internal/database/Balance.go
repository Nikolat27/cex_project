package database

import (
	"cex_project/internal/utils"
	"context"
	"database/sql"
	"errors"
	"math/big"
	"strings"
)

type BalanceModel struct {
	db *sql.DB
}

type Balance struct {
	UserID    int64  `json:"user_id"`
	Asset     string `json:"asset"`
	Available *big.Int `json:"available"`
	Locked    *big.Int `json:"locked"`
}

func NewBalanceModel(db *sql.DB) *BalanceModel {
	return &BalanceModel{db: db}
}

func (bm *BalanceModel) Create(ctx context.Context, balance *Balance) error {
	query := `INSERT INTO balances(user_id, asset, available, locked)
	 VALUES ($1, $2, $3, $4)`

	result, err := bm.db.ExecContext(ctx, query, balance.UserID, balance.Asset,
		numeric(balance.Available), numeric(balance.Locked))

	if err != nil {
		return err
	}

	return utils.RequireOneRowAffected(result)
}

func (bm *BalanceModel) Get(ctx context.Context, userID int64, asset string) (*Balance, error) {
	query := `SELECT user_id, asset, available, locked FROM balances
	 WHERE user_id = $1 AND asset = $2`

	var balance Balance

	var available, locked string
	err := bm.db.QueryRowContext(ctx, query, userID,
		asset).Scan(&balance.UserID, &balance.Asset, &available, &locked)
	if err != nil {
		return nil, err
	}
	parsed, err := parseBalance(available, locked)
	if err != nil {
		return nil, err
	}
	balance.Available = parsed.Available
	balance.Locked = parsed.Locked

	return &balance, nil
}

func (bm *BalanceModel) GetAll(ctx context.Context) ([]Balance, error) {
	query := `SELECT user_id, asset, available, locked FROM balances`

	rows, err := bm.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	balances := make([]Balance, 0)

	for rows.Next() {
		var balance Balance

		var available, locked string
		if err := rows.Scan(&balance.UserID, &balance.Asset, &available, &locked); err != nil {
			return nil, err
		}
		parsed, err := parseBalance(available, locked)
		if err != nil {
			return nil, err
		}
		balance.Available = parsed.Available
		balance.Locked = parsed.Locked

		balances = append(balances, balance)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return balances, nil
}

func (bm *BalanceModel) Delete(ctx context.Context, userID int64, asset string) error {
	query := `DELETE FROM balances WHERE user_id = $1 AND asset = $2`

	result, err := bm.db.ExecContext(ctx, query, userID, asset)
	if err != nil {
		return err
	}

	return utils.RequireOneRowAffected(result)
}

func (bm *BalanceModel) Deposit(ctx context.Context, userID int64, asset string, amount *big.Int) error {
	query := `UPDATE balances SET available = available + $1::numeric WHERE user_id = $2 AND asset = $3`

	result, err := bm.db.ExecContext(ctx, query, numeric(amount), userID, asset)
	if err != nil {
		return err
	}

	return utils.RequireOneRowAffected(result)
}

func (bm *BalanceModel) Withdraw(ctx context.Context, userID int64, asset string, amount *big.Int) error {
	query := `UPDATE balances SET available = available - $1::numeric
	 WHERE user_id = $2 AND asset = $3 AND available >= $1::numeric`

	result, err := bm.db.ExecContext(ctx, query, numeric(amount), userID, asset)
	if err != nil {
		return err
	}

	return utils.RequireOneRowAffected(result)
}

func (bm *BalanceModel) Debit(ctx context.Context, userID int64, asset string,
	amount *big.Int) error {

   query := `UPDATE balances SET locked = locked - $1::numeric
	WHERE user_id = $2 AND asset = $3 AND locked >= $1::numeric`

   result, err := bm.db.ExecContext(ctx, query, numeric(amount), userID, asset)
   if err != nil {
	   return err
   }

   return utils.RequireOneRowAffected(result)
}

func (bm *BalanceModel) UndoDebit(ctx context.Context, userID int64, asset string,
	amount *big.Int) error {

   query := `UPDATE balances SET locked = locked + $1::numeric
	WHERE user_id = $2 AND asset = $3`

   result, err := bm.db.ExecContext(ctx, query, numeric(amount), userID, asset)
   if err != nil {
	   return err
   }

   return utils.RequireOneRowAffected(result)
}

func (bm *BalanceModel) Lock(ctx context.Context, userID int64, asset string, amount *big.Int) error {
	query := `UPDATE balances SET locked = locked + $1::numeric, available = available - $1::numeric
	 WHERE user_id = $2 AND asset = $3 AND available >= $1::numeric`

	result, err := bm.db.ExecContext(ctx, query, numeric(amount), userID, asset)
	if err != nil {
		return err
	}

	return utils.RequireOneRowAffected(result)
}

func (bm *BalanceModel) LockTx(ctx context.Context, tx *sql.Tx, userID int64, asset string,
	 amount int64) error {
	query := `UPDATE balances SET locked = locked + $1::numeric, available = available - $1::numeric
	 WHERE user_id = $2 AND asset = $3 AND available >= $1::numeric`

	result, err := tx.ExecContext(ctx, query, numeric(big.NewInt(amount)), userID, asset)
	if err != nil {
		return err
	}

	return utils.RequireOneRowAffected(result)
}

func (bm *BalanceModel) Unlock(ctx context.Context, userID int64, asset string,
	 amount *big.Int) error {

	query := `UPDATE balances SET locked = locked - $1::numeric, available = available + $1::numeric
	 WHERE user_id = $2 AND asset = $3 AND locked >= $1::numeric`

	result, err := bm.db.ExecContext(ctx, query, numeric(amount), userID, asset)
	if err != nil {
		return err
	}

	return utils.RequireOneRowAffected(result)
}

func (bm *BalanceModel) UnlockTx(ctx context.Context, tx *sql.Tx, userID int64, asset string,
	 amount int64) error {

	query := `UPDATE balances SET locked = locked - $1::numeric, available = available + $1::numeric
	 WHERE user_id = $2 AND asset = $3 AND locked >= $1::numeric`

	result, err := tx.ExecContext(ctx, query, numeric(big.NewInt(amount)), userID, asset)
	if err != nil {
		return err
	}

	return utils.RequireOneRowAffected(result)
}

func numeric(amount *big.Int) string {
	if amount == nil {
		return "0"
	}
	return amount.String()
}

func parseBalance(available, locked string) (Balance, error) {
	availableAmount, err := parseNumeric(available)
	if err != nil {
		return Balance{}, err
	}
	lockedAmount, err := parseNumeric(locked)
	if err != nil {
		return Balance{}, err
	}
	return Balance{Available: availableAmount, Locked: lockedAmount}, nil
}

func parseNumeric(value string) (*big.Int, error) {
	if dot := strings.IndexByte(value, '.'); dot >= 0 {
		fraction := strings.TrimRight(value[dot+1:], "0")
		if fraction != "" {
			return nil, errors.New("fractional balance")
		}
		value = value[:dot]
	}
	amount, ok := new(big.Int).SetString(value, 10)
	if !ok {
		return nil, errors.New("invalid balance")
	}
	return amount, nil
}