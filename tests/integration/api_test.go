//go:build integration

package integration

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/Bajusz15/go-backend-home-assignment/internal/config"
	"github.com/Bajusz15/go-backend-home-assignment/internal/handler"
	"github.com/Bajusz15/go-backend-home-assignment/internal/middleware"
	"github.com/Bajusz15/go-backend-home-assignment/internal/model"
	"github.com/Bajusz15/go-backend-home-assignment/internal/repository"
	"github.com/Bajusz15/go-backend-home-assignment/internal/service"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/crypto/bcrypt"
)

type APISuite struct {
	suite.Suite
	db     *sql.DB
	server *httptest.Server
}

func TestAPISuite(t *testing.T) {
	suite.Run(t, new(APISuite))
}

func (s *APISuite) SetupSuite() {
	os.Setenv("JWT_SECRET", "test-integration-secret")
	cfg, err := config.Load()
	require.NoError(s.T(), err)

	s.db, err = sql.Open("pgx", cfg.DatabaseURL)
	require.NoError(s.T(), err)
	require.NoError(s.T(), s.db.Ping())

	driver, err := postgres.WithInstance(s.db, &postgres.Config{})
	require.NoError(s.T(), err)
	m, err := migrate.NewWithDatabaseInstance("file://../../migrations", "postgres", driver)
	require.NoError(s.T(), err)
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		require.NoError(s.T(), err)
	}

	userRepo := repository.NewUserRepository(s.db)
	restaurantRepo := repository.NewRestaurantRepository(s.db)
	orderRepo := repository.NewOrderRepository(s.db)

	authService := service.NewAuthService(userRepo, cfg.JWTSecret)
	restaurantService := service.NewRestaurantService(restaurantRepo)
	orderService := service.NewOrderService(orderRepo, restaurantRepo)

	r := handler.NewRouter(handler.Services{
		Auth:       handler.NewAuthHandler(authService),
		Restaurant: handler.NewRestaurantHandler(restaurantService),
		Order:      handler.NewOrderHandler(orderService),
		AuthMW:     middleware.NewAuthMiddleware(cfg.JWTSecret),
	})

	s.server = httptest.NewServer(r)
}

func (s *APISuite) TearDownSuite() {
	s.cleanup()
	s.server.Close()
	s.db.Close()
}

func (s *APISuite) SetupTest() {
	s.cleanup()
	s.seedRestaurant()
}

func (s *APISuite) cleanup() {
	ctx := context.Background()
	s.db.ExecContext(ctx, "DELETE FROM order_items")
	s.db.ExecContext(ctx, "DELETE FROM orders")
	s.db.ExecContext(ctx, "DELETE FROM menu_items")
	s.db.ExecContext(ctx, "DELETE FROM restaurants")
	s.db.ExecContext(ctx, "DELETE FROM users")
}

func (s *APISuite) seedRestaurant() {
	ctx := context.Background()
	hash, _ := bcrypt.GenerateFromPassword([]byte("restaurant123"), bcrypt.DefaultCost)

	s.db.ExecContext(ctx,
		`INSERT INTO users (id, email, password_hash, name, role) VALUES ($1, $2, $3, $4, 'restaurant')`,
		"a1111111-1111-1111-1111-111111111111", "mario@test.com", string(hash), "Mario")

	s.db.ExecContext(ctx,
		`INSERT INTO restaurants (id, user_id, name, description, address) VALUES ($1, $2, $3, $4, $5)`,
		"b1111111-1111-1111-1111-111111111111", "a1111111-1111-1111-1111-111111111111",
		"Mario's Italian", "Test restaurant", "123 Main St")

	s.db.ExecContext(ctx,
		`INSERT INTO menu_items (id, restaurant_id, name, description, price, available) VALUES ($1, $2, $3, $4, $5, $6)`,
		"c1111111-1111-1111-1111-111111111111", "b1111111-1111-1111-1111-111111111111",
		"Margherita Pizza", "Classic pizza", 12.99, true)

	s.db.ExecContext(ctx,
		`INSERT INTO menu_items (id, restaurant_id, name, description, price, available) VALUES ($1, $2, $3, $4, $5, $6)`,
		"c2222222-2222-2222-2222-222222222222", "b1111111-1111-1111-1111-111111111111",
		"Bruschetta", "Unavailable item", 7.50, false)
}

