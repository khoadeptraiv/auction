package models

import "time"

type Auction struct {
	ID            string    `json:"id"`
	SellerID      string    `json:"seller_id"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	ImageURL      string    `json:"image_url"`
	StartingPrice float64   `json:"starting_price"`
	CurrentPrice  float64   `json:"current_price"`
	MinIncrement  float64   `json:"min_increment"`
	StartTime     time.Time `json:"start_time"`
	EndTime       time.Time `json:"end_time"`
	Status        string    `json:"status"` // pending, active, ended, cancelled
	WinnerID      *string   `json:"winner_id,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

type CreateAuctionRequest struct {
	Title         string  `json:"title"`
	Description   string  `json:"description"`
	ImageURL      string  `json:"image_url"`
	StartingPrice float64 `json:"starting_price"`
	MinIncrement  float64 `json:"min_increment"`
	StartTime     string  `json:"start_time"` // ISO 8601 format
	EndTime       string  `json:"end_time"`   // ISO 8601 format
}

type AuctionListResponse struct {
	Auctions []Auction `json:"auctions"`
	Total    int       `json:"total"`
}

// NATS Events
type AuctionEvent struct {
	Type    string  `json:"type"` // auction:start, auction:end, auction:winner
	Auction Auction `json:"auction"`
}
