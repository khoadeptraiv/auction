package models

import "time"

type Bid struct {
	ID        string    `json:"id"`
	AuctionID string    `json:"auction_id"`
	UserID    string    `json:"user_id"`
	Amount    float64   `json:"amount"`
	CreatedAt time.Time `json:"created_at"`
}

type PlaceBidRequest struct {
	AuctionID string  `json:"auction_id"`
	Amount    float64 `json:"amount"`
}

type BidResponse struct {
	Success      bool    `json:"success"`
	Message      string  `json:"message"`
	CurrentPrice float64 `json:"current_price"`
	BidderID     string  `json:"bidder_id,omitempty"`
}

type CurrentBidResponse struct {
	AuctionID    string  `json:"auction_id"`
	CurrentPrice float64 `json:"current_price"`
	BidderID     string  `json:"bidder_id,omitempty"`
}

// NATS Events
type BidEvent struct {
	Type      string  `json:"type"` // bid:new
	AuctionID string  `json:"auction_id"`
	Amount    float64 `json:"amount"`
	BidderID  string  `json:"bidder_id"`
}

type AuctionEvent struct {
	Type    string  `json:"type"`
	Auction Auction `json:"auction"`
}

type Auction struct {
	ID            string  `json:"id"`
	StartingPrice float64 `json:"starting_price"`
	CurrentPrice  float64 `json:"current_price"`
	MinIncrement  float64 `json:"min_increment"`
	Status        string  `json:"status"`
}