// --- Helpers ---

func (s *APISuite) postJSON(path string, body any, token string) *http.Response {
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, s.server.URL+path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(s.T(), err)
	return resp
}

func (s *APISuite) get(path, token string) *http.Response {
	req, _ := http.NewRequest(http.MethodGet, s.server.URL+path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(s.T(), err)
	return resp
}

func (s *APISuite) patch(path string, body any, token string) *http.Response {
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPatch, s.server.URL+path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(s.T(), err)
	return resp
}

func decode[T any](t *testing.T, resp *http.Response) T {
	var v T
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&v))
	resp.Body.Close()
	return v
}

func (s *APISuite) registerCustomer(email, password, name string) string {
	resp := s.postJSON("/auth/register", map[string]string{
		"email": email, "password": password, "name": name,
	}, "")
	require.Equal(s.T(), http.StatusCreated, resp.StatusCode)
	result := decode[service.AuthResponse](s.T(), resp)
	return result.Token
}

func (s *APISuite) loginUser(email, password string) string {
	resp := s.postJSON("/auth/login", map[string]string{
		"email": email, "password": password,
	}, "")
	require.Equal(s.T(), http.StatusOK, resp.StatusCode)
	result := decode[service.AuthResponse](s.T(), resp)
	return result.Token
}

// --- Auth Tests ---

func (s *APISuite) TestAuth() {
	t := s.T()

	t.Run("register and login", func(t *testing.T) {
		token := s.registerCustomer("customer@test.com", "password123", "Test Customer")
		assert.NotEmpty(t, token)

		loginToken := s.loginUser("customer@test.com", "password123")
		assert.NotEmpty(t, loginToken)
	})

	t.Run("who-am-i", func(t *testing.T) {
		token := s.registerCustomer("whoami@test.com", "password123", "Who Am I")
		resp := s.get("/auth/who-am-i", token)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		user := decode[model.User](t, resp)
		assert.Equal(t, "whoami@test.com", user.Email)
		assert.Equal(t, model.RoleCustomer, user.Role)
	})

	t.Run("duplicate registration", func(t *testing.T) {
		s.registerCustomer("dup@test.com", "password123", "First")
		resp := s.postJSON("/auth/register", map[string]string{
			"email": "dup@test.com", "password": "password123", "name": "Second",
		}, "")
		assert.Equal(t, http.StatusConflict, resp.StatusCode)
		resp.Body.Close()
	})

	t.Run("login with wrong password", func(t *testing.T) {
		s.registerCustomer("wrongpw@test.com", "password123", "User")
		resp := s.postJSON("/auth/login", map[string]string{
			"email": "wrongpw@test.com", "password": "wrongpassword",
		}, "")
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		resp.Body.Close()
	})

	t.Run("who-am-i without token", func(t *testing.T) {
		resp := s.get("/auth/who-am-i", "")
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		resp.Body.Close()
	})
}

// --- Restaurant Tests ---

func (s *APISuite) TestRestaurants() {
	t := s.T()

	t.Run("list restaurants", func(t *testing.T) {
		resp := s.get("/restaurants", "")
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		restaurants := decode[[]model.Restaurant](t, resp)
		assert.Len(t, restaurants, 1)
		assert.Equal(t, "Mario's Italian", restaurants[0].Name)
	})

	t.Run("get restaurant with menu", func(t *testing.T) {
		resp := s.get("/restaurants/b1111111-1111-1111-1111-111111111111", "")
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		restaurant := decode[model.RestaurantWithMenu](t, resp)
		assert.Equal(t, "Mario's Italian", restaurant.Name)
		assert.Len(t, restaurant.Menu, 2)
	})

	t.Run("get nonexistent restaurant", func(t *testing.T) {
		resp := s.get("/restaurants/00000000-0000-0000-0000-000000000000", "")
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		resp.Body.Close()
	})
}

