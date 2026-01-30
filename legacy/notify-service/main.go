package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/nats-io/nats.go"

	"auction/notify-service/handlers"
	"auction/notify-service/hub"
)

func main() {
	// Initialize hub
	h := hub.NewHub()
	go h.Run()

	// Initialize NATS
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = nats.DefaultURL
	}
	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatal("Failed to connect to NATS:", err)
	}
	defer nc.Close()

	// Subscribe to bid events
	nc.Subscribe("auction:bid", func(m *nats.Msg) {
		log.Printf("Received bid event, broadcasting to %d clients", h.ClientCount())
		h.Broadcast(m.Data)
	})

	// Subscribe to auction events
	nc.Subscribe("auction.events", func(m *nats.Msg) {
		log.Printf("Received auction event, broadcasting to %d clients", h.ClientCount())
		h.Broadcast(m.Data)
	})

	// Initialize handlers
	wsHandler := handlers.NewWebSocketHandler(h)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	// WebSocket endpoint
	r.Get("/ws", wsHandler.HandleWebSocket)

	// Stats endpoint
	r.Get("/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"connected_clients": ` + string(rune(h.ClientCount())) + `}`))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8084"
	}

	log.Printf("Notify Service starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
