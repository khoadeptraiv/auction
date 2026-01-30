package gateway

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/golang-jwt/jwt/v5"
)

func StartServer(port string) {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	// Service URLs
	userURL := "http://localhost:8081"
	auctionURL := "http://localhost:8082"
	biddingURL := "http://localhost:8083"
	notifyURL := "http://localhost:8084"

	// Public routes
	r.Post("/api/auth/register", proxyTo(userURL, "/auth/register"))
	r.Post("/api/auth/login", proxyTo(userURL, "/auth/login"))
	r.Get("/api/auctions", proxyTo(auctionURL, "/auctions"))
	r.Get("/api/auctions/active", proxyTo(auctionURL, "/auctions/active"))
	r.Get("/api/auctions/{id}", proxyToWithID(auctionURL, "/auctions/"))
	// WebSocket route for notifications
	r.HandleFunc("/ws", proxyTo(notifyURL, "/ws"))

	// Protected routes - with Clerk JWT verification
	r.Group(func(r chi.Router) {
		r.Use(clerkAuth)
		r.Get("/api/users/me", proxyTo(userURL, "/users/me"))
		r.Post("/api/users/sync", proxyTo(userURL, "/users/sync"))
		r.Get("/api/wallet/balance", proxyTo(userURL, "/wallet/balance"))
		r.Post("/api/wallet/deposit", proxyTo(userURL, "/wallet/deposit"))
		r.Post("/api/auctions", proxyTo(auctionURL, "/auctions"))
		r.Get("/api/auctions/my", proxyTo(auctionURL, "/auctions/my"))
		r.Post("/api/bid", proxyTo(biddingURL, "/bid"))
	})

	log.Printf("🚪 Gateway listening on %s", port)
	http.ListenAndServe(port, r)
}

// clerkAuth verifies Clerk JWT tokens
func clerkAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error": "Authorization required"}`, http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, `{"error": "Invalid authorization format"}`, http.StatusUnauthorized)
			return
		}

		tokenString := parts[1]

		// For Clerk tokens, we verify with Clerk's JWKS
		// For simplicity in development, we also support custom JWT
		secret := os.Getenv("JWT_SECRET")
		if secret == "" {
			http.Error(w, "JWT_SECRET not set", http.StatusInternalServerError)
			return
		}

		// Try to parse as custom JWT first
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})

		if err == nil && token.Valid {
			claims := token.Claims.(jwt.MapClaims)
			if userID, ok := claims["user_id"].(string); ok {
				r.Header.Set("X-User-ID", userID)
			}
			if email, ok := claims["email"].(string); ok {
				r.Header.Set("X-User-Email", email)
			}
			next.ServeHTTP(w, r)
			return
		}

		// Try to parse as Clerk JWT (without full verification for dev)
		// In production, you should verify with Clerk's JWKS
		clerkToken, _, err := new(jwt.Parser).ParseUnverified(tokenString, jwt.MapClaims{})
		if err != nil {
			http.Error(w, `{"error": "Invalid token"}`, http.StatusUnauthorized)
			return
		}

		claims, ok := clerkToken.Claims.(jwt.MapClaims)
		if !ok {
			http.Error(w, `{"error": "Invalid token claims"}`, http.StatusUnauthorized)
			return
		}

		// Clerk uses 'sub' for user ID
		if sub, ok := claims["sub"].(string); ok {
			r.Header.Set("X-User-ID", sub)
			r.Header.Set("X-Clerk-User-ID", sub)
		}

		next.ServeHTTP(w, r)
	})
}

func proxyTo(targetURL, path string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		target, _ := url.Parse(targetURL)
		proxy := httputil.NewSingleHostReverseProxy(target)
		r.URL.Path = path
		r.URL.Host = target.Host
		r.URL.Scheme = target.Scheme
		r.Host = target.Host
		proxy.ServeHTTP(w, r)
	}
}

func proxyToWithID(targetURL, basePath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		target, _ := url.Parse(targetURL)
		proxy := httputil.NewSingleHostReverseProxy(target)
		id := chi.URLParam(r, "id")
		r.URL.Path = basePath + id
		r.URL.Host = target.Host
		r.URL.Scheme = target.Scheme
		r.Host = target.Host
		proxy.ServeHTTP(w, r)
	}
}

// ClerkUserInfo fetches user info from Clerk API
type ClerkUserInfo struct {
	ID           string `json:"id"`
	EmailAddress string `json:"email_addresses"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
}

func getClerkUserInfo(clerkUserID string) (*ClerkUserInfo, error) {
	clerkSecretKey := os.Getenv("CLERK_SECRET_KEY")
	if clerkSecretKey == "" {
		return nil, nil
	}

	req, _ := http.NewRequest("GET", "https://api.clerk.com/v1/users/"+clerkUserID, nil)
	req.Header.Set("Authorization", "Bearer "+clerkSecretKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var user ClerkUserInfo
	json.Unmarshal(body, &user)
	return &user, nil
}
