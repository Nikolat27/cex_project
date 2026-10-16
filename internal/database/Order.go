package database

import (
	"cex_project/internal/utils"
	"context"
	"database/sql"
	"time"
)

type OrderModel struct {
	db *sql.DB
}

type Order struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Symbol    string    `json:"symbol"`
	Side      string    `json:"side"`
	Type      string    `json:"type"`
	Price     int64     `json:"price"`
	Quantity  int64     `json:"quantity"`
	Filled    int64     `json:"filled"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

func NewOrderModel(db *sql.DB) *OrderModel {
	return &OrderModel{db: db}
}

func (om *OrderModel) Create(ctx context.Context, order *Order) (*Order, error) {
	query := `INSERT INTO orders (id, user_id, symbol, side, type, price,
	 quantity, filled, status, created_at)
	 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	 RETURNING id`

	err := om.db.QueryRowContext(ctx, query, order.ID, order.UserID, order.Symbol, order.Side,
		order.Type, order.Price, order.Quantity, order.Filled, order.Status,
		order.CreatedAt).Scan(&order.ID)
	if err != nil {
		return nil, err
	}

	return order, nil
}

func (om *OrderModel) CreateTx(ctx context.Context, tx *sql.Tx, order *Order) (*Order, error) {
	query := `INSERT INTO orders (id, user_id, symbol, side, type, price,
	 quantity, filled, status, created_at)
	 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	 RETURNING id`

	err := tx.QueryRowContext(ctx, query, order.ID, order.UserID, order.Symbol, order.Side,
		order.Type, order.Price, order.Quantity, order.Filled, order.Status,
		order.CreatedAt).Scan(&order.ID)
	if err != nil {
		return nil, err
	}

	return order, nil
}

func (om *OrderModel) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM orders WHERE id = $1`

	result, err := om.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	return utils.RequireOneRowAffected(result)
}

func (om *OrderModel) Get(ctx context.Context, id int64) (*Order, error) {
	query := `SELECT id, user_id, symbol, side, type, price,
	 quantity, filled, status, created_at FROM orders WHERE id = $1`

	var order Order

	err := om.db.QueryRowContext(ctx, query, id).Scan(&order.ID, &order.UserID, &order.Symbol,
		&order.Side, &order.Type, &order.Price, &order.Quantity, &order.Filled, &order.Status,
		&order.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &order, nil
}

func (om *OrderModel) List(ctx context.Context, userID int64) ([]Order, error) {
	query := `SELECT id, user_id, symbol, side, type, price,
	 quantity, filled, status, created_at FROM orders WHERE user_id = $1`

	rows, err := om.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []Order

	for rows.Next() {
		var order Order
		err := rows.Scan(&order.ID, &order.UserID, &order.Symbol, &order.Side, &order.Type,
			&order.Price, &order.Quantity, &order.Filled, &order.Status, &order.CreatedAt)
		if err != nil {
			return nil, err
		}

		orders = append(orders, order)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return orders, nil
}

func (om *OrderModel) Update(ctx context.Context, id int64, filledAmount int64) error {
	query := `UPDATE orders SET filled = filled + $1,
	 status = CASE WHEN filled + $1 = quantity THEN 'filled' ELSE
	  'partially_filled' END WHERE id = $2 AND status IN ('pending', 'partially_filled') AND filled + $1 <= quantity`

	result, err := om.db.ExecContext(ctx, query, filledAmount, id)
	if err != nil {
		return err
	}

	return utils.RequireOneRowAffected(result)
}

func (om *OrderModel) Cancel(ctx context.Context, id int64) error {
	query := `UPDATE orders SET status = 'cancelled' WHERE id = $1
	 AND status IN ('pending', 'partially_filled')`

	result, err := om.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	return utils.RequireOneRowAffected(result)
}

func (om *OrderModel) CancelTx(ctx context.Context, tx *sql.Tx, id int64) error {
	query := `UPDATE orders SET status = 'cancelled' WHERE id = $1
	 AND status IN ('pending', 'partially_filled')`

	result, err := tx.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	return utils.RequireOneRowAffected(result)
}

