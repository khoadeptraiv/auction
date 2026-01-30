package db

import (
	"database/sql"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
)

func ConnectMysql() (*sql.DB, error) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		// Log warning but don't hardcode sensitive defaults for production code
		// Fallback only if absolutely necessary for local dev without env
		// databaseURL = "auction:auction123@tcp(localhost:3306)/auction_db?parseTime=true"
	}

	db, err := sql.Open("mysql", databaseURL)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

// Kept for backward compatibility if needed, but redirects to MySQL
func ConnectPostgres() (*sql.DB, error) {
	return ConnectMysql()
}

func ConnectRedis() *redis.Client {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "localhost:6379"
	}

	return redis.NewClient(&redis.Options{
		Addr: redisURL,
	})
}

func ConnectNATS() (*nats.Conn, error) {
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = nats.DefaultURL
	}

	return nats.Connect(natsURL)
}
