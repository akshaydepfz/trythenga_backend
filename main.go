package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
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

	if err := database.EnsureRestaurantPasswordColumn(db); err != nil {
		log.Fatalf("database schema update failed: %v", err)
	}

	restaurantHandler := handler.NewRestaurantHandler(db)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /restaurants", restaurantHandler.CreateRestaurant)
	mux.HandleFunc("GET /restaurants", restaurantHandler.GetRestaurants)
	mux.HandleFunc("POST /restaurants/login", restaurantHandler.LoginRestaurant)
	mux.HandleFunc("GET /restaurants/{id}", restaurantHandler.GetRestaurantByID)
	mux.HandleFunc("PUT /restaurants/{id}", restaurantHandler.UpdateRestaurant)
	mux.HandleFunc("DELETE /restaurants/{id}", restaurantHandler.DisableRestaurant)

	server := &http.Server{
		Addr:         ":8080",
		Handler:      corsMiddleware(mux),
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

func corsMiddleware(next http.Handler) http.Handler {
	allowedOrigins := map[string]bool{
		"http://localhost:3000":  true,
		"http://localhost:5173":  true,
		"http://localhost:52564": true,
		"http://127.0.0.1:3000":  true,
		"http://127.0.0.1:5173":  true,
		"http://127.0.0.1:52564": true,
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if allowedOrigins[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
