package models

import "time"

type Table struct {
	ID           string    `json:"id"`
	RestaurantID string    `json:"restaurant_id"`
	FloorID      string    `json:"floor_id"`
	TableNumber  string    `json:"table_number"`
	Capacity     int       `json:"capacity"`
	Status       string    `json:"status"`
	PosX         int       `json:"pos_x"`
	PosY         int       `json:"pos_y"`
	Shape        string    `json:"shape"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type TablePositionUpdate struct {
	ID   string `json:"id"`
	PosX int    `json:"pos_x"`
	PosY int    `json:"pos_y"`
}
