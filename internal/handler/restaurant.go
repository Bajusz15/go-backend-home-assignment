package handler

import (
	"errors"
	"net/http"

	"github.com/Bajusz15/go-backend-home-assignment/internal/model"
	"github.com/Bajusz15/go-backend-home-assignment/internal/repository"
	"github.com/Bajusz15/go-backend-home-assignment/internal/service"
	"github.com/go-chi/chi/v5"
)

type RestaurantHandler struct {
	restaurantService *service.RestaurantService
}

func NewRestaurantHandler(restaurantService *service.RestaurantService) *RestaurantHandler {
	return &RestaurantHandler{restaurantService: restaurantService}
}

func (h *RestaurantHandler) List(w http.ResponseWriter, r *http.Request) {
	restaurants, err := h.restaurantService.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if restaurants == nil {
		restaurants = []model.Restaurant{}
	}
	writeJSON(w, http.StatusOK, restaurants)
}

func (h *RestaurantHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	restaurant, err := h.restaurantService.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "restaurant not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, restaurant)
}

func (h *RestaurantHandler) GetMenu(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	menu, err := h.restaurantService.GetMenu(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "restaurant not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, menu)
}
