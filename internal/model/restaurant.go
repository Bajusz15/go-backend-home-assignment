package model

import "time"

type Restaurant struct {
	ID          string    `json:"id"`
	UserID      string    `json:"-"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Address     string    `json:"address"`
	CreatedAt   time.Time `json:"created_at"`
}

type MenuItem struct {
	ID           string    `json:"id"`
	RestaurantID string    `json:"restaurant_id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Price        float64   `json:"price"`
	Available    bool      `json:"available"`
	CreatedAt    time.Time `json:"created_at"`
}

type RestaurantWithMenu struct {
	Restaurant
	Menu []MenuItem `json:"menu"`
}
