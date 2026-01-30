package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"auction/user-service/db"
	"auction/user-service/handlers"
)

func main() {
	// Initialize database
	database, err := db.Connect()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer database.Close()

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(database)
	walletHandler := handlers.NewWalletHandler(database)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	// Auth routes
	r.Post("/auth/register", authHandler.Register)
	r.Post("/auth/login", authHandler.Login)

	// User routes (protected via gateway)
	r.Get("/users/me", authHandler.GetMe)

	// Wallet routes
	r.Get("/wallet/balance", walletHandler.GetBalance)
	r.Post("/wallet/deposit", walletHandler.Deposit)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	log.Printf("User Service starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
