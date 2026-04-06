package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"trythenga.com/database"
	"trythenga.com/handler"
)

func main() {
	db, err := database.Connect()
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer db.Close()

	restaurantHandler := handler.NewRestaurantHandler(db)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /restaurants", restaurantHandler.CreateRestaurant)
	mux.HandleFunc("GET /restaurants", restaurantHandler.GetRestaurants)
	mux.HandleFunc("GET /restaurants/{id}", restaurantHandler.GetRestaurantByID)
	mux.HandleFunc("PUT /restaurants/{id}", restaurantHandler.UpdateRestaurant)
	mux.HandleFunc("DELETE /restaurants/{id}", restaurantHandler.DisableRestaurant)

	server := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("TryThenga backend listening on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server start failed: %v", err)
		}
	}()

	shutdownSignal := make(chan os.Signal, 1)
	signal.Notify(shutdownSignal, syscall.SIGINT, syscall.SIGTERM)
	<-shutdownSignal

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("server shutdown failed: %v", err)
	}

	log.Println("server stopped")
}
