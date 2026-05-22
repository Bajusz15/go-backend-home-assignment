package handler

import (
	"net/http"

	"github.com/Bajusz15/go-backend-home-assignment/internal/middleware"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
)

type Services struct {
	Auth       *AuthHandler
	Restaurant *RestaurantHandler
	Order      *OrderHandler
	AuthMW     *middleware.AuthMiddleware
}

func NewRouter(svc Services) chi.Router {
	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.RequestID)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", svc.Auth.Register)
		r.Post("/login", svc.Auth.Login)
		r.With(svc.AuthMW.Authenticate).Get("/who-am-i", svc.Auth.WhoAmI)
	})

	r.Route("/restaurants", func(r chi.Router) {
		r.Get("/", svc.Restaurant.List)
		r.Get("/{id}", svc.Restaurant.GetByID)
		r.Get("/{id}/menu", svc.Restaurant.GetMenu)
	})

	r.Route("/orders", func(r chi.Router) {
		r.Use(svc.AuthMW.Authenticate)
		r.Post("/", svc.AuthMW.RequireRole("customer", svc.Order.Create))
		r.Get("/", svc.AuthMW.RequireRole("restaurant", svc.Order.List))
		r.Get("/{id}", svc.AuthMW.RequireRole("restaurant", svc.Order.GetByID))
		r.Patch("/{id}", svc.AuthMW.RequireRole("restaurant", svc.Order.UpdateStatus))
	})

	return r
}
