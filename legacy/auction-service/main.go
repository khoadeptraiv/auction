package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/nats-io/nats.go"

	"auction/auction-service/db"
	"auction/auction-service/handlers"
)

func main() {
	// Initialize database
	database, err := db.Connect()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer database.Close()

	// Initialize NATS
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = nats.DefaultURL
	}
	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatal("Failed to connect to NATS:", err)
	}
	defer nc.Close()

	// Initialize handlers
	auctionHandler := handlers.NewAuctionHandler(database, nc)

	// Start auction scheduler (check for ending auctions)
	go auctionHandler.StartScheduler()

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	// Public routes
	r.Get("/auctions", auctionHandler.List)
	r.Get("/auctions/active", auctionHandler.ListActive)
	r.Get("/auctions/{id}", auctionHandler.Get)

	// Protected routes (auth checked via X-User-ID header from gateway)
	r.Post("/auctions", auctionHandler.Create)
	r.Get("/auctions/my", auctionHandler.ListMy)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	log.Printf("Auction Service starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
