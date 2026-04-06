package models

import "time"

type Restaurant struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	OwnerName       string    `json:"owner_name"`
	Phone           string    `json:"phone"`
	Email           string    `json:"email"`
	Address         string    `json:"address"`
	City            string    `json:"city"`
	State           string    `json:"state"`
	Pincode         string    `json:"pincode"`
	Country         string    `json:"country"`
	GSTNumber       string    `json:"gst_number"`
	LogoURL         string    `json:"logo_url"`
	OpeningTime     string    `json:"opening_time"`
	ClosingTime     string    `json:"closing_time"`
	SeatingCapacity int       `json:"seating_capacity"`
	Plan            string    `json:"plan"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
