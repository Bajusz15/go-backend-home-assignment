package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Bajusz15/go-backend-home-assignment/internal/config"
	"github.com/Bajusz15/go-backend-home-assignment/internal/database"
	"github.com/Bajusz15/go-backend-home-assignment/internal/handler"
	"github.com/Bajusz15/go-backend-home-assignment/internal/middleware"
	"github.com/Bajusz15/go-backend-home-assignment/internal/repository"
	"github.com/Bajusz15/go-backend-home-assignment/internal/service"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	httpSwagger "github.com/swaggo/http-swagger"

	_ "github.com/Bajusz15/go-backend-home-assignment/docs"
)

// @title           Food Ordering API
// @version         1.0
// @description     A RESTful food ordering system where customers can browse restaurants, place orders, and restaurants can manage orders.
//
// @host            localhost:8080
// @BasePath        /
//
// @securityDefinitions.apikey  BearerAuth
// @in                          header
// @name                        Authorization
// @description                 Enter your bearer token as: Bearer <token>
func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

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

	if err := runMigrations(db); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	userRepo := repository.NewUserRepository(db)
	restaurantRepo := repository.NewRestaurantRepository(db)
	orderRepo := repository.NewOrderRepository(db)

	authService := service.NewAuthService(userRepo, cfg.JWTSecret)
	restaurantService := service.NewRestaurantService(restaurantRepo)
	orderService := service.NewOrderService(orderRepo, restaurantRepo)

	authHandler := handler.NewAuthHandler(authService)
	restaurantHandler := handler.NewRestaurantHandler(restaurantService)
	orderHandler := handler.NewOrderHandler(orderService)

	authMW := middleware.NewAuthMiddleware(cfg.JWTSecret)

	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.RequestID)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	r.Get("/swagger/*", httpSwagger.WrapHandler)

	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Login)
		r.With(authMW.Authenticate).Get("/who-am-i", authHandler.WhoAmI)
	})

	r.Route("/restaurants", func(r chi.Router) {
		r.Get("/", restaurantHandler.List)
		r.Get("/{id}", restaurantHandler.GetByID)
		r.Get("/{id}/menu", restaurantHandler.GetMenu)
	})

	r.Route("/orders", func(r chi.Router) {
		r.Use(authMW.Authenticate)
		r.Post("/", authMW.RequireRole("customer", orderHandler.Create))
		r.Get("/", authMW.RequireRole("restaurant", orderHandler.List))
		r.Get("/{id}", authMW.RequireRole("restaurant", orderHandler.GetByID))
		r.Patch("/{id}", authMW.RequireRole("restaurant", orderHandler.UpdateStatus))
	})

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("server starting", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	}
	slog.Info("server stopped")
}

func runMigrations(db *sql.DB) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithDatabaseInstance("file://migrations", "postgres", driver)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}

	slog.Info("migrations applied successfully")
	return nil
}
