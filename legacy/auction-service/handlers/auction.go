package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"

	"auction/auction-service/models"
)

type AuctionHandler struct {
	db *sql.DB
	nc *nats.Conn
}

func NewAuctionHandler(db *sql.DB, nc *nats.Conn) *AuctionHandler {
	return &AuctionHandler{db: db, nc: nc}
}

func (h *AuctionHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, `{"error": "User not authenticated"}`, http.StatusUnauthorized)
		return
	}

	var req models.CreateAuctionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Validate
	if req.Title == "" {
		http.Error(w, `{"error": "Title is required"}`, http.StatusBadRequest)
		return
	}
	if req.StartingPrice <= 0 {
		http.Error(w, `{"error": "Starting price must be positive"}`, http.StatusBadRequest)
		return
	}

	// Parse times
	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		http.Error(w, `{"error": "Invalid start_time format (use ISO 8601)"}`, http.StatusBadRequest)
		return
	}
	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		http.Error(w, `{"error": "Invalid end_time format (use ISO 8601)"}`, http.StatusBadRequest)
		return
	}

	if endTime.Before(startTime) {
		http.Error(w, `{"error": "end_time must be after start_time"}`, http.StatusBadRequest)
		return
	}

	// Set defaults
	minIncrement := req.MinIncrement
	if minIncrement <= 0 {
		minIncrement = 10000 // Default 10,000 VND
	}

	// Determine initial status
	status := "pending"
	if time.Now().After(startTime) && time.Now().Before(endTime) {
		status = "active"
	}

	// Create auction
	auctionID := uuid.New().String()
	var auction models.Auction
	err = h.db.QueryRow(
		`INSERT INTO auctions (id, seller_id, title, description, image_url, starting_price, current_price, min_increment, start_time, end_time, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $6, $7, $8, $9, $10)
		 RETURNING id, seller_id, title, description, image_url, starting_price, current_price, min_increment, start_time, end_time, status, created_at`,
		auctionID, userID, req.Title, req.Description, req.ImageURL, req.StartingPrice, minIncrement, startTime, endTime, status,
	).Scan(&auction.ID, &auction.SellerID, &auction.Title, &auction.Description, &auction.ImageURL, &auction.StartingPrice, &auction.CurrentPrice, &auction.MinIncrement, &auction.StartTime, &auction.EndTime, &auction.Status, &auction.CreatedAt)

	if err != nil {
		log.Printf("Error creating auction: %v", err)
		http.Error(w, `{"error": "Failed to create auction"}`, http.StatusInternalServerError)
		return
	}

	// Publish event if auction is active
	if status == "active" {
		h.publishEvent("auction:start", auction)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(auction)
}

