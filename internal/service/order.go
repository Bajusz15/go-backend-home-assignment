package service

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/Bajusz15/go-backend-home-assignment/internal/model"
	"github.com/Bajusz15/go-backend-home-assignment/internal/repository"
)

var (
	ErrItemsRequired       = errors.New("at least one item is required")
	ErrItemNotFound        = errors.New("one or more menu items not found")
	ErrItemNotAvailable    = errors.New("one or more menu items are not available")
	ErrItemWrongRestaurant = errors.New("all items must belong to the specified restaurant")
	ErrInvalidTransition   = errors.New("invalid status transition")
	ErrOrderNotFound       = errors.New("order not found")
	ErrNotYourOrder        = errors.New("order does not belong to your restaurant")
)

type OrderService struct {
	orderRepo      orderRepository
	restaurantRepo restaurantRepository
}

func NewOrderService(orderRepo orderRepository, restaurantRepo restaurantRepository) *OrderService {
	return &OrderService{orderRepo: orderRepo, restaurantRepo: restaurantRepo}
}

type CreateOrderInput struct {
	RestaurantID string            `json:"restaurant_id" validate:"required,uuid"`
	Items        []CreateOrderItem `json:"items" validate:"required,min=1,dive"`
}

type CreateOrderItem struct {
	MenuItemID string `json:"menu_item_id" validate:"required,uuid"`
	Quantity   int    `json:"quantity" validate:"required,min=1"`
}

type UpdateStatusInput struct {
	Status string `json:"status" validate:"required"`
}

func (s *OrderService) Create(ctx context.Context, customerID string, input CreateOrderInput) (*model.Order, error) {
	if _, err := s.restaurantRepo.FindByID(ctx, input.RestaurantID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("restaurant not found")
		}
		return nil, err
	}

	menuItemIDs := make([]string, len(input.Items))
	for i, item := range input.Items {
		menuItemIDs[i] = item.MenuItemID
	}

	menuItems, err := s.restaurantRepo.FindMenuItemsByIDs(ctx, menuItemIDs)
	if err != nil {
		return nil, err
	}

	menuItemMap := make(map[string]model.MenuItem, len(menuItems))
	for _, mi := range menuItems {
		menuItemMap[mi.ID] = mi
	}

	if len(menuItemMap) != len(input.Items) {
		return nil, ErrItemNotFound
	}

	var totalPrice float64
	orderItems := make([]model.OrderItem, 0, len(input.Items))

	for _, item := range input.Items {
		mi, ok := menuItemMap[item.MenuItemID]
		if !ok {
			return nil, ErrItemNotFound
		}
		if mi.RestaurantID != input.RestaurantID {
			return nil, ErrItemWrongRestaurant
		}
		if !mi.Available {
			return nil, fmt.Errorf("item %q is not available", mi.Name)
		}

		itemTotal := mi.Price * float64(item.Quantity)
		totalPrice += itemTotal

		orderItems = append(orderItems, model.OrderItem{
			MenuItemID:  item.MenuItemID,
			Quantity:    item.Quantity,
			PriceAtTime: mi.Price,
		})
	}

	totalPrice = math.Round(totalPrice*100) / 100

	order := &model.Order{
		CustomerID:   customerID,
		RestaurantID: input.RestaurantID,
		Status:       model.OrderStatusReceived,
		TotalPrice:   totalPrice,
	}

	if err := s.orderRepo.Create(ctx, order, orderItems); err != nil {
		return nil, err
	}

	return order, nil
}

func (s *OrderService) ListByRestaurant(ctx context.Context, userID string) ([]model.Order, error) {
	restaurant, err := s.restaurantRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	orders, err := s.orderRepo.ListByRestaurantID(ctx, restaurant.ID)
	if err != nil {
		return nil, err
	}
	if orders == nil {
		orders = []model.Order{}
	}
	return orders, nil
}

func (s *OrderService) GetDetail(ctx context.Context, userID, orderID string) (*model.OrderDetail, error) {
	restaurant, err := s.restaurantRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	detail, err := s.orderRepo.FindDetailByID(ctx, orderID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}

	if detail.RestaurantID != restaurant.ID {
		return nil, ErrNotYourOrder
	}

	return detail, nil
}

func (s *OrderService) UpdateStatus(ctx context.Context, userID, orderID string, input UpdateStatusInput) (*model.Order, error) {
	if !model.ValidOrderStatus(input.Status) {
		return nil, fmt.Errorf("invalid status: %s", input.Status)
	}

	restaurant, err := s.restaurantRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	order, err := s.orderRepo.FindByID(ctx, orderID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}

	if order.RestaurantID != restaurant.ID {
		return nil, ErrNotYourOrder
	}

	newStatus := model.OrderStatus(input.Status)
	if err := order.Status.CanTransitionTo(newStatus); err != nil {
		return nil, err
	}

	return s.orderRepo.UpdateStatus(ctx, orderID, newStatus)
}
