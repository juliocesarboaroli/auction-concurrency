# Auction Concurrency — Go Expert

Auction system with automatic closing using Goroutines.

## About the Challenge

This project implements **automatic auction closing**: as soon as an auction is created, a Goroutine is launched in the background that waits for the time configured in `AUCTION_DURATION` and, once expired, updates the auction status to `Completed` directly in the database — no manual intervention required.

## Prerequisites

- Docker
- Docker Compose

## How to Run

```bash
# Clone the repository
git clone https://github.com/juliocesarboaroli/auction-concurrency.git
cd auction-concurrency

# Start the application (API + MongoDB)
docker compose up --build
```

The API will be available at `http://localhost:8080`.

## Environment Variables

Configured in `cmd/auction/.env`:

| Variable                | Description                                                                 | Default |
|-------------------------|-----------------------------------------------------------------------------|---------|
| `AUCTION_DURATION`      | Time until the auction is automatically closed (e.g. `20s`, `1m`, `2h`)    | `5m`    |
| `AUCTION_INTERVAL`      | Interval used by the bid routine to validate whether the auction is active  | `20s`   |
| `BATCH_INSERT_INTERVAL` | Maximum interval for the bid batch insert                                   | `20s`   |
| `MAX_BATCH_SIZE`        | Number of bids to force an immediate batch flush                            | `4`     |
| `MONGODB_URL`           | MongoDB connection URL                                                      | —       |
| `MONGODB_DB`            | Database name                                                               | `auctions` |

> **Tip for quick testing:** set `AUCTION_DURATION=10s` to see the automatic closing happen in 10 seconds.

## Running the Tests

```bash
go test ./internal/infra/database/auction/... -v
```

### What the Tests Cover

- **Automatic closing:** creates an auction with a 100ms duration and uses a channel with select/timeout to confirm the Goroutine triggered the closing. Fails if the auction is not closed within 2 seconds.
- **Insert failure:** ensures the closing Goroutine is **not** started if the database insert fails.

## API Endpoints

| Method | Route                          | Description                              |
|--------|--------------------------------|------------------------------------------|
| POST   | `/auction`                     | Create an auction                        |
| GET    | `/auction?status=`             | List auctions (`0` = active, `1` = closed) |
| GET    | `/auction/:auctionId`          | Find auction by ID                       |
| GET    | `/auction/winner/:auctionId`   | Winning bid for the auction              |
| POST   | `/bid`                         | Place a bid                              |
| GET    | `/bid/:auctionId`              | List bids for an auction                 |
| GET    | `/user/:userId`                | Find user by ID                          |

## Full Flow Example

```bash
# 1. Create an auction
curl -X POST http://localhost:8080/auction \
  -H "Content-Type: application/json" \
  -d '{"product_name":"iPhone 15","category":"Electronics","description":"iPhone 15 in great condition, lightly used.","condition":1}'

# 2. List auctions to get the ID
curl "http://localhost:8080/auction?status=0"

# 3. Place a bid
curl -X POST http://localhost:8080/bid \
  -H "Content-Type: application/json" \
  -d '{"user_id":"550e8400-e29b-41d4-a716-446655440000","auction_id":"<ID>","amount":1500.00}'

# 4. Wait for AUCTION_DURATION to expire...

# 5. Confirm the auction was closed automatically
curl "http://localhost:8080/auction?status=1"

# 6. Check the winning bid
curl "http://localhost:8080/auction/winner/<ID>"
```