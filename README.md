# Auction System

Hệ thống đấu giá real-time với Go microservices backend.

## Architecture

```
┌─────────────┐     ┌─────────────┐
│ Mobile App  │     │Desktop App  │
│   (Expo)    │     │ (Electron)  │
└──────┬──────┘     └──────┬──────┘
       │                   │
       └─────────┬─────────┘
                 │
         ┌───────▼───────┐
         │  API Gateway  │ :8080
         └───────┬───────┘
                 │
    ┌────────────┼────────────┬────────────┐
    │            │            │            │
┌───▼───┐   ┌───▼───┐   ┌───▼───┐   ┌───▼───┐
│ User  │   │Auction│   │Bidding│   │Notify │
│Service│   │Service│   │Service│   │Service│
│ :8081 │   │ :8082 │   │ :8083 │   │ :8084 │
└───┬───┘   └───┬───┘   └───┬───┘   └───┬───┘
    │           │           │           │
    └─────┬─────┘           │           │
          │                 │           │
    ┌─────▼─────┐     ┌─────▼─────┐     │
    │PostgreSQL │     │   Redis   │◄────┘
    └───────────┘     └─────┬─────┘
                            │
                      ┌─────▼─────┐
                      │   NATS    │
                      └───────────┘
```

## Services

| Service         | Port | Database   | Description                            |
| --------------- | ---- | ---------- | -------------------------------------- |
| API Gateway     | 8080 | -          | Routing, authentication, rate limiting |
| User Service    | 8081 | PostgreSQL | User accounts, wallet                  |
| Auction Service | 8082 | PostgreSQL | Products, auctions management          |
| Bidding Service | 8083 | Redis      | Real-time bidding                      |
| Notify Service  | 8084 | Redis      | WebSocket notifications                |

## Quick Start

### Prerequisites

- Docker & Docker Compose
- Go 1.21+ (for local development)

### Run with Docker

```bash
# Build and run all services
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down
```

### Run Locally (Development)

```bash
# Start infrastructure only
docker-compose up -d postgres redis nats

# Initialize go.sum files
make init

# Run each service in separate terminals
make run-gateway
make run-user
make run-auction
make run-bidding
make run-notify
```

## Environment Variables

| Variable     | Default                                                 | Description           |
| ------------ | ------------------------------------------------------- | --------------------- |
| DATABASE_URL | postgres://auction:auction123@localhost:5432/auction_db | PostgreSQL connection |
| REDIS_URL    | localhost:6379                                          | Redis connection      |
| NATS_URL     | nats://localhost:4222                                   | NATS connection       |
| JWT_SECRET   | your-super-secret-key                                   | JWT signing key       |
