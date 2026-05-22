package service

import (
	"context"

	"github.com/Bajusz15/go-backend-home-assignment/internal/model"
)

type userRepository interface {
	Create(ctx context.Context, user *model.User) error
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	FindByID(ctx context.Context, id string) (*model.User, error)
}

type restaurantRepository interface {
	List(ctx context.Context) ([]model.Restaurant, error)
	FindByID(ctx context.Context, id string) (*model.Restaurant, error)
	FindByUserID(ctx context.Context, userID string) (*model.Restaurant, error)
	FindMenuItems(ctx context.Context, restaurantID string) ([]model.MenuItem, error)
	FindMenuItemsByIDs(ctx context.Context, ids []string) ([]model.MenuItem, error)
}

type orderRepository interface {
	Create(ctx context.Context, order *model.Order, items []model.OrderItem) error
	FindByID(ctx context.Context, id string) (*model.Order, error)
	FindDetailByID(ctx context.Context, id string) (*model.OrderDetail, error)
	ListByRestaurantID(ctx context.Context, restaurantID string) ([]model.Order, error)
	UpdateStatus(ctx context.Context, id string, expected, status model.OrderStatus) (*model.Order, error)
}
