package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"

	"auction/bidding-service/models"
	redisClient "auction/bidding-service/redis"
)

var ctx = context.Background()

type BidHandler struct {
	rdb     *redis.Client
	nc      *nats.Conn
	db      *sql.DB
	scripts *redisClient.Scripts
}

func NewBidHandler(rdb *redis.Client, nc *nats.Conn, db *sql.DB, scripts *redisClient.Scripts) *BidHandler {
	return &BidHandler{
		rdb:     rdb,
		nc:      nc,
		db:      db,
		scripts: scripts,
	}
}

func (h *BidHandler) PlaceBid(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, `{"error": "User not authenticated"}`, http.StatusUnauthorized)
		return
	}

	var req models.PlaceBidRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.AuctionID == "" {
		http.Error(w, `{"error": "auction_id is required"}`, http.StatusBadRequest)
		return
	}
	if req.Amount <= 0 {
		http.Error(w, `{"error": "Amount must be positive"}`, http.StatusBadRequest)
		return
	}

	// Get auction info from PostgreSQL to get min_increment
	var minIncrement float64
	var status string
	err := h.db.QueryRow(
		"SELECT min_increment, status FROM auctions WHERE id = $1",
		req.AuctionID,
	).Scan(&minIncrement, &status)

	if err == sql.ErrNoRows {
		http.Error(w, `{"error": "Auction not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, `{"error": "Database error"}`, http.StatusInternalServerError)
		return
	}

	if status != "active" {
		http.Error(w, `{"error": "Auction is not active"}`, http.StatusBadRequest)
		return
	}

	// Check user wallet balance
	var walletBalance float64
	err = h.db.QueryRow("SELECT wallet_balance FROM users WHERE id = $1", userID).Scan(&walletBalance)
	if err != nil {
		http.Error(w, `{"error": "Failed to check wallet balance"}`, http.StatusInternalServerError)
		return
	}
	if walletBalance < req.Amount {
		http.Error(w, `{"error": "Insufficient wallet balance"}`, http.StatusBadRequest)
		return
	}

	// Place bid using Lua script (atomic operation)
	result, err := h.scripts.PlaceBid(ctx, req.AuctionID, req.Amount, minIncrement, userID)
	if err != nil {
		log.Printf("Error placing bid: %v", err)
		http.Error(w, `{"error": "Failed to place bid"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	switch result {
	case 1:
		// Success - save to PostgreSQL for history
		bidID := uuid.New().String()
		_, err := h.db.Exec(
			"INSERT INTO bids (id, auction_id, user_id, amount) VALUES ($1, $2, $3, $4)",
			bidID, req.AuctionID, userID, req.Amount,
		)
		if err != nil {
			log.Printf("Error saving bid to DB: %v", err)
		}

		// Update current_price in auctions table
		h.db.Exec(
			"UPDATE auctions SET current_price = $1, winner_id = $2 WHERE id = $3",
			req.Amount, userID, req.AuctionID,
		)

		// Publish bid event via NATS
		event := models.BidEvent{
			Type:      "bid:new",
			AuctionID: req.AuctionID,
			Amount:    req.Amount,
			BidderID:  userID,
		}
		data, _ := json.Marshal(event)
		h.nc.Publish("auction:bid", data)

		json.NewEncoder(w).Encode(models.BidResponse{
			Success:      true,
			Message:      "Bid placed successfully",
			CurrentPrice: req.Amount,
			BidderID:     userID,
		})

	case 0:
		currentPrice, _, _ := h.scripts.GetCurrentBid(ctx, req.AuctionID)
		json.NewEncoder(w).Encode(models.BidResponse{
			Success:      false,
			Message:      "Bid too low. Must be at least current price + min increment",
			CurrentPrice: currentPrice,
		})

	case -1:
		http.Error(w, `{"error": "Auction not initialized in cache"}`, http.StatusNotFound)
	}
}

func (h *BidHandler) GetCurrentBid(w http.ResponseWriter, r *http.Request) {
	auctionID := chi.URLParam(r, "auctionId")

	price, bidder, err := h.scripts.GetCurrentBid(ctx, auctionID)
	if err != nil {
		http.Error(w, `{"error": "Failed to get current bid"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.CurrentBidResponse{
		AuctionID:    auctionID,
		CurrentPrice: price,
		BidderID:     bidder,
	})
}

// HandleAuctionEvent syncs auction data to Redis when auction starts
func (h *BidHandler) HandleAuctionEvent(data []byte) {
	var event models.AuctionEvent
	if err := json.Unmarshal(data, &event); err != nil {
		log.Printf("Error unmarshaling auction event: %v", err)
		return
	}

	switch event.Type {
	case "auction:start":
		// Initialize auction in Redis
		err := h.scripts.InitAuction(ctx, event.Auction.ID, event.Auction.StartingPrice)
		if err != nil {
			log.Printf("Error initializing auction in Redis: %v", err)
		} else {
			log.Printf("Auction %s initialized in Redis with price %.2f", event.Auction.ID, event.Auction.StartingPrice)
		}
	}
}
