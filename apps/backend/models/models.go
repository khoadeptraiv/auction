package models

import "time"

// User models
type User struct {
	ID            string    `json:"id"`
	Email         string    `json:"email"`
	PasswordHash  string    `json:"-"`
	FullName      string    `json:"full_name"`
	WalletBalance float64   `json:"wallet_balance"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type DepositRequest struct {
	Amount float64 `json:"amount"`
}

type WalletResponse struct {
	Balance float64 `json:"balance"`
}

// Auction models
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
	Status        string    `json:"status"`
	WinnerID      *string   `json:"winner_id,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

type CreateAuctionRequest struct {
	Title         string  `json:"title"`
	Description   string  `json:"description"`
	ImageURL      string  `json:"image_url"`
	StartingPrice float64 `json:"starting_price"`
	MinIncrement  float64 `json:"min_increment"`
	StartTime     string  `json:"start_time"`
	EndTime       string  `json:"end_time"`
}

type AuctionListResponse struct {
	Auctions []Auction `json:"auctions"`
	Total    int       `json:"total"`
}

type AuctionEvent struct {
	Type    string  `json:"type"`
	Auction Auction `json:"auction"`
}

// Bid models
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

type BidEvent struct {
	Type      string  `json:"type"`
	AuctionID string  `json:"auction_id"`
	Amount    float64 `json:"amount"`
	BidderID  string  `json:"bidder_id"`
}
