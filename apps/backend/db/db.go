package db

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
)

// ConnectPostgres connects to PostgreSQL database
func ConnectPostgres() (*sql.DB, error) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		// Log warning but don't hardcode sensitive defaults for production code
	}

	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

// Deprecated: Migrated to Postgres
func ConnectMysql() (*sql.DB, error) {
	return nil, fmt.Errorf("MySQL is deprecated, use Postgres")
}

func ConnectRedis() *redis.Client {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "localhost:6379"
	}

	// If REDIS_URL starts with redis:// or rediss://, use ParseURL
	if len(redisURL) > 8 && (redisURL[:8] == "redis://" || redisURL[:9] == "rediss://") {
		opts, err := redis.ParseURL(redisURL)
		if err != nil {
			fmt.Printf("❌ Failed to parse REDIS_URL: %v\n", err)
			// Fallback or exit? For now, try to continue but likely fail
		}
		return redis.NewClient(opts)
	}

	return redis.NewClient(&redis.Options{
		Addr: redisURL,
	})
}

func ConnectNATS() (*nats.Conn, error) {
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsHost := os.Getenv("NATS_HOST")
		natsPort := os.Getenv("NATS_PORT")
		if natsHost != "" {
			if natsPort == "" {
				natsPort = "4222"
			}
			natsURL = fmt.Sprintf("nats://%s:%s", natsHost, natsPort)
		}
	}
	if natsURL == "" {
		natsURL = nats.DefaultURL
	}

	return nats.Connect(natsURL)
}
