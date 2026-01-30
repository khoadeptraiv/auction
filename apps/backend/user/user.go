package user

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"auction-app/models"
)

var db *sql.DB

func StartServer(database *sql.DB, port string) {
	db = database

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	// Traditional auth routes
	r.Post("/auth/register", register)
	r.Post("/auth/login", login)

	// Clerk sync route - creates/updates user from Clerk
	r.Post("/users/sync", syncClerkUser)

	r.Get("/users/me", getMe)
	r.Get("/wallet/balance", getBalance)
	r.Post("/wallet/deposit", deposit)

	log.Printf("👤 User Service listening on %s", port)
	http.ListenAndServe(port, r)
}

type SyncUserRequest struct {
	ClerkID  string `json:"clerk_id"`
	Email    string `json:"email"`
	FullName string `json:"full_name"`
}

func syncClerkUser(w http.ResponseWriter, r *http.Request) {
	var req SyncUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.ClerkID = r.Header.Get("X-Clerk-User-ID")
		req.Email = r.Header.Get("X-User-Email")
	}

	if req.ClerkID == "" {
		req.ClerkID = r.Header.Get("X-User-ID")
	}

	if req.ClerkID == "" {
		http.Error(w, `{"error": "Missing clerk_id"}`, http.StatusBadRequest)
		return
	}

	var user models.User
	err := db.QueryRow(
		`SELECT id, email, full_name, wallet_balance, created_at, updated_at FROM users WHERE id = ?`,
		req.ClerkID,
	).Scan(&user.ID, &user.Email, &user.FullName, &user.WalletBalance, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		fullName := req.FullName
		if fullName == "" {
			fullName = "User"
		}
		email := req.Email
		if email == "" {
			email = req.ClerkID + "@clerk.user"
		}

		_, err = db.Exec(
			`INSERT INTO users (id, email, password_hash, full_name, wallet_balance) 
			 VALUES (?, ?, '', ?, 0)`,
			req.ClerkID, email, fullName,
		)

		if err != nil {
			log.Printf("Error creating user: %v", err)
			http.Error(w, `{"error": "Failed to create user"}`, http.StatusInternalServerError)
			return
		}

		// Fetch back created user
		db.QueryRow(
			`SELECT id, email, full_name, wallet_balance, created_at, updated_at FROM users WHERE id = ?`,
			req.ClerkID,
		).Scan(&user.ID, &user.Email, &user.FullName, &user.WalletBalance, &user.CreatedAt, &user.UpdatedAt)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(user)
		return
	}

	if err != nil {
		http.Error(w, `{"error": "Database error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid request"}`, http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" || req.FullName == "" {
		http.Error(w, `{"error": "All fields required"}`, http.StatusBadRequest)
		return
	}

	var exists bool
	db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE email = ?)", req.Email).Scan(&exists)
	if exists {
		http.Error(w, `{"error": "Email already registered"}`, http.StatusConflict)
		return
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	userID := uuid.New().String()

	_, err := db.Exec(
		`INSERT INTO users (id, email, password_hash, full_name, wallet_balance) 
		 VALUES (?, ?, ?, ?, 0)`,
		userID, req.Email, string(hash), req.FullName,
	)

	if err != nil {
		http.Error(w, `{"error": "Failed to create user"}`, http.StatusInternalServerError)
		return
	}

	// Fetch back
	var user models.User
	db.QueryRow(
		`SELECT id, email, full_name, wallet_balance, created_at, updated_at FROM users WHERE id = ?`,
		userID,
	).Scan(&user.ID, &user.Email, &user.FullName, &user.WalletBalance, &user.CreatedAt, &user.UpdatedAt)

	token := generateToken(user.ID, user.Email)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(models.AuthResponse{Token: token, User: user})
}

func login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid request"}`, http.StatusBadRequest)
		return
	}

	var user models.User
	err := db.QueryRow(
		`SELECT id, email, password_hash, full_name, wallet_balance, created_at, updated_at 
		 FROM users WHERE email = ?`, req.Email,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.FullName, &user.WalletBalance, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		http.Error(w, `{"error": "Invalid credentials"}`, http.StatusUnauthorized)
		return
	}

	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
		http.Error(w, `{"error": "Invalid credentials"}`, http.StatusUnauthorized)
		return
	}

	token := generateToken(user.ID, user.Email)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.AuthResponse{Token: token, User: user})
}

func getMe(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var user models.User
	err := db.QueryRow(
		`SELECT id, email, full_name, wallet_balance, created_at, updated_at FROM users WHERE id = ?`,
		userID,
	).Scan(&user.ID, &user.Email, &user.FullName, &user.WalletBalance, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		http.Error(w, `{"error": "User not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func getBalance(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")

	var exists bool
	db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = ?)", userID).Scan(&exists)
	if !exists {
		db.Exec(
			`INSERT INTO users (id, email, password_hash, full_name, wallet_balance) VALUES (?, ?, '', ?, 0)`,
			userID, userID+"@clerk.user", "User",
		)
	}

	var balance float64
	db.QueryRow("SELECT wallet_balance FROM users WHERE id = ?", userID).Scan(&balance)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.WalletResponse{Balance: balance})
}

func deposit(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	var req models.DepositRequest
	json.NewDecoder(r.Body).Decode(&req)

	if req.Amount <= 0 {
		http.Error(w, `{"error": "Invalid amount"}`, http.StatusBadRequest)
		return
	}

	var exists bool
	db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = ?)", userID).Scan(&exists)
	if !exists {
		db.Exec(
			`INSERT INTO users (id, email, password_hash, full_name, wallet_balance) VALUES (?, ?, '', ?, 0)`,
			userID, userID+"@clerk.user", "User",
		)
	}

	_, err := db.Exec(
		`UPDATE users SET wallet_balance = wallet_balance + ? WHERE id = ?`,
		req.Amount, userID,
	)
	if err != nil {
		http.Error(w, `{"error": "Database error"}`, http.StatusInternalServerError)
		return
	}

	var newBalance float64
	db.QueryRow(
		`SELECT wallet_balance FROM users WHERE id = ?`, userID,
	).Scan(&newBalance)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.WalletResponse{Balance: newBalance})
}

func generateToken(userID, email string) string {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Println("ERROR: JWT_SECRET not set")
		return ""
	}

	claims := jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, _ := token.SignedString([]byte(secret))
	return signed
}
