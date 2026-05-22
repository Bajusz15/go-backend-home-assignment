package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/Bajusz15/go-backend-home-assignment/internal/model"
)

type OrderRepository struct {
	db *sql.DB
}

func NewOrderRepository(db *sql.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) Create(ctx context.Context, order *model.Order, items []model.OrderItem) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	err = tx.QueryRowContext(ctx,
		`INSERT INTO orders (customer_id, restaurant_id, status, total_price)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, created_at, updated_at`,
		order.CustomerID, order.RestaurantID, order.Status, order.TotalPrice,
	).Scan(&order.ID, &order.CreatedAt, &order.UpdatedAt)
	if err != nil {
		return err
	}

	for i := range items {
		items[i].OrderID = order.ID
		err = tx.QueryRowContext(ctx,
			`INSERT INTO order_items (order_id, menu_item_id, quantity, price_at_time)
			 VALUES ($1, $2, $3, $4)
			 RETURNING id`,
			items[i].OrderID, items[i].MenuItemID, items[i].Quantity, items[i].PriceAtTime,
		).Scan(&items[i].ID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *OrderRepository) FindByID(ctx context.Context, id string) (*model.Order, error) {
	order := &model.Order{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, customer_id, restaurant_id, status, total_price, created_at, updated_at
		 FROM orders WHERE id = $1`,
		id,
	).Scan(&order.ID, &order.CustomerID, &order.RestaurantID, &order.Status, &order.TotalPrice, &order.CreatedAt, &order.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return order, err
}

func (r *OrderRepository) FindDetailByID(ctx context.Context, id string) (*model.OrderDetail, error) {
	order, err := r.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	customer := &model.User{}
	err = r.db.QueryRowContext(ctx,
		`SELECT id, email, name, role, created_at FROM users WHERE id = $1`,
		order.CustomerID,
	).Scan(&customer.ID, &customer.Email, &customer.Name, &customer.Role, &customer.CreatedAt)
	if err != nil {
		return nil, err
	}

	items, err := r.findOrderItems(ctx, order.ID)
	if err != nil {
		return nil, err
	}

	return &model.OrderDetail{
		Order:    *order,
		Customer: customer,
		Items:    items,
	}, nil
}

func (r *OrderRepository) ListByRestaurantID(ctx context.Context, restaurantID string) ([]model.Order, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, customer_id, restaurant_id, status, total_price, created_at, updated_at
		 FROM orders WHERE restaurant_id = $1 ORDER BY created_at DESC`,
		restaurantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []model.Order
	for rows.Next() {
		var o model.Order
		if err := rows.Scan(&o.ID, &o.CustomerID, &o.RestaurantID, &o.Status, &o.TotalPrice, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	return orders, rows.Err()
}

func (r *OrderRepository) UpdateStatus(ctx context.Context, id string, expected, status model.OrderStatus) (*model.Order, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	order := &model.Order{}
	err = tx.QueryRowContext(ctx,
		`UPDATE orders SET status = $1, updated_at = NOW()
		 WHERE id = $2 AND status = $3
		 RETURNING id, customer_id, restaurant_id, status, total_price, created_at, updated_at`,
		status, id, expected,
	).Scan(&order.ID, &order.CustomerID, &order.RestaurantID, &order.Status, &order.TotalPrice, &order.CreatedAt, &order.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrConflict
	}
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return order, nil
}

func (r *OrderRepository) findOrderItems(ctx context.Context, orderID string) ([]model.OrderItem, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, order_id, menu_item_id, quantity, price_at_time
		 FROM order_items WHERE order_id = $1`,
		orderID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.OrderItem
	for rows.Next() {
		var item model.OrderItem
		if err := rows.Scan(&item.ID, &item.OrderID, &item.MenuItemID, &item.Quantity, &item.PriceAtTime); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