// --- Order Tests ---

func (s *APISuite) TestOrderFlow() {
	t := s.T()

	customerToken := s.registerCustomer("orderer@test.com", "password123", "Orderer")
	restaurantToken := s.loginUser("mario@test.com", "restaurant123")

	var orderID string

	t.Run("place order", func(t *testing.T) {
		resp := s.postJSON("/orders", map[string]any{
			"restaurant_id": "b1111111-1111-1111-1111-111111111111",
			"items": []map[string]any{
				{"menu_item_id": "c1111111-1111-1111-1111-111111111111", "quantity": 2},
			},
		}, customerToken)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		order := decode[model.Order](t, resp)
		assert.Equal(t, model.OrderStatusReceived, order.Status)
		assert.Equal(t, 25.98, order.TotalPrice)
		orderID = order.ID
	})

	t.Run("restaurant lists orders", func(t *testing.T) {
		resp := s.get("/orders", restaurantToken)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		orders := decode[[]model.Order](t, resp)
		assert.Len(t, orders, 1)
		assert.Equal(t, orderID, orders[0].ID)
	})

	t.Run("restaurant gets order detail", func(t *testing.T) {
		resp := s.get("/orders/"+orderID, restaurantToken)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		detail := decode[model.OrderDetail](t, resp)
		assert.Equal(t, "orderer@test.com", detail.Customer.Email)
		assert.Len(t, detail.Items, 1)
		assert.Equal(t, 12.99, detail.Items[0].PriceAtTime)
		assert.Equal(t, 2, detail.Items[0].Quantity)
	})

	t.Run("status transitions", func(t *testing.T) {
		for _, status := range []string{"preparing", "ready", "delivered"} {
			resp := s.patch("/orders/"+orderID, map[string]string{"status": status}, restaurantToken)
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			order := decode[model.Order](t, resp)
			assert.Equal(t, model.OrderStatus(status), order.Status)
		}
	})

	t.Run("invalid transition after delivered", func(t *testing.T) {
		resp := s.patch("/orders/"+orderID, map[string]string{"status": "preparing"}, restaurantToken)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		resp.Body.Close()
	})
}

func (s *APISuite) TestOrderValidation() {
	t := s.T()

	customerToken := s.registerCustomer("validator@test.com", "password123", "Validator")

	t.Run("unavailable item", func(t *testing.T) {
		resp := s.postJSON("/orders", map[string]any{
			"restaurant_id": "b1111111-1111-1111-1111-111111111111",
			"items": []map[string]any{
				{"menu_item_id": "c2222222-2222-2222-2222-222222222222", "quantity": 1},
			},
		}, customerToken)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		resp.Body.Close()
	})

	t.Run("nonexistent restaurant", func(t *testing.T) {
		resp := s.postJSON("/orders", map[string]any{
			"restaurant_id": "00000000-0000-0000-0000-000000000000",
			"items": []map[string]any{
				{"menu_item_id": "c1111111-1111-1111-1111-111111111111", "quantity": 1},
			},
		}, customerToken)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		resp.Body.Close()
	})

	t.Run("nonexistent menu item", func(t *testing.T) {
		resp := s.postJSON("/orders", map[string]any{
			"restaurant_id": "b1111111-1111-1111-1111-111111111111",
			"items": []map[string]any{
				{"menu_item_id": "00000000-0000-0000-0000-000000000000", "quantity": 1},
			},
		}, customerToken)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		resp.Body.Close()
	})

	t.Run("customer cannot list restaurant orders", func(t *testing.T) {
		resp := s.get("/orders", customerToken)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
		resp.Body.Close()
	})

	t.Run("restaurant cannot place orders", func(t *testing.T) {
		restToken := s.loginUser("mario@test.com", "restaurant123")
		resp := s.postJSON("/orders", map[string]any{
			"restaurant_id": "b1111111-1111-1111-1111-111111111111",
			"items": []map[string]any{
				{"menu_item_id": "c1111111-1111-1111-1111-111111111111", "quantity": 1},
			},
		}, restToken)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
		resp.Body.Close()
	})
}
