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

// List godoc
// @Summary      List all restaurants
// @Description  Returns a list of all restaurants
// @Tags         restaurants
// @Produce      json
// @Success      200  {array}  model.Restaurant
// @Router       /restaurants [get]
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

// GetByID godoc
// @Summary      Get restaurant details
// @Description  Returns a restaurant with its full menu
// @Tags         restaurants
// @Produce      json
// @Param        id   path      string  true  "Restaurant ID (UUID)"
// @Success      200  {object}  model.RestaurantWithMenu
// @Failure      404  {object}  ErrorResponse
// @Router       /restaurants/{id} [get]
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

// GetMenu godoc
// @Summary      Get restaurant menu
// @Description  Returns the menu items for a specific restaurant
// @Tags         restaurants
// @Produce      json
// @Param        id   path      string  true  "Restaurant ID (UUID)"
// @Success      200  {array}   model.MenuItem
// @Failure      404  {object}  ErrorResponse
// @Router       /restaurants/{id}/menu [get]
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
