package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"auction/user-service/models"
)

type WalletHandler struct {
	db *sql.DB
}

func NewWalletHandler(db *sql.DB) *WalletHandler {
	return &WalletHandler{db: db}
}

func (h *WalletHandler) GetBalance(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, `{"error": "User not authenticated"}`, http.StatusUnauthorized)
		return
	}

	var balance float64
	err := h.db.QueryRow("SELECT wallet_balance FROM users WHERE id = $1", userID).Scan(&balance)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error": "User not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, `{"error": "Database error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.WalletResponse{Balance: balance})
}

func (h *WalletHandler) Deposit(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, `{"error": "User not authenticated"}`, http.StatusUnauthorized)
		return
	}

	var req models.DepositRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Amount <= 0 {
		http.Error(w, `{"error": "Amount must be positive"}`, http.StatusBadRequest)
		return
	}

	var newBalance float64
	err := h.db.QueryRow(
		`UPDATE users SET wallet_balance = wallet_balance + $1, updated_at = CURRENT_TIMESTAMP 
		 WHERE id = $2 RETURNING wallet_balance`,
		req.Amount, userID,
	).Scan(&newBalance)

	if err == sql.ErrNoRows {
		http.Error(w, `{"error": "User not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, `{"error": "Database error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.WalletResponse{Balance: newBalance})
}
