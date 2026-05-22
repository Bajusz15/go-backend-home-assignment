package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Bajusz15/go-backend-home-assignment/internal/middleware"
	"github.com/Bajusz15/go-backend-home-assignment/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

type OrderHandler struct {
	orderService *service.OrderService
	validate     *validator.Validate
}

func NewOrderHandler(orderService *service.OrderService) *OrderHandler {
	return &OrderHandler{
		orderService: orderService,
		validate:     validator.New(),
	}
}

// Create godoc
// @Summary      Place a new order
// @Description  Allows a customer to place an order for items from a restaurant's menu
// @Tags         orders
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      service.CreateOrderInput  true  "Order details"
// @Success      201   {object}  model.Order
// @Failure      400   {object}  ErrorResponse
// @Failure      401   {object}  ErrorResponse
// @Router       /orders [post]
func (h *OrderHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input service.CreateOrderInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.validate.Struct(input); err != nil {
		writeError(w, http.StatusBadRequest, formatValidationError(err))
		return
	}

	customerID := middleware.GetUserID(r.Context())

	order, err := h.orderService.Create(r.Context(), customerID, input)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrRestaurantNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, service.ErrItemNotFound):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, service.ErrItemNotAvailable):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, service.ErrItemWrongRestaurant):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	writeJSON(w, http.StatusCreated, order)
}

// List godoc
// @Summary      List restaurant orders
// @Description  Returns all orders placed with the authenticated restaurant
// @Tags         orders
// @Produce      json
// @Security     BearerAuth
// @Success      200  {array}   model.Order
// @Failure      401  {object}  ErrorResponse
// @Router       /orders [get]
func (h *OrderHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	orders, err := h.orderService.ListByRestaurant(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, orders)
}

// GetByID godoc
// @Summary      Get order details
// @Description  Returns order details including customer info and order items
// @Tags         orders
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Order ID (UUID)"
// @Success      200  {object}  model.OrderDetail
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Router       /orders/{id} [get]
func (h *OrderHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	orderID := chi.URLParam(r, "id")

	detail, err := h.orderService.GetDetail(r.Context(), userID, orderID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrOrderNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, service.ErrNotYourOrder):
			writeError(w, http.StatusForbidden, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	writeJSON(w, http.StatusOK, detail)
}

// UpdateStatus godoc
// @Summary      Update order status
// @Description  Updates the status of an order (received -> preparing -> ready -> delivered)
// @Tags         orders
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string                   true  "Order ID (UUID)"
// @Param        body  body      service.UpdateStatusInput  true  "New status"
// @Success      200   {object}  model.Order
// @Failure      400   {object}  ErrorResponse
// @Failure      401   {object}  ErrorResponse
// @Failure      403   {object}  ErrorResponse
// @Failure      404   {object}  ErrorResponse
// @Router       /orders/{id} [patch]
func (h *OrderHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	orderID := chi.URLParam(r, "id")

	var input service.UpdateStatusInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.validate.Struct(input); err != nil {
		writeError(w, http.StatusBadRequest, formatValidationError(err))
		return
	}

	order, err := h.orderService.UpdateStatus(r.Context(), userID, orderID, input)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrOrderNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, service.ErrNotYourOrder):
			writeError(w, http.StatusForbidden, err.Error())
		default:
			writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, order)
}
