//go:build integration

package integration

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

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
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		s.T().Skip("TEST_DATABASE_URL is required for integration tests")
	}
	const jwtSecret = "test-integration-secret"

	var err error
	s.db, err = sql.Open("pgx", databaseURL)
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

	authService := service.NewAuthService(userRepo, jwtSecret)
	restaurantService := service.NewRestaurantService(restaurantRepo)
	orderService := service.NewOrderService(orderRepo, restaurantRepo)

	r := handler.NewRouter(handler.Services{
		Auth:       handler.NewAuthHandler(authService),
		Restaurant: handler.NewRestaurantHandler(restaurantService),
		Order:      handler.NewOrderHandler(orderService),
		AuthMW:     middleware.NewAuthMiddleware(jwtSecret),
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
	_, err := s.db.ExecContext(
		context.Background(),
		"TRUNCATE order_items, orders, menu_items, restaurants, users CASCADE",
	)
	require.NoError(s.T(), err)
}

func (s *APISuite) seedRestaurant() {
	ctx := context.Background()
	hash, err := bcrypt.GenerateFromPassword([]byte("restaurant123"), bcrypt.DefaultCost)
	require.NoError(s.T(), err)

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO users (id, email, password_hash, name, role) VALUES ($1, $2, $3, $4, 'restaurant')`,
		"a1111111-1111-1111-1111-111111111111", "mario@test.com", string(hash), "Mario")
	require.NoError(s.T(), err)

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO restaurants (id, user_id, name, description, address) VALUES ($1, $2, $3, $4, $5)`,
		"b1111111-1111-1111-1111-111111111111", "a1111111-1111-1111-1111-111111111111",
		"Mario's Italian", "Test restaurant", "123 Main St")
	require.NoError(s.T(), err)

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO menu_items (id, restaurant_id, name, description, price, available) VALUES ($1, $2, $3, $4, $5, $6)`,
		"c1111111-1111-1111-1111-111111111111", "b1111111-1111-1111-1111-111111111111",
		"Margherita Pizza", "Classic pizza", 12.99, true)
	require.NoError(s.T(), err)

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO menu_items (id, restaurant_id, name, description, price, available) VALUES ($1, $2, $3, $4, $5, $6)`,
		"c2222222-2222-2222-2222-222222222222", "b1111111-1111-1111-1111-111111111111",
		"Bruschetta", "Unavailable item", 7.50, false)
	require.NoError(s.T(), err)
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

func decodeAuthResponse(t *testing.T, resp *http.Response) service.AuthResponse {
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.NotContains(t, string(body), "password_hash")

	var result service.AuthResponse
	require.NoError(t, json.Unmarshal(body, &result))
	return result
}

func decodeError(t *testing.T, resp *http.Response) handler.ErrorResponse {
	return decode[handler.ErrorResponse](t, resp)
}

func (s *APISuite) registerCustomer(email, password, name string) string {
	resp := s.postJSON("/auth/register", map[string]string{
		"email": email, "password": password, "name": name,
	}, "")
	require.Equal(s.T(), http.StatusCreated, resp.StatusCode)
	result := decodeAuthResponse(s.T(), resp)
	return result.Token
}

func (s *APISuite) loginUser(email, password string) string {
	resp := s.postJSON("/auth/login", map[string]string{
		"email": email, "password": password,
	}, "")
	require.Equal(s.T(), http.StatusOK, resp.StatusCode)
	result := decodeAuthResponse(s.T(), resp)
	return result.Token
}

// --- Auth Tests ---

func (s *APISuite) TestAuth() {
	t := s.T()

	t.Run("register and login", func(t *testing.T) {
		registerResp := s.postJSON("/auth/register", map[string]string{
			"email": "customer@test.com", "password": "password123", "name": "Test Customer",
		}, "")
		require.Equal(t, http.StatusCreated, registerResp.StatusCode)
		registered := decodeAuthResponse(t, registerResp)
		assert.NotEmpty(t, registered.Token)
		require.NotNil(t, registered.User)
		assert.Equal(t, "customer@test.com", registered.User.Email)
		assert.Equal(t, "Test Customer", registered.User.Name)
		assert.Equal(t, model.RoleCustomer, registered.User.Role)

		loginResp := s.postJSON("/auth/login", map[string]string{
			"email": "customer@test.com", "password": "password123",
		}, "")
		require.Equal(t, http.StatusOK, loginResp.StatusCode)
		loggedIn := decodeAuthResponse(t, loginResp)
		assert.NotEmpty(t, loggedIn.Token)
		require.NotNil(t, loggedIn.User)
		assert.Equal(t, "customer@test.com", loggedIn.User.Email)
		assert.Equal(t, "Test Customer", loggedIn.User.Name)
		assert.Equal(t, model.RoleCustomer, loggedIn.User.Role)
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

		errResp := decodeError(t, resp)
		assert.NotEmpty(t, errResp.Error.Message)
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

	t.Run("get restaurant menu", func(t *testing.T) {
		resp := s.get("/restaurants/b1111111-1111-1111-1111-111111111111/menu", "")
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		menu := decode[[]model.MenuItem](t, resp)
		assert.Len(t, menu, 2)
		assert.ElementsMatch(t, []string{"Bruschetta", "Margherita Pizza"}, []string{menu[0].Name, menu[1].Name})
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

		errResp := decodeError(t, resp)
		assert.NotEmpty(t, errResp.Error.Message)
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

	t.Run("empty item list", func(t *testing.T) {
		resp := s.postJSON("/orders", map[string]any{
			"restaurant_id": "b1111111-1111-1111-1111-111111111111",
			"items":         []map[string]any{},
		}, customerToken)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		resp.Body.Close()
	})

	t.Run("invalid item quantity", func(t *testing.T) {
		resp := s.postJSON("/orders", map[string]any{
			"restaurant_id": "b1111111-1111-1111-1111-111111111111",
			"items": []map[string]any{
				{"menu_item_id": "c1111111-1111-1111-1111-111111111111", "quantity": 0},
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
