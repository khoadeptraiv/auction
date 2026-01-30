package bidding

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"

	"auction-app/models"
)

var ctx = context.Background()
var db *sql.DB
var rdb *redis.Client
var nc *nats.Conn

// Lua script for atomic bidding
const placeBidScript = `
local currentPrice = redis.call("GET", KEYS[1])
if not currentPrice then
    return -1
end
local newBid = tonumber(ARGV[1])
local minIncrement = tonumber(ARGV[2])
local currentPriceNum = tonumber(currentPrice)
if newBid >= currentPriceNum + minIncrement then
    redis.call("SET", KEYS[1], newBid)
    redis.call("SET", KEYS[2], ARGV[3])
    return 1
end
return 0
`

var bidScript *redis.Script

func StartServer(database *sql.DB, redisClient *redis.Client, natsConn *nats.Conn, port string) {
	db = database
	rdb = redisClient
	nc = natsConn
	bidScript = redis.NewScript(placeBidScript)

	// Subscribe to auction events to init Redis
	nc.Subscribe("auction.events", func(m *nats.Msg) {
		handleAuctionEvent(m.Data)
	})

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	r.Post("/bid", placeBid)
	r.Get("/bid/{auctionId}/current", getCurrentBid)

	log.Printf("💰 Bidding Service listening on %s", port)
	http.ListenAndServe(port, r)
}

func placeBid(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var req models.PlaceBidRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid request"}`, http.StatusBadRequest)
		return
	}

	// Get auction info
	var minIncrement float64
	var status string
	err := db.QueryRow("SELECT min_increment, status FROM auctions WHERE id = ?", req.AuctionID).Scan(&minIncrement, &status)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error": "Auction not found"}`, http.StatusNotFound)
		return
	}
	if status != "active" {
		http.Error(w, `{"error": "Auction not active"}`, http.StatusBadRequest)
		return
	}

	// Check wallet balance
	var balance float64
	db.QueryRow("SELECT wallet_balance FROM users WHERE id = ?", userID).Scan(&balance)
	if balance < req.Amount {
		http.Error(w, `{"error": "Insufficient balance"}`, http.StatusBadRequest)
		return
	}

	// Execute Lua script
	priceKey := "auction:" + req.AuctionID + ":price"
	bidderKey := "auction:" + req.AuctionID + ":bidder"
	result, err := bidScript.Run(ctx, rdb, []string{priceKey, bidderKey}, req.Amount, minIncrement, userID).Int64()

	w.Header().Set("Content-Type", "application/json")

	switch result {
	case 1:
		// Save to Database
		bidID := uuid.New().String()
		db.Exec("INSERT INTO bids (id, auction_id, user_id, amount) VALUES (?, ?, ?, ?)",
			bidID, req.AuctionID, userID, req.Amount)
		db.Exec("UPDATE auctions SET current_price = ?, winner_id = ? WHERE id = ?",
			req.Amount, userID, req.AuctionID)

		// Publish event
		event := models.BidEvent{
			Type:      "bid:new",
			AuctionID: req.AuctionID,
			Amount:    req.Amount,
			BidderID:  userID,
		}
		data, _ := json.Marshal(event)
		nc.Publish("auction:bid", data)

		json.NewEncoder(w).Encode(models.BidResponse{
			Success:      true,
			Message:      "Bid placed successfully",
			CurrentPrice: req.Amount,
			BidderID:     userID,
		})

	case 0:
		currentPrice, _ := rdb.Get(ctx, priceKey).Float64()
		json.NewEncoder(w).Encode(models.BidResponse{
			Success:      false,
			Message:      "Bid too low",
			CurrentPrice: currentPrice,
		})

	case -1:
		http.Error(w, `{"error": "Auction not initialized"}`, http.StatusNotFound)
	}
}

func getCurrentBid(w http.ResponseWriter, r *http.Request) {
	auctionID := chi.URLParam(r, "auctionId")
	priceKey := "auction:" + auctionID + ":price"
	bidderKey := "auction:" + auctionID + ":bidder"

	price, _ := rdb.Get(ctx, priceKey).Float64()
	bidder, _ := rdb.Get(ctx, bidderKey).Result()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.CurrentBidResponse{
		AuctionID:    auctionID,
		CurrentPrice: price,
		BidderID:     bidder,
	})
}

func handleAuctionEvent(data []byte) {
	var event models.AuctionEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return
	}

	if event.Type == "auction:start" {
		priceKey := "auction:" + event.Auction.ID + ":price"
		rdb.Set(ctx, priceKey, event.Auction.StartingPrice, 0)
		log.Printf("Initialized auction %s with price %.0f", event.Auction.ID, event.Auction.StartingPrice)
	}
}
