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

## API Endpoints

### Authentication

- `POST /api/auth/register` - Register new user
- `POST /api/auth/login` - Login

### Users

- `GET /api/users/me` - Get current user info

### Wallet

- `GET /api/wallet/balance` - Get wallet balance
- `POST /api/wallet/deposit` - Deposit money

### Auctions

- `GET /api/auctions` - List all auctions
- `GET /api/auctions/active` - List active auctions
- `GET /api/auctions/{id}` - Get auction details
- `POST /api/auctions` - Create new auction
- `GET /api/auctions/my` - List my auctions

### Bidding

- `POST /api/bid` - Place a bid

### WebSocket

- `ws://localhost:8084/ws` - Real-time updates

## WebSocket Events

```json
// New bid
{
  "type": "bid:new",
  "auction_id": "uuid",
  "amount": 100000,
  "bidder_id": "uuid"
}

// Auction started
{
  "type": "auction:start",
  "auction": {...}
}

// Auction ended
{
  "type": "auction:end",
  "auction": {...}
}
```

## Environment Variables

| Variable     | Default                                                 | Description           |
| ------------ | ------------------------------------------------------- | --------------------- |
| DATABASE_URL | postgres://auction:auction123@localhost:5432/auction_db | PostgreSQL connection |
| REDIS_URL    | localhost:6379                                          | Redis connection      |
| NATS_URL     | nats://localhost:4222                                   | NATS connection       |
| JWT_SECRET   | your-super-secret-key                                   | JWT signing key       |

## Testing

### Register a user

```bash
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"123456","full_name":"Test User"}'
```

### Login

```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"123456"}'
```

### Create an auction

```bash
curl -X POST http://localhost:8080/api/auctions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "title": "iPhone 15 Pro Max",
    "description": "New, sealed",
    "starting_price": 20000000,
    "min_increment": 100000,
    "start_time": "2024-01-30T10:00:00Z",
    "end_time": "2024-01-30T22:00:00Z"
  }'
```

### Place a bid

```bash
curl -X POST http://localhost:8080/api/bid \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{"auction_id": "AUCTION_ID", "amount": 20100000}'
```
