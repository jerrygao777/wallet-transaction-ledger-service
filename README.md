# Wallet and Transaction Ledger Service

A backend service that manages user wallets for two currencies: Gold Coins (GC) and Sweeps Coins (SC). All balance changes are tracked in an immutable transaction ledger with full atomicity guarantees.

## Features

- **Dual Currency System**: Gold Coins (play money) and Sweeps Coins (redeemable)
- **Immutable Ledger**: All balance changes recorded as transactions
- **Idempotent Purchases**: Duplicate requests safely handled
- **Atomic Operations**: All multi-step operations in database transactions
- **Cursor-based Pagination**: Efficient transaction history queries
- **Clean Architecture**: Separated layers (handlers → services → repositories)

## Tech Stack

- **Language**: Go 1.21+
- **Framework**: Chi (lightweight HTTP router)
- **Database**: PostgreSQL
- **Driver**: lib/pq

## Project Structure

```
wallet-ledger/
├── main.go              # Application entry point
├── models/              # Domain models and types
│   └── models.go
├── repository/          # Database access layer
│   └── repository.go
├── service/             # Business logic layer
│   └── service.go
├── handlers/            # HTTP handlers and routing
│   └── handlers.go
├── migrations/          # Database schema
│   └── 001_init.sql
├── go.mod               # Go module file
├── .env.example         # Example environment configuration
└── README.md
```

## Quick Start

### Prerequisites

- Go 1.21 or higher
- PostgreSQL 12 or higher

### 1. Setup Database

```bash
# Create database
psql -U postgres -c "CREATE DATABASE wallet_ledger;"

# Run migrations
psql -U postgres -d wallet_ledger -f migrations/001_init.sql
```

### 2. Configure Environment

```bash
# Copy example env file
cp .env.example .env

# Edit .env with your database credentials
# DATABASE_URL=postgres://postgres:postgres@localhost:5432/wallet_ledger?sslmode=disable
# PORT=8080
```

### 3. Install Dependencies

```bash
go mod download
```

### 4. Run the Server

```bash
go run main.go
```

The server will start on `http://localhost:8080`

## Run with Docker

This project includes a `Dockerfile` and `docker-compose.yml` so you can run the app and Postgres locally without installing Postgres manually.

Note: Docker Desktop must be installed and running on your machine.

Start everything with:

```bash
docker-compose up --build
```

This will:
- Start a Postgres container and initialize the database using the SQL files in `./migrations` (mounted into the container so Postgres runs them on first start).
- Build the Go application image and run it, exposing port 8080 on the host.

Stop and remove containers and volumes (clean slate):

```bash
docker-compose down -v
```

If you prefer to run only the database for manual testing, you can bring up just the DB:

```bash
docker-compose up -d db
```

Environment variables when running with Docker are set in the `docker-compose.yml`. The app is configured to connect to Postgres at `db:5432`.


## API Endpoints

### Health Check

```bash
GET /health
```

### List Available Packages

```bash
GET /packages
```

**Response:**
```json
[
  {
    "code": "starter_10k",
    "gold_coins": 10000,
    "sweep_coins": 10
  },
  {
    "code": "grinder_50k",
    "gold_coins": 50000,
    "sweep_coins": 50
  },
  {
    "code": "highroller_250k",
    "gold_coins": 250000,
    "sweep_coins": 250
  }
]
```

### Get User and Balances

```bash
GET /users/:id
```

**Example:**
```bash
curl http://localhost:8080/users/1
```

**Response:**
```json
{
  "id": 1,
  "username": "alice",
  "created_at": "2025-11-14T10:00:00Z",
  "gold_balance": 10000,
  "sweeps_balance": 10,
  "total_gc_wagered": 500,
  "total_gc_won": 900,
  "total_sc_wagered": 0,
  "total_sc_won": 0,
  "total_sc_redeemed": 0
}
```

### List User Transactions

```bash
GET /users/:id/transactions?cursor=...&limit=...&type=...&currency=...
```

**Query Parameters:**
- `cursor` (optional): Pagination cursor from previous response
- `limit` (optional): Number of items per page (default: 20, max: 100)
- `type` (optional): Filter by transaction type (`purchase`, `wager_gc`, `win_gc`, `wager_sc`, `win_sc`, `redeem_sc`)
- `currency` (optional): Filter by currency (`GC`, `SC`)

**Example:**
```bash
curl "http://localhost:8080/users/1/transactions?limit=10&currency=GC"
```

**Response:**
```json
{
  "items": [
    {
      "id": 1,
      "user_id": 1,
      "currency": "GC",
      "type": "purchase",
      "amount": 10000,
      "balance_after": 10000,
      "metadata": {"package_code": "starter_10k"},
      "created_at": "2025-11-14T10:00:00Z"
    }
  ],
  "next_cursor": "eyJpZCI6MSwidGltZXN0YW1wIjoxNjk5OTU2MDAwfQ=="
}
```

### Purchase Package

```bash
POST /users/:id/purchase
```

**Body:**
```json
{
  "package_code": "starter_10k",
  "idempotency_key": "unique-key-123"
}
```

**Available Packages:**
- `starter_10k` - 10,000 GC + 10 SC
- `grinder_50k` - 50,000 GC + 50 SC
- `highroller_250k` - 250,000 GC + 250 SC

**Example:**
```bash
curl -X POST http://localhost:8080/users/1/purchase \
  -H "Content-Type: application/json" \
  -d '{"package_code":"starter_10k","idempotency_key":"key-001"}'
```

**Response:**
```json
{
  "id": 1,
  "user_id": 1,
  "currency": "GC",
  "type": "purchase",
  "amount": 10000,
  "balance_after": 10000,
  "created_at": "2025-11-14T10:00:00Z"
}
```

### Simulate Wager

```bash
POST /users/:id/wager
```

**Body:**
```json
{
  "stake_gc": 500,
  "payout_gc": 900
}
```

Or for Sweeps Coins:
```json
{
  "stake_sc": 5,
  "payout_sc": 9
}
```

**Example:**
```bash
curl -X POST http://localhost:8080/users/1/wager \
  -H "Content-Type: application/json" \
  -d '{"stake_gc":500,"payout_gc":900}'
```

**Response:**
```json
{
  "status": "success"
}
```

### Redeem Sweeps Coins

```bash
POST /users/:id/redeem
```

**Body:**
```json
{
  "amount_sc": 10
}
```

**Example:**
```bash
curl -X POST http://localhost:8080/users/1/redeem \
  -H "Content-Type: application/json" \
  -d '{"amount_sc":10}'
```

**Response:**
```json
{
  "status": "success"
}
```

## Complete Test Workflow

Here's a complete example workflow:

```bash
# 1. Check health
curl http://localhost:8080/health

# 2. Check user's initial state
curl http://localhost:8080/users/1

# 3. Purchase a starter package
curl -X POST http://localhost:8080/users/1/purchase \
  -H "Content-Type: application/json" \
  -d '{"package_code":"starter_10k","idempotency_key":"test-001"}'

# 4. Check updated balance
curl http://localhost:8080/users/1

# 5. Place a wager with Gold Coins
curl -X POST http://localhost:8080/users/1/wager \
  -H "Content-Type: application/json" \
  -d '{"stake_gc":500,"payout_gc":900}'

# 6. View transaction history
curl "http://localhost:8080/users/1/transactions?limit=10"

# 7. Purchase with Sweeps Coins
curl -X POST http://localhost:8080/users/1/purchase \
  -H "Content-Type: application/json" \
  -d '{"package_code":"grinder_50k","idempotency_key":"test-002"}'

# 8. Wager with Sweeps Coins
curl -X POST http://localhost:8080/users/1/wager \
  -H "Content-Type: application/json" \
  -d '{"stake_sc":5,"payout_sc":9}'

# 9. Redeem Sweeps Coins
curl -X POST http://localhost:8080/users/1/redeem \
  -H "Content-Type: application/json" \
  -d '{"amount_sc":10}'

# 10. Check final balances
curl http://localhost:8080/users/1
```

## Key Design Decisions

### Ledger-Based Balances
All balances are calculated from the transaction ledger, not stored separately. The `balance_after` field in each transaction provides a running total for auditing.

### Idempotency
Purchase requests use idempotency keys to prevent duplicate charges. The same key returns the original transaction without creating duplicates.

### Atomicity
All balance-changing operations use database transactions with `SELECT FOR UPDATE` to prevent race conditions and ensure consistency.

### Cursor Pagination
Transaction lists use cursor-based pagination (encoded ID + timestamp) for efficient queries on large datasets.

### Immutable Transactions
Once written, transactions cannot be modified. All corrections are new transactions.

## Database Schema

### Users Table
```sql
id          SERIAL PRIMARY KEY
username    VARCHAR(255) UNIQUE
created_at  TIMESTAMP
```

### Transactions Table
```sql
id            SERIAL PRIMARY KEY
user_id       INTEGER REFERENCES users(id)
currency      VARCHAR(2) CHECK (currency IN ('GC', 'SC'))
type          VARCHAR(20) CHECK (type IN (...))
amount        BIGINT
balance_after BIGINT
metadata      JSONB
created_at    TIMESTAMP
```

### Idempotency Keys Table
```sql
key            VARCHAR(255) PRIMARY KEY
user_id        INTEGER REFERENCES users(id)
transaction_id INTEGER REFERENCES transactions(id)
created_at     TIMESTAMP
```

## Error Handling

The API returns appropriate HTTP status codes:

- `200 OK` - Successful request
- `400 Bad Request` - Invalid input or insufficient funds
- `404 Not Found` - User not found
- `500 Internal Server Error` - Server error

Error responses follow this format:
```json
{
  "error": "error message here"
}
```

## Development

### Running Tests
```bash
go test ./...
```

### Building

**Linux/macOS:**
```bash
go build -o wallet-ledger
```

**Windows:**
```bash
go build -o wallet-ledger.exe
```

### Database Cleanup (Dev)
```bash
# Drop and recreate database
psql -U postgres -c "DROP DATABASE wallet_ledger;"
psql -U postgres -c "CREATE DATABASE wallet_ledger;"
psql -U postgres -d wallet_ledger -f migrations/001_init.sql
```

## Architecture Notes

### Layer Responsibilities

**Handlers Layer** (`handlers/`)
- HTTP request/response handling
- Input validation
- Error response formatting
- No business logic

**Service Layer** (`service/`)
- Business logic and rules
- Transaction orchestration
- Multi-step operations
- Validation of business constraints

**Repository Layer** (`repository/`)
- Database queries
- Transaction management
- Data access patterns
- No business logic

**Models** (`models/`)
- Data structures
- Type definitions
- Constants

This separation ensures:
- Easy testing (mock interfaces)
- Clear responsibilities
- Maintainable codebase
- Flexible infrastructure changes

## License

MIT
