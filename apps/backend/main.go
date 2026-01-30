package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/nats-io/nats.go"

	"auction-app/auction"
	"auction-app/bidding"
	"auction-app/db"
	"auction-app/gateway"
	"auction-app/notify"
	"auction-app/user"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  No .env file found")
	}

	log.Println("🚀 Starting Auction App...")

	// Initialize shared database
	// Updated to use MySQL
	database, err := db.ConnectMysql()
	if err != nil {
		log.Fatal("❌ Failed to connect to MySQL:", err)
	}
	defer database.Close()
	log.Println("✅ Connected to MySQL")

	// Initialize Redis
	rdb := db.ConnectRedis()
	log.Println("✅ Connected to Redis")

	// Initialize NATS
	nc, err := db.ConnectNATS()
	if err != nil {
		log.Fatal("❌ Failed to connect to NATS:", err)
	}
	defer nc.Close()
	log.Println("✅ Connected to NATS")

	// Start Notify Service (WebSocket hub)
	notifyHub := notify.NewHub()
	go notifyHub.Run()

	// Subscribe NATS to WebSocket hub
	nc.Subscribe("auction:bid", func(m *nats.Msg) {
		notifyHub.Broadcast(m.Data)
	})
	nc.Subscribe("auction.events", func(m *nats.Msg) {
		notifyHub.Broadcast(m.Data)
	})

	// Start all services as goroutines
	go user.StartServer(database, ":8081")
	go auction.StartServer(database, nc, ":8082")
	go bidding.StartServer(database, rdb, nc, ":8083")
	go notify.StartServer(notifyHub, ":8084")
	go gateway.StartServer(":8080")

	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Println("🎉 All services started successfully!")
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Println("📡 API Gateway:     http://localhost:8080")
	log.Println("👤 User Service:    http://localhost:8081")
	log.Println("🏷️  Auction Service: http://localhost:8082")
	log.Println("💰 Bidding Service: http://localhost:8083")
	log.Println("🔔 Notify Service:  ws://localhost:8084/ws")
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("👋 Shutting down...")
}
