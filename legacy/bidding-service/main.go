package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"

	"auction/bidding-service/db"
	"auction/bidding-service/handlers"
	redisClient "auction/bidding-service/redis"
)

func main() {
	// Initialize Redis
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "localhost:6379"
	}
	rdb := redis.NewClient(&redis.Options{
		Addr: redisURL,
	})

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

	// Initialize PostgreSQL (for saving bids history)
	database, err := db.Connect()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer database.Close()

	// Initialize Redis scripts
	scripts := redisClient.NewScripts(rdb)

	// Initialize handlers
	bidHandler := handlers.NewBidHandler(rdb, nc, database, scripts)

	// Subscribe to auction events to sync Redis
	nc.Subscribe("auction.events", func(m *nats.Msg) {
		bidHandler.HandleAuctionEvent(m.Data)
	})

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	// Bidding routes
	r.Post("/bid", bidHandler.PlaceBid)
	r.Get("/bid/{auctionId}/current", bidHandler.GetCurrentBid)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8083"
	}

	log.Printf("Bidding Service starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
