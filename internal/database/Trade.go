package database

import (
	"database/sql"
	"context"
	"time"
)

type TradeModel struct {
	db *sql.DB
}

type Trade struct {
	ID int64
	Symbol string
	Price int64
	Quantity int64
	BuyOrderID int64
	SellOrderID int64
	BuyUserID int64
	SellUserID int64
	BuyLimitPrice int64
	BuyerFee int64
	SellerFee int64
	ExecutedAt time.Time
}

func NewTradeModel(db *sql.DB) *TradeModel {
	return &TradeModel{
		db: db,
	}
}

func (tm *TradeModel) Create(ctx context.Context, trade *Trade) error {
    query := `INSERT INTO trades (id, symbol, price, quantity, buy_order_id, sell_order_id,
	 buy_user_id, sell_user_id, buy_limit_price, buyer_fee, seller_fee, executed_at)
     VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

    result, err := tm.db.ExecContext(ctx, query, trade.ID, trade.Symbol, trade.Price,
		trade.Quantity, trade.BuyOrderID, trade.SellOrderID, trade.BuyUserID, trade.SellUserID,
		trade.BuyLimitPrice, trade.BuyerFee, trade.SellerFee, trade.ExecutedAt)

	if err != nil {
		return err
	}

    rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (tm *TradeModel) Get(ctx context.Context, id int64) (*Trade, error) {
	query := `SELECT id, symbol, price, quantity, buy_order_id, sell_order_id, buy_user_id,
	 sell_user_id, buy_limit_price, buyer_fee, seller_fee, executed_at FROM trades WHERE id = $1`

	var trade Trade

	err := tm.db.QueryRowContext(ctx, query, id).Scan(&trade.ID, &trade.Symbol, &trade.Price,
		&trade.Quantity, &trade.BuyOrderID, &trade.SellOrderID, &trade.BuyUserID,
		&trade.SellUserID, &trade.BuyLimitPrice, &trade.BuyerFee, &trade.SellerFee,
		&trade.ExecutedAt)

	if err != nil {
		return nil, err
	}

	return &trade, nil
}

func (tm *TradeModel) ListBySymbol(ctx context.Context, symbol string) ([]Trade, error) {
	query := `SELECT id, symbol, price, quantity, buy_order_id, sell_order_id, buy_user_id,
	 sell_user_id, buy_limit_price, buyer_fee, seller_fee, executed_at
	 FROM trades WHERE symbol = $1 ORDER BY executed_at DESC`

	rows, err := tm.db.QueryContext(ctx, query, symbol)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trades []Trade

	for rows.Next() {
		var trade Trade

		err := rows.Scan(&trade.ID, &trade.Symbol, &trade.Price, &trade.Quantity, &trade.BuyOrderID,
			&trade.SellOrderID, &trade.BuyUserID, &trade.SellUserID, &trade.BuyLimitPrice,
			&trade.BuyerFee, &trade.SellerFee, &trade.ExecutedAt)

		if err != nil {
			return nil, err
		}

		trades = append(trades, trade)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return trades, nil
}

func (tm *TradeModel) ListByUserID(ctx context.Context, userID int64) ([]Trade, error) {
	query := `SELECT id, symbol, price, quantity, buy_order_id, sell_order_id, buy_user_id,
	 sell_user_id, buy_limit_price, buyer_fee, seller_fee, executed_at
	 FROM trades WHERE buy_user_id = $1 OR sell_user_id = $1 ORDER BY executed_at DESC`

	rows, err := tm.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trades []Trade

	for rows.Next() {
		var trade Trade

		err := rows.Scan(&trade.ID, &trade.Symbol, &trade.Price, &trade.Quantity, &trade.BuyOrderID,
			&trade.SellOrderID, &trade.BuyUserID, &trade.SellUserID, &trade.BuyLimitPrice,
			&trade.BuyerFee, &trade.SellerFee, &trade.ExecutedAt)

		if err != nil {
			return nil, err
		}

		trades = append(trades, trade)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return trades, nil
}
