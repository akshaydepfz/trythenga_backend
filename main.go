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

	if err := database.EnsureRestaurantPasswordColumn(db); err != nil {
		log.Fatalf("database schema update failed: %v", err)
	}
	if err := database.EnsureWaitersTable(db); err != nil {
		log.Fatalf("database schema update failed: %v", err)
	}
	if err := database.EnsureMenuTables(db); err != nil {
		log.Fatalf("database schema update failed: %v", err)
	}

	restaurantHandler := handler.NewRestaurantHandler(db)
	waiterHandler := handler.NewWaiterHandler(db)
	menuHandler := handler.NewMenuHandler(db)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /restaurants", restaurantHandler.CreateRestaurant)
	mux.HandleFunc("GET /restaurants", restaurantHandler.GetRestaurants)
	mux.HandleFunc("POST /restaurants/login", restaurantHandler.LoginRestaurant)
	mux.HandleFunc("GET /restaurants/{restaurant_id}/waiters", waiterHandler.GetWaitersByRestaurantID)
	mux.HandleFunc("GET /restaurants/{id}", restaurantHandler.GetRestaurantByID)
	mux.HandleFunc("PUT /restaurants/{id}", restaurantHandler.UpdateRestaurant)
	mux.HandleFunc("DELETE /restaurants/{id}", restaurantHandler.DisableRestaurant)

	mux.HandleFunc("POST /waiters", waiterHandler.CreateWaiter)
	mux.HandleFunc("GET /waiters", waiterHandler.GetWaiters)
	mux.HandleFunc("GET /waiters/{id}", waiterHandler.GetWaiterByID)
	mux.HandleFunc("PUT /waiters/{id}", waiterHandler.UpdateWaiter)
	mux.HandleFunc("DELETE /waiters/{id}", waiterHandler.DeleteWaiter)
	mux.HandleFunc("POST /waiters/login", waiterHandler.LoginWaiter)

	mux.HandleFunc("POST /api/v1/categories", menuHandler.CreateCategory)
	mux.HandleFunc("GET /api/v1/categories", menuHandler.GetCategoriesByRestaurant)
	mux.HandleFunc("PUT /api/v1/categories/{id}", menuHandler.UpdateCategory)
	mux.HandleFunc("DELETE /api/v1/categories/{id}", menuHandler.DeleteCategory)

	mux.HandleFunc("POST /api/v1/menu-items", menuHandler.CreateMenuItem)
	mux.HandleFunc("GET /api/v1/menu-items", menuHandler.GetMenuItemsByRestaurant)
	mux.HandleFunc("GET /api/v1/menu-items/category/{category_id}", menuHandler.GetMenuItemsByCategory)
	mux.HandleFunc("PUT /api/v1/menu-items/{id}", menuHandler.UpdateMenuItem)
	mux.HandleFunc("DELETE /api/v1/menu-items/{id}", menuHandler.DeleteMenuItem)

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
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