func (h *AuctionHandler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(
		`SELECT id, seller_id, title, description, image_url, starting_price, current_price, min_increment, start_time, end_time, status, winner_id, created_at 
		 FROM auctions ORDER BY created_at DESC LIMIT 50`,
	)
	if err != nil {
		http.Error(w, `{"error": "Database error"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	auctions := []models.Auction{}
	for rows.Next() {
		var a models.Auction
		err := rows.Scan(&a.ID, &a.SellerID, &a.Title, &a.Description, &a.ImageURL, &a.StartingPrice, &a.CurrentPrice, &a.MinIncrement, &a.StartTime, &a.EndTime, &a.Status, &a.WinnerID, &a.CreatedAt)
		if err != nil {
			continue
		}
		auctions = append(auctions, a)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.AuctionListResponse{
		Auctions: auctions,
		Total:    len(auctions),
	})
}

func (h *AuctionHandler) ListActive(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(
		`SELECT id, seller_id, title, description, image_url, starting_price, current_price, min_increment, start_time, end_time, status, winner_id, created_at 
		 FROM auctions WHERE status = 'active' ORDER BY end_time ASC`,
	)
	if err != nil {
		http.Error(w, `{"error": "Database error"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	auctions := []models.Auction{}
	for rows.Next() {
		var a models.Auction
		err := rows.Scan(&a.ID, &a.SellerID, &a.Title, &a.Description, &a.ImageURL, &a.StartingPrice, &a.CurrentPrice, &a.MinIncrement, &a.StartTime, &a.EndTime, &a.Status, &a.WinnerID, &a.CreatedAt)
		if err != nil {
			continue
		}
		auctions = append(auctions, a)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.AuctionListResponse{
		Auctions: auctions,
		Total:    len(auctions),
	})
}

func (h *AuctionHandler) ListMy(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, `{"error": "User not authenticated"}`, http.StatusUnauthorized)
		return
	}

	rows, err := h.db.Query(
		`SELECT id, seller_id, title, description, image_url, starting_price, current_price, min_increment, start_time, end_time, status, winner_id, created_at 
		 FROM auctions WHERE seller_id = $1 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		http.Error(w, `{"error": "Database error"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	auctions := []models.Auction{}
	for rows.Next() {
		var a models.Auction
		err := rows.Scan(&a.ID, &a.SellerID, &a.Title, &a.Description, &a.ImageURL, &a.StartingPrice, &a.CurrentPrice, &a.MinIncrement, &a.StartTime, &a.EndTime, &a.Status, &a.WinnerID, &a.CreatedAt)
		if err != nil {
			continue
		}
		auctions = append(auctions, a)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.AuctionListResponse{
		Auctions: auctions,
		Total:    len(auctions),
	})
}

func (h *AuctionHandler) Get(w http.ResponseWriter, r *http.Request) {
	auctionID := chi.URLParam(r, "id")

	var a models.Auction
	err := h.db.QueryRow(
		`SELECT id, seller_id, title, description, image_url, starting_price, current_price, min_increment, start_time, end_time, status, winner_id, created_at 
		 FROM auctions WHERE id = $1`,
		auctionID,
	).Scan(&a.ID, &a.SellerID, &a.Title, &a.Description, &a.ImageURL, &a.StartingPrice, &a.CurrentPrice, &a.MinIncrement, &a.StartTime, &a.EndTime, &a.Status, &a.WinnerID, &a.CreatedAt)

	if err == sql.ErrNoRows {
		http.Error(w, `{"error": "Auction not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, `{"error": "Database error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(a)
}

// StartScheduler runs a background job to start/end auctions
func (h *AuctionHandler) StartScheduler() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		h.checkPendingAuctions()
		h.checkEndingAuctions()
	}
}

func (h *AuctionHandler) checkPendingAuctions() {
	rows, err := h.db.Query(
		`UPDATE auctions SET status = 'active' 
		 WHERE status = 'pending' AND start_time <= NOW() AND end_time > NOW()
		 RETURNING id, seller_id, title, description, image_url, starting_price, current_price, min_increment, start_time, end_time, status, created_at`,
	)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var a models.Auction
		if err := rows.Scan(&a.ID, &a.SellerID, &a.Title, &a.Description, &a.ImageURL, &a.StartingPrice, &a.CurrentPrice, &a.MinIncrement, &a.StartTime, &a.EndTime, &a.Status, &a.CreatedAt); err == nil {
			log.Printf("Auction started: %s", a.ID)
			h.publishEvent("auction:start", a)
		}
	}
}

func (h *AuctionHandler) checkEndingAuctions() {
	rows, err := h.db.Query(
		`UPDATE auctions SET status = 'ended' 
		 WHERE status = 'active' AND end_time <= NOW()
		 RETURNING id, seller_id, title, description, image_url, starting_price, current_price, min_increment, start_time, end_time, status, winner_id, created_at`,
	)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var a models.Auction
		if err := rows.Scan(&a.ID, &a.SellerID, &a.Title, &a.Description, &a.ImageURL, &a.StartingPrice, &a.CurrentPrice, &a.MinIncrement, &a.StartTime, &a.EndTime, &a.Status, &a.WinnerID, &a.CreatedAt); err == nil {
			log.Printf("Auction ended: %s", a.ID)
			h.publishEvent("auction:end", a)

			if a.WinnerID != nil {
				h.publishEvent("auction:winner", a)
			}
		}
	}
}

func (h *AuctionHandler) publishEvent(eventType string, auction models.Auction) {
	event := models.AuctionEvent{
		Type:    eventType,
		Auction: auction,
	}
	data, _ := json.Marshal(event)
	h.nc.Publish("auction.events", data)
}
