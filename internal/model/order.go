package model

import (
	"fmt"
	"time"
)

type OrderStatus string

const (
	OrderStatusReceived  OrderStatus = "received"
	OrderStatusPreparing OrderStatus = "preparing"
	OrderStatusReady     OrderStatus = "ready"
	OrderStatusDelivered OrderStatus = "delivered"
)

var validTransitions = map[OrderStatus]OrderStatus{
	OrderStatusReceived:  OrderStatusPreparing,
	OrderStatusPreparing: OrderStatusReady,
	OrderStatusReady:     OrderStatusDelivered,
}

func (s OrderStatus) CanTransitionTo(next OrderStatus) error {
	allowed, exists := validTransitions[s]
	if !exists {
		return fmt.Errorf("order in status %q cannot be updated", s)
	}
	if allowed != next {
		return fmt.Errorf("cannot transition from %q to %q (expected %q)", s, next, allowed)
	}
	return nil
}

func ValidOrderStatus(s string) bool {
	switch OrderStatus(s) {
	case OrderStatusReceived, OrderStatusPreparing, OrderStatusReady, OrderStatusDelivered:
		return true
	}
	return false
}

type Order struct {
	ID           string      `json:"id"`
	CustomerID   string      `json:"customer_id"`
	RestaurantID string      `json:"restaurant_id"`
	Status       OrderStatus `json:"status"`
	TotalPrice   float64     `json:"total_price"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

type OrderItem struct {
	ID          string  `json:"id"`
	OrderID     string  `json:"order_id"`
	MenuItemID  string  `json:"menu_item_id"`
	Quantity    int     `json:"quantity"`
	PriceAtTime float64 `json:"price_at_time"`
}

type OrderDetail struct {
	Order
	Customer *User       `json:"customer"`
	Items    []OrderItem `json:"items"`
}
