package models

import "time"

type MenuItem struct {
	ID           string    `json:"id"`
	RestaurantID string    `json:"restaurant_id"`
	CategoryID   string    `json:"category_id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Price        float64   `json:"price"`
	IsAvailable  bool      `json:"is_available"`
	ImageURL     string    `json:"image_url"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
