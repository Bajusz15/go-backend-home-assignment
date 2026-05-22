package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/Bajusz15/go-backend-home-assignment/internal/model"
)

type RestaurantRepository struct {
	db *sql.DB
}

func NewRestaurantRepository(db *sql.DB) *RestaurantRepository {
	return &RestaurantRepository{db: db}
}

func (r *RestaurantRepository) List(ctx context.Context) ([]model.Restaurant, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, name, description, address, created_at FROM restaurants ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var restaurants []model.Restaurant
	for rows.Next() {
		var rest model.Restaurant
		if err := rows.Scan(&rest.ID, &rest.UserID, &rest.Name, &rest.Description, &rest.Address, &rest.CreatedAt); err != nil {
			return nil, err
		}
		restaurants = append(restaurants, rest)
	}
	return restaurants, rows.Err()
}

func (r *RestaurantRepository) FindByID(ctx context.Context, id string) (*model.Restaurant, error) {
	rest := &model.Restaurant{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, name, description, address, created_at FROM restaurants WHERE id = $1`,
		id,
	).Scan(&rest.ID, &rest.UserID, &rest.Name, &rest.Description, &rest.Address, &rest.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return rest, err
}

func (r *RestaurantRepository) FindByUserID(ctx context.Context, userID string) (*model.Restaurant, error) {
	rest := &model.Restaurant{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, name, description, address, created_at FROM restaurants WHERE user_id = $1`,
		userID,
	).Scan(&rest.ID, &rest.UserID, &rest.Name, &rest.Description, &rest.Address, &rest.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return rest, err
}

func (r *RestaurantRepository) FindMenuItems(ctx context.Context, restaurantID string) ([]model.MenuItem, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, restaurant_id, name, description, price, available, created_at
		 FROM menu_items WHERE restaurant_id = $1 ORDER BY name`,
		restaurantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.MenuItem
	for rows.Next() {
		var item model.MenuItem
		if err := rows.Scan(&item.ID, &item.RestaurantID, &item.Name, &item.Description, &item.Price, &item.Available, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *RestaurantRepository) FindMenuItemsByIDs(ctx context.Context, ids []string) ([]model.MenuItem, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	query := `SELECT id, restaurant_id, name, description, price, available, created_at
	           FROM menu_items WHERE id = ANY($1)`

	rows, err := r.db.QueryContext(ctx, query, pqStringArray(ids))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.MenuItem
	for rows.Next() {
		var item model.MenuItem
		if err := rows.Scan(&item.ID, &item.RestaurantID, &item.Name, &item.Description, &item.Price, &item.Available, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

type stringArray []string

func pqStringArray(a []string) stringArray {
	return stringArray(a)
}

func (a stringArray) Value() (interface{}, error) {
	return "{" + joinStrings(a) + "}", nil
}

func joinStrings(a []string) string {
	if len(a) == 0 {
		return ""
	}
	result := `"` + a[0] + `"`
	for _, s := range a[1:] {
		result += `,"` + s + `"`
	}
	return result
}
