package database

import (
	"cex_project/internal/core"
	"context"
	"database/sql"
	"errors"
)

func (db *DB) SaveTrade(ctx context.Context, trade *Trade, settlement internal.TradeSettlement) error {
	tx, err := db.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	if err := applyBalance(ctx, tx, trade.BuyUserID, settlement.QuoteAsset,
		settlement.BuyerQuoteRefund, -settlement.BuyerQuoteLocked); err != nil {
		return err
	}

	if err := applyBalance(ctx, tx, trade.BuyUserID, settlement.BaseAsset,
		settlement.BuyerBaseReceived, 0); err != nil {
		return err
	}

	if err := applyBalance(ctx, tx, trade.SellUserID, settlement.BaseAsset,
		0, -settlement.SellerBaseLocked); err != nil {
		return err
	}

	if err := applyBalance(ctx, tx, trade.SellUserID, settlement.QuoteAsset,
		settlement.SellerQuoteReceived, 0); err != nil {
		return err
	}

	if err := fillOrder(ctx, tx, trade.BuyOrderID, trade.Quantity); err != nil {
		return err
	}

	if err := fillOrder(ctx, tx, trade.SellOrderID, trade.Quantity); err != nil {
		return err
	}

	trade.BuyerFee = settlement.BuyerFee
	trade.SellerFee = settlement.SellerFee
	if err := insertTrade(ctx, tx, trade); err != nil {
		return err
	}

	if err := addFee(ctx, tx, settlement.BaseAsset, settlement.BuyerFee); err != nil {
		return err
	}

	if err := addFee(ctx, tx, settlement.QuoteAsset, settlement.SellerFee); err != nil {
		return err
	}

	return tx.Commit()
}

func insertTrade(ctx context.Context, tx *sql.Tx, trade *Trade) error {
	result, err := tx.ExecContext(ctx, `
	INSERT INTO trades (
		id, symbol, price, quantity, buy_order_id, sell_order_id,
		buy_user_id, sell_user_id, buy_limit_price, buyer_fee, seller_fee, executed_at
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
	`, trade.ID, trade.Symbol, trade.Price, trade.Quantity, trade.BuyOrderID,
		trade.SellOrderID, trade.BuyUserID, trade.SellUserID, trade.BuyLimitPrice,
		trade.BuyerFee, trade.SellerFee, trade.ExecutedAt)

	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return errors.New("settlement update matched not rows")
	}

	return nil
}

func fillOrder(ctx context.Context, tx *sql.Tx, orderID, quantity int64) error {
	result, err := tx.ExecContext(ctx, `
	UPDATE orders
	SET filled = filled + $1,
		status = CASE
			WHEN filled + $1 = quantity THEN 'filled'
			ELSE 'partially_filled'
		END
	WHERE id = $2
		AND status IN ('pending', 'partially_filled')
		AND filled + $1 <= quantity
	`, quantity, orderID)

	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return errors.New("settlement update matched not rows")
	}

	return nil
}

func applyBalance(ctx context.Context, tx *sql.Tx, userID int64,
	asset internal.Asset, availableDelta, lockedDelta int64) error {

	result, err := tx.ExecContext(ctx, `
	UPDATE balances
	SET available = available + $1, locked = locked + $2
	WHERE user_id = $3 AND asset = $4
	AND available + $1 >= 0 AND locked + $2 >= 0
	`, availableDelta, lockedDelta, userID, string(asset))

	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		if availableDelta < 0 || lockedDelta < 0 {
			return errors.New("settlement update matched not rows")
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO balances (user_id, asset, available, locked)
			VALUES ($1, $2, $3, $4)
		`, userID, string(asset), availableDelta, lockedDelta)
		return err
	}

	return nil
}

func addFee(ctx context.Context, tx *sql.Tx, asset internal.Asset,
	amount int64) error {

	if amount == 0 {
		return nil
	}

	_, err := tx.ExecContext(ctx, `
	    INSERT INTO exchange_fees (asset, available)
		VALUES ($1, $2)
		ON CONFLICT (asset) DO UPDATE SET available = exchange_fees.available + $2
	`, string(asset), amount)

	return err
}
