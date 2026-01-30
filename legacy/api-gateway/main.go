package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	customMiddleware "auction/api-gateway/middleware"
)

func main() {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	// Service URLs
	userServiceURL := getEnv("USER_SERVICE_URL", "http://localhost:8081")
	auctionServiceURL := getEnv("AUCTION_SERVICE_URL", "http://localhost:8082")
	biddingServiceURL := getEnv("BIDDING_SERVICE_URL", "http://localhost:8083")

	// Public routes (no auth required)
	r.Group(func(r chi.Router) {
		r.Post("/api/auth/register", proxyTo(userServiceURL, "/auth/register"))
		r.Post("/api/auth/login", proxyTo(userServiceURL, "/auth/login"))
		r.Get("/api/auctions", proxyTo(auctionServiceURL, "/auctions"))
		r.Get("/api/auctions/{id}", proxyToWithID(auctionServiceURL, "/auctions/"))
	})

	// Protected routes (auth required)
	r.Group(func(r chi.Router) {
		r.Use(customMiddleware.JWTAuth)

		// User routes
		r.Get("/api/users/me", proxyTo(userServiceURL, "/users/me"))
		r.Get("/api/wallet/balance", proxyTo(userServiceURL, "/wallet/balance"))
		r.Post("/api/wallet/deposit", proxyTo(userServiceURL, "/wallet/deposit"))

		// Auction routes
		r.Post("/api/auctions", proxyTo(auctionServiceURL, "/auctions"))
		r.Get("/api/auctions/my", proxyTo(auctionServiceURL, "/auctions/my"))

		// Bidding routes
		r.Post("/api/bid", proxyTo(biddingServiceURL, "/bid"))
	})

	port := getEnv("PORT", "8080")
	log.Printf("API Gateway starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func proxyTo(targetURL, path string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		target, _ := url.Parse(targetURL)
		proxy := httputil.NewSingleHostReverseProxy(target)

		r.URL.Path = path
		r.URL.Host = target.Host
		r.URL.Scheme = target.Scheme
		r.Header.Set("X-Forwarded-Host", r.Header.Get("Host"))
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
		r.Header.Set("X-Forwarded-Host", r.Header.Get("Host"))
		r.Host = target.Host

		proxy.ServeHTTP(w, r)
	}
}
