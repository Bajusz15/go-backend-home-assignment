package service

import (
	"context"
	"testing"

	"github.com/Bajusz15/go-backend-home-assignment/internal/model"
	"github.com/Bajusz15/go-backend-home-assignment/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRestaurantRepo struct {
	restaurants map[string]*model.Restaurant
	menuItems   map[string][]model.MenuItem
}

func newMockRestaurantRepo() *mockRestaurantRepo {
	return &mockRestaurantRepo{
		restaurants: make(map[string]*model.Restaurant),
		menuItems:   make(map[string][]model.MenuItem),
	}
}

func (m *mockRestaurantRepo) List(_ context.Context) ([]model.Restaurant, error) {
	var result []model.Restaurant
	for _, r := range m.restaurants {
		result = append(result, *r)
	}
	return result, nil
}

func (m *mockRestaurantRepo) FindByID(_ context.Context, id string) (*model.Restaurant, error) {
	r, ok := m.restaurants[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return r, nil
}

func (m *mockRestaurantRepo) FindByUserID(_ context.Context, userID string) (*model.Restaurant, error) {
	for _, r := range m.restaurants {
		if r.UserID == userID {
			return r, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (m *mockRestaurantRepo) FindMenuItems(_ context.Context, restaurantID string) ([]model.MenuItem, error) {
	return m.menuItems[restaurantID], nil
}

func (m *mockRestaurantRepo) FindMenuItemsByIDs(_ context.Context, ids []string) ([]model.MenuItem, error) {
	idSet := make(map[string]bool, len(ids))
	for _, id := range ids {
		idSet[id] = true
	}
	var result []model.MenuItem
	for _, items := range m.menuItems {
		for _, item := range items {
			if idSet[item.ID] {
				result = append(result, item)
			}
		}
	}
	return result, nil
}

type mockOrderRepo struct {
	orders                map[string]*model.Order
	statusChangedOnUpdate bool
}

func newMockOrderRepo() *mockOrderRepo {
	return &mockOrderRepo{orders: make(map[string]*model.Order)}
}

func (m *mockOrderRepo) Create(_ context.Context, order *model.Order, _ []model.OrderItem) error {
	order.ID = "order-1"
	m.orders[order.ID] = order
	return nil
}

func (m *mockOrderRepo) FindByID(_ context.Context, id string) (*model.Order, error) {
	o, ok := m.orders[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return o, nil
}

func (m *mockOrderRepo) FindDetailByID(_ context.Context, id string) (*model.OrderDetail, error) {
	o, ok := m.orders[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return &model.OrderDetail{Order: *o}, nil
}

func (m *mockOrderRepo) ListByRestaurantID(_ context.Context, restaurantID string) ([]model.Order, error) {
	var result []model.Order
	for _, o := range m.orders {
		if o.RestaurantID == restaurantID {
			result = append(result, *o)
		}
	}
	return result, nil
}

func (m *mockOrderRepo) UpdateStatus(_ context.Context, id string, expected, status model.OrderStatus) (*model.Order, error) {
	o, ok := m.orders[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	if m.statusChangedOnUpdate || o.Status != expected {
		return nil, repository.ErrConflict
	}
	o.Status = status
	return o, nil
}

func setupOrderTest() (*OrderService, *mockRestaurantRepo, *mockOrderRepo) {
	restRepo := newMockRestaurantRepo()
	orderRepo := newMockOrderRepo()

	restRepo.restaurants["rest-1"] = &model.Restaurant{
		ID:     "rest-1",
		UserID: "rest-user-1",
		Name:   "Test Restaurant",
	}

	restRepo.menuItems["rest-1"] = []model.MenuItem{
		{ID: "item-1", RestaurantID: "rest-1", Name: "Pizza", Price: 12.99, Available: true},
		{ID: "item-2", RestaurantID: "rest-1", Name: "Pasta", Price: 14.50, Available: true},
		{ID: "item-3", RestaurantID: "rest-1", Name: "Bruschetta", Price: 7.50, Available: false},
	}

	restRepo.restaurants["rest-2"] = &model.Restaurant{
		ID:     "rest-2",
		UserID: "rest-user-2",
		Name:   "Other Restaurant",
	}
	restRepo.menuItems["rest-2"] = []model.MenuItem{
		{ID: "item-other", RestaurantID: "rest-2", Name: "Sushi", Price: 15.00, Available: true},
	}

	svc := NewOrderService(orderRepo, restRepo)
	return svc, restRepo, orderRepo
}

func TestOrderService_Create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc, _, _ := setupOrderTest()

		order, err := svc.Create(context.Background(), "customer-1", CreateOrderInput{
			RestaurantID: "rest-1",
			Items: []CreateOrderItem{
				{MenuItemID: "item-1", Quantity: 2},
				{MenuItemID: "item-2", Quantity: 1},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, model.OrderStatusReceived, order.Status)
		assert.Equal(t, 40.48, order.TotalPrice) // 12.99*2 + 14.50
		assert.Equal(t, "customer-1", order.CustomerID)
		assert.Equal(t, "rest-1", order.RestaurantID)
	})

	t.Run("allows repeated menu items", func(t *testing.T) {
		svc, _, _ := setupOrderTest()

		order, err := svc.Create(context.Background(), "customer-1", CreateOrderInput{
			RestaurantID: "rest-1",
			Items: []CreateOrderItem{
				{MenuItemID: "item-1", Quantity: 1},
				{MenuItemID: "item-1", Quantity: 2},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, 38.97, order.TotalPrice)
	})

	t.Run("restaurant not found", func(t *testing.T) {
		svc, _, _ := setupOrderTest()

		_, err := svc.Create(context.Background(), "customer-1", CreateOrderInput{
			RestaurantID: "nonexistent",
			Items:        []CreateOrderItem{{MenuItemID: "item-1", Quantity: 1}},
		})

		assert.ErrorIs(t, err, ErrRestaurantNotFound)
	})

	t.Run("menu item not found", func(t *testing.T) {
		svc, _, _ := setupOrderTest()

		_, err := svc.Create(context.Background(), "customer-1", CreateOrderInput{
			RestaurantID: "rest-1",
			Items:        []CreateOrderItem{{MenuItemID: "nonexistent", Quantity: 1}},
		})

		assert.ErrorIs(t, err, ErrItemNotFound)
	})

	t.Run("item from wrong restaurant", func(t *testing.T) {
		svc, _, _ := setupOrderTest()

		_, err := svc.Create(context.Background(), "customer-1", CreateOrderInput{
			RestaurantID: "rest-1",
			Items:        []CreateOrderItem{{MenuItemID: "item-other", Quantity: 1}},
		})

		assert.ErrorIs(t, err, ErrItemWrongRestaurant)
	})

	t.Run("unavailable item", func(t *testing.T) {
		svc, _, _ := setupOrderTest()

		_, err := svc.Create(context.Background(), "customer-1", CreateOrderInput{
			RestaurantID: "rest-1",
			Items:        []CreateOrderItem{{MenuItemID: "item-3", Quantity: 1}},
		})

		assert.ErrorIs(t, err, ErrItemNotAvailable)
	})
}

func TestOrderService_UpdateStatus(t *testing.T) {
	t.Run("valid transitions", func(t *testing.T) {
		svc, _, orderRepo := setupOrderTest()

		orderRepo.orders["order-1"] = &model.Order{
			ID:           "order-1",
			RestaurantID: "rest-1",
			Status:       model.OrderStatusReceived,
		}

		order, err := svc.UpdateStatus(context.Background(), "rest-user-1", "order-1", UpdateStatusInput{Status: "preparing"})
		require.NoError(t, err)
		assert.Equal(t, model.OrderStatusPreparing, order.Status)

		order, err = svc.UpdateStatus(context.Background(), "rest-user-1", "order-1", UpdateStatusInput{Status: "ready"})
		require.NoError(t, err)
		assert.Equal(t, model.OrderStatusReady, order.Status)

		order, err = svc.UpdateStatus(context.Background(), "rest-user-1", "order-1", UpdateStatusInput{Status: "delivered"})
		require.NoError(t, err)
		assert.Equal(t, model.OrderStatusDelivered, order.Status)
	})

	t.Run("invalid transition skipping step", func(t *testing.T) {
		svc, _, orderRepo := setupOrderTest()

		orderRepo.orders["order-2"] = &model.Order{
			ID:           "order-2",
			RestaurantID: "rest-1",
			Status:       model.OrderStatusReceived,
		}

		_, err := svc.UpdateStatus(context.Background(), "rest-user-1", "order-2", UpdateStatusInput{Status: "ready"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot transition")
	})

	t.Run("invalid transition going backwards", func(t *testing.T) {
		svc, _, orderRepo := setupOrderTest()

		orderRepo.orders["order-3"] = &model.Order{
			ID:           "order-3",
			RestaurantID: "rest-1",
			Status:       model.OrderStatusReady,
		}

		_, err := svc.UpdateStatus(context.Background(), "rest-user-1", "order-3", UpdateStatusInput{Status: "preparing"})
		assert.Error(t, err)
	})

	t.Run("order not found", func(t *testing.T) {
		svc, _, _ := setupOrderTest()

		_, err := svc.UpdateStatus(context.Background(), "rest-user-1", "nonexistent", UpdateStatusInput{Status: "preparing"})
		assert.ErrorIs(t, err, ErrOrderNotFound)
	})

	t.Run("not your order", func(t *testing.T) {
		svc, _, orderRepo := setupOrderTest()

		orderRepo.orders["order-4"] = &model.Order{
			ID:           "order-4",
			RestaurantID: "rest-2",
			Status:       model.OrderStatusReceived,
		}

		_, err := svc.UpdateStatus(context.Background(), "rest-user-1", "order-4", UpdateStatusInput{Status: "preparing"})
		assert.ErrorIs(t, err, ErrNotYourOrder)
	})

	t.Run("invalid status value", func(t *testing.T) {
		svc, _, _ := setupOrderTest()

		_, err := svc.UpdateStatus(context.Background(), "rest-user-1", "order-1", UpdateStatusInput{Status: "cancelled"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid status")
	})

	t.Run("status changed before update", func(t *testing.T) {
		svc, _, orderRepo := setupOrderTest()
		orderRepo.orders["order-5"] = &model.Order{
			ID:           "order-5",
			RestaurantID: "rest-1",
			Status:       model.OrderStatusReceived,
		}
		orderRepo.statusChangedOnUpdate = true

		_, err := svc.UpdateStatus(context.Background(), "rest-user-1", "order-5", UpdateStatusInput{Status: "preparing"})

		assert.ErrorIs(t, err, ErrStatusChanged)
	})
}

func TestOrderService_GetDetail(t *testing.T) {
	t.Run("order not found", func(t *testing.T) {
		svc, _, _ := setupOrderTest()

		_, err := svc.GetDetail(context.Background(), "rest-user-1", "missing-order")

		assert.ErrorIs(t, err, ErrOrderNotFound)
	})

	t.Run("rejects other restaurant order", func(t *testing.T) {
		svc, _, orderRepo := setupOrderTest()
		orderRepo.orders["other-order"] = &model.Order{
			ID:           "other-order",
			RestaurantID: "rest-2",
		}

		_, err := svc.GetDetail(context.Background(), "rest-user-1", "other-order")

		assert.ErrorIs(t, err, ErrNotYourOrder)
	})
}
