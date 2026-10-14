package database

import (
	"cex_project/internal/core"
	"context"
	"database/sql"
	"errors"
)

func (db *DB) SaveTrade(ctx context.Context, trade *core.Trade,
	settlement core.TradeSettlement) error {

	tx, err := db.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	if err := applyBalance(ctx, tx, int64(trade.BuyUserID), settlement.QuoteAsset,
		settlement.BuyerQuoteRefund, -settlement.BuyerQuoteLocked); err != nil {
		return err
	}

	if err := applyBalance(ctx, tx, int64(trade.BuyUserID), settlement.BaseAsset,
		settlement.BuyerBaseReceived, 0); err != nil {
		return err
	}

	if err := applyBalance(ctx, tx, int64(trade.SellUserID), settlement.BaseAsset,
		0, -settlement.SellerBaseLocked); err != nil {
		return err
	}

	if err := applyBalance(ctx, tx, int64(trade.SellUserID), settlement.QuoteAsset,
		settlement.SellerQuoteReceived, 0); err != nil {
		return err
	}

	if err := fillOrder(ctx, tx, int64(trade.BuyOrderID), trade.Quantity); err != nil {
		return err
	}

	if err := fillOrder(ctx, tx, int64(trade.SellOrderID), trade.Quantity); err != nil {
		return err
	}

	if err := insertTrade(ctx, tx, trade, settlement.BuyerFee, settlement.SellerFee); err != nil {
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

func insertTrade(ctx context.Context, tx *sql.Tx, trade *core.Trade, buyerFee, sellerFee int64) error {
	result, err := tx.ExecContext(ctx, `
	INSERT INTO trades (
		id, symbol, price, quantity, buy_order_id, sell_order_id,
		buy_user_id, sell_user_id, buy_limit_price, buyer_fee, seller_fee, executed_at
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
	`, int64(trade.ID), trade.Symbol, trade.Price, trade.Quantity, int64(trade.BuyOrderID),
		int64(trade.SellOrderID), int64(trade.BuyUserID), int64(trade.SellUserID), trade.BuyLimitPrice,
		buyerFee, sellerFee, trade.ExecutedAt)

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
	asset core.Asset, availableDelta, lockedDelta int64) error {

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

func addFee(ctx context.Context, tx *sql.Tx, asset core.Asset,
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
