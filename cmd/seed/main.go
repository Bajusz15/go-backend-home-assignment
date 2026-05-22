package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"

	"github.com/Bajusz15/go-backend-home-assignment/internal/config"
	"github.com/Bajusz15/go-backend-home-assignment/internal/database"
	_ "github.com/jackc/pgx/v5/stdlib"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := seed(db); err != nil {
		slog.Error("seed failed", "error", err)
		os.Exit(1)
	}

	slog.Info("seed data inserted successfully")
}

func seed(db *sql.DB) error {
	ctx := context.Background()

	hash, err := bcrypt.GenerateFromPassword([]byte("restaurant123"), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}
	passwordHash := string(hash)

	restaurants := []struct {
		userID      string
		email       string
		name        string
		restID      string
		restName    string
		description string
		address     string
	}{
		{
			userID: "a1111111-1111-1111-1111-111111111111", email: "mario@restaurant.com", name: "Mario",
			restID: "b1111111-1111-1111-1111-111111111111", restName: "Mario's Italian",
			description: "Authentic Italian cuisine with fresh pasta and wood-fired pizzas.", address: "123 Main St, New York, NY",
		},
		{
			userID: "a2222222-2222-2222-2222-222222222222", email: "sakura@restaurant.com", name: "Sakura",
			restID: "b2222222-2222-2222-2222-222222222222", restName: "Sakura Sushi",
			description: "Traditional Japanese sushi and ramen.", address: "456 Oak Ave, San Francisco, CA",
		},
		{
			userID: "a3333333-3333-3333-3333-333333333333", email: "ali@restaurant.com", name: "Ali",
			restID: "b3333333-3333-3333-3333-333333333333", restName: "Ali's Kebab House",
			description: "Middle Eastern grills and fresh mezze platters.", address: "789 Pine Rd, Chicago, IL",
		},
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, r := range restaurants {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO users (id, email, password_hash, name, role)
			 VALUES ($1, $2, $3, $4, 'restaurant')
			 ON CONFLICT (email) DO NOTHING`,
			r.userID, r.email, passwordHash, r.name,
		)
		if err != nil {
			return fmt.Errorf("inserting user %s: %w", r.email, err)
		}

		_, err = tx.ExecContext(ctx,
			`INSERT INTO restaurants (id, user_id, name, description, address)
			 VALUES ($1, $2, $3, $4, $5)
			 ON CONFLICT (user_id) DO NOTHING`,
			r.restID, r.userID, r.restName, r.description, r.address,
		)
		if err != nil {
			return fmt.Errorf("inserting restaurant %s: %w", r.restName, err)
		}
	}

	menuItems := []struct {
		id           string
		restaurantID string
		name         string
		description  string
		price        float64
		available    bool
	}{
		{"c1111111-1111-1111-1111-111111111111", "b1111111-1111-1111-1111-111111111111", "Margherita Pizza", "Classic tomato, mozzarella, and basil.", 12.99, true},
		{"c1111111-1111-1111-1111-222222222222", "b1111111-1111-1111-1111-111111111111", "Spaghetti Carbonara", "Creamy pasta with pancetta and parmesan.", 14.50, true},
		{"c1111111-1111-1111-1111-333333333333", "b1111111-1111-1111-1111-111111111111", "Tiramisu", "Traditional Italian coffee dessert.", 8.00, true},
		{"c1111111-1111-1111-1111-444444444444", "b1111111-1111-1111-1111-111111111111", "Bruschetta", "Grilled bread with tomato and basil topping.", 7.50, false},

		{"c2222222-2222-2222-2222-111111111111", "b2222222-2222-2222-2222-222222222222", "Salmon Nigiri (6pc)", "Fresh Atlantic salmon on seasoned rice.", 11.00, true},
		{"c2222222-2222-2222-2222-222222222222", "b2222222-2222-2222-2222-222222222222", "Tonkotsu Ramen", "Rich pork bone broth with chashu and egg.", 15.00, true},
		{"c2222222-2222-2222-2222-333333333333", "b2222222-2222-2222-2222-222222222222", "Edamame", "Steamed soybeans with sea salt.", 5.00, true},
		{"c2222222-2222-2222-2222-444444444444", "b2222222-2222-2222-2222-222222222222", "Dragon Roll", "Eel, avocado, and cucumber roll.", 16.50, true},

		{"c3333333-3333-3333-3333-111111111111", "b3333333-3333-3333-3333-333333333333", "Chicken Shawarma Plate", "Marinated chicken with rice, salad, and garlic sauce.", 13.00, true},
		{"c3333333-3333-3333-3333-222222222222", "b3333333-3333-3333-3333-333333333333", "Lamb Kebab", "Grilled lamb skewers with hummus and pita.", 16.00, true},
		{"c3333333-3333-3333-3333-333333333333", "b3333333-3333-3333-3333-333333333333", "Falafel Wrap", "Crispy falafel with tahini sauce in fresh pita.", 10.50, true},
		{"c3333333-3333-3333-3333-444444444444", "b3333333-3333-3333-3333-333333333333", "Baklava", "Layered pastry with honey and pistachios.", 6.00, true},
	}

	for _, item := range menuItems {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO menu_items (id, restaurant_id, name, description, price, available)
			 VALUES ($1, $2, $3, $4, $5, $6)
			 ON CONFLICT (id) DO NOTHING`,
			item.id, item.restaurantID, item.name, item.description, item.price, item.available,
		)
		if err != nil {
			return fmt.Errorf("inserting menu item %s: %w", item.name, err)
		}
	}

	return tx.Commit()
}
