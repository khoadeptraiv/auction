package auction

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"

	"auction-app/models"
)

var db *sql.DB
var nc *nats.Conn

func StartServer(database *sql.DB, natsConn *nats.Conn, port string) {
	db = database
	nc = natsConn

	// Start scheduler
	go startScheduler()

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	r.Get("/auctions", listAuctions)
	r.Get("/auctions/active", listActiveAuctions)
	r.Get("/auctions/{id}", getAuction)
	r.Post("/auctions", createAuction)
	r.Get("/auctions/my", listMyAuctions)

	log.Printf("🏷️  Auction Service listening on %s", port)
	http.ListenAndServe(port, r)
}

func createAuction(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var req models.CreateAuctionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid request"}`, http.StatusBadRequest)
		return
	}

	startTime, _ := time.Parse(time.RFC3339, req.StartTime)
	endTime, _ := time.Parse(time.RFC3339, req.EndTime)

	minIncrement := req.MinIncrement
	if minIncrement <= 0 {
		minIncrement = 10000
	}

	status := "pending"
	if time.Now().After(startTime) && time.Now().Before(endTime) {
		status = "active"
	}

	auctionID := uuid.New().String()
	_, err := db.Exec(
		`INSERT INTO auctions (id, seller_id, title, description, image_url, starting_price, current_price, min_increment, start_time, end_time, status)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		auctionID, userID, req.Title, req.Description, req.ImageURL, req.StartingPrice, req.StartingPrice, minIncrement, startTime, endTime, status,
	)

	if err != nil {
		log.Printf("Error creating auction: %v", err)
		http.Error(w, `{"error": "Failed to create auction"}`, http.StatusInternalServerError)
		return
	}

	// Fetch back
	var a models.Auction
	db.QueryRow(
		`SELECT id, seller_id, title, description, image_url, starting_price, current_price, min_increment, start_time, end_time, status, created_at 
		 FROM auctions WHERE id = ?`,
		auctionID,
	).Scan(&a.ID, &a.SellerID, &a.Title, &a.Description, &a.ImageURL, &a.StartingPrice, &a.CurrentPrice, &a.MinIncrement, &a.StartTime, &a.EndTime, &a.Status, &a.CreatedAt)

	if status == "active" {
		publishEvent("auction:start", a)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(a)
}

func listAuctions(w http.ResponseWriter, r *http.Request) {
	rows, _ := db.Query(
		`SELECT id, seller_id, title, description, image_url, starting_price, current_price, min_increment, start_time, end_time, status, winner_id, created_at 
		 FROM auctions ORDER BY created_at DESC LIMIT 50`,
	)
	if rows != nil {
		defer rows.Close()
	}

	auctions := scanAuctions(rows)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.AuctionListResponse{Auctions: auctions, Total: len(auctions)})
}

func listActiveAuctions(w http.ResponseWriter, r *http.Request) {
	rows, _ := db.Query(
		`SELECT id, seller_id, title, description, image_url, starting_price, current_price, min_increment, start_time, end_time, status, winner_id, created_at 
		 FROM auctions WHERE status = 'active' ORDER BY end_time ASC`,
	)
	if rows != nil {
		defer rows.Close()
	}

	auctions := scanAuctions(rows)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.AuctionListResponse{Auctions: auctions, Total: len(auctions)})
}

func listMyAuctions(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	rows, _ := db.Query(
		`SELECT id, seller_id, title, description, image_url, starting_price, current_price, min_increment, start_time, end_time, status, winner_id, created_at 
		 FROM auctions WHERE seller_id = ? ORDER BY created_at DESC`, userID,
	)
	if rows != nil {
		defer rows.Close()
	}

	auctions := scanAuctions(rows)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.AuctionListResponse{Auctions: auctions, Total: len(auctions)})
}

func getAuction(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var a models.Auction
	err := db.QueryRow(
		`SELECT id, seller_id, title, description, image_url, starting_price, current_price, min_increment, start_time, end_time, status, winner_id, created_at 
		 FROM auctions WHERE id = ?`, id,
	).Scan(&a.ID, &a.SellerID, &a.Title, &a.Description, &a.ImageURL, &a.StartingPrice, &a.CurrentPrice, &a.MinIncrement, &a.StartTime, &a.EndTime, &a.Status, &a.WinnerID, &a.CreatedAt)

	if err == sql.ErrNoRows {
		http.Error(w, `{"error": "Auction not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(a)
}

func scanAuctions(rows *sql.Rows) []models.Auction {
	auctions := []models.Auction{}
	if rows == nil {
		return auctions
	}
	for rows.Next() {
		var a models.Auction
		rows.Scan(&a.ID, &a.SellerID, &a.Title, &a.Description, &a.ImageURL, &a.StartingPrice, &a.CurrentPrice, &a.MinIncrement, &a.StartTime, &a.EndTime, &a.Status, &a.WinnerID, &a.CreatedAt)
		auctions = append(auctions, a)
	}
	return auctions
}

func startScheduler() {
	ticker := time.NewTicker(10 * time.Second)
	for range ticker.C {
		// 1. Find pending auctions to start
		rows, _ := db.Query(
			`SELECT id, seller_id, title, description, image_url, starting_price, current_price, min_increment, start_time, end_time, status, created_at
			 FROM auctions WHERE status = 'pending' AND start_time <= NOW() AND end_time > NOW()`,
		)

		var toStart []models.Auction
		if rows != nil {
			for rows.Next() {
				var a models.Auction
				rows.Scan(&a.ID, &a.SellerID, &a.Title, &a.Description, &a.ImageURL, &a.StartingPrice, &a.CurrentPrice, &a.MinIncrement, &a.StartTime, &a.EndTime, &a.Status, &a.CreatedAt)
				toStart = append(toStart, a)
			}
			rows.Close()
		}

		// Update and publish
		for _, a := range toStart {
			_, err := db.Exec("UPDATE auctions SET status = 'active' WHERE id = ?", a.ID)
			if err == nil {
				a.Status = "active"
				log.Printf("Auction started: %s", a.ID)
				publishEvent("auction:start", a)
			}
		}

		// 2. Find active auctions to end
		rows, _ = db.Query(
			`SELECT id, seller_id, title, description, image_url, starting_price, current_price, min_increment, start_time, end_time, status, winner_id, created_at
			 FROM auctions WHERE status = 'active' AND end_time <= NOW()`,
		)

		var toEnd []models.Auction
		if rows != nil {
			for rows.Next() {
				var a models.Auction
				rows.Scan(&a.ID, &a.SellerID, &a.Title, &a.Description, &a.ImageURL, &a.StartingPrice, &a.CurrentPrice, &a.MinIncrement, &a.StartTime, &a.EndTime, &a.Status, &a.WinnerID, &a.CreatedAt)
				toEnd = append(toEnd, a)
			}
			rows.Close()
		}

		// Update and publish
		for _, a := range toEnd {
			_, err := db.Exec("UPDATE auctions SET status = 'ended' WHERE id = ?", a.ID)
			if err == nil {
				a.Status = "ended"
				log.Printf("Auction ended: %s", a.ID)
				publishEvent("auction:end", a)
			}
		}
	}
}

func publishEvent(eventType string, auction models.Auction) {
	event := models.AuctionEvent{Type: eventType, Auction: auction}
	data, _ := json.Marshal(event)
	nc.Publish("auction.events", data)
}
