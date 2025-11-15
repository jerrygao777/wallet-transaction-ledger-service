# Wallet and Transaction Ledger Service

A backend service that manages user wallets for two currencies: Gold Coins (GC) and Sweeps Coins (SC). All balance changes are tracked in an immutable transaction ledger with full atomicity guarantees.

## Features

- **Dual Currency System**: Gold Coins (play money) and Sweeps Coins (redeemable)
- **Immutable Ledger**: All balance changes recorded as transactions
- **Idempotency Protection**: All financial operations (purchase, wager, redeem) prevent duplicates
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
  "idempotency_key": "purchase-001"
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
  -d '{"package_code":"starter_10k","idempotency_key":"purchase-001"}'
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

**Body (all fields optional, at least one must be > 0):**
```json
{
  "stake_gc": 500,
  "payout_gc": 900,
  "idempotency_key": "wager-001"
}
```

Or for Sweeps Coins:
```json
{
  "stake_sc": 5,
  "payout_sc": 9,
  "idempotency_key": "wager-002"
}
```

Or payout only:
```json
{
  "payout_gc": 100,
  "idempotency_key": "wager-003"
}
```

Or any combination of the four fields. The endpoint supports maximum flexibility:
- Stake only (user places bet)
- Payout only (user receives winnings)
- Both stake and payout (complete round)
- Mixed currencies

**Note**: `idempotency_key` is required to prevent duplicate wagers.

**Example:**
```bash
curl -X POST http://localhost:8080/users/1/wager \
  -H "Content-Type: application/json" \
  -d '{"stake_gc":500,"payout_gc":900,"idempotency_key":"wager-001"}'
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
  "amount_sc": 10,
  "idempotency_key": "redeem-001"
}
```

**Note**: `idempotency_key` is required to prevent duplicate redemptions.

**Example:**
```bash
curl -X POST http://localhost:8080/users/1/redeem \
  -H "Content-Type: application/json" \
  -d '{"amount_sc":10,"idempotency_key":"redeem-001"}'
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
  -d '{"package_code":"starter_10k","idempotency_key":"purchase-001"}'

# 4. Check updated balance
curl http://localhost:8080/users/1

# 5. Place a wager with Gold Coins
curl -X POST http://localhost:8080/users/1/wager \
  -H "Content-Type: application/json" \
  -d '{"stake_gc":500,"payout_gc":900,"idempotency_key":"wager-001"}'

# 6. View transaction history
curl "http://localhost:8080/users/1/transactions?limit=10"

# 7. Purchase with Sweeps Coins
curl -X POST http://localhost:8080/users/1/purchase \
  -H "Content-Type: application/json" \
  -d '{"package_code":"grinder_50k","idempotency_key":"purchase-002"}'

# 8. Wager with Sweeps Coins
curl -X POST http://localhost:8080/users/1/wager \
  -H "Content-Type: application/json" \
  -d '{"stake_sc":5,"payout_sc":9,"idempotency_key":"wager-002"}'

# 9. Redeem Sweeps Coins
curl -X POST http://localhost:8080/users/1/redeem \
  -H "Content-Type: application/json" \
  -d '{"amount_sc":10,"idempotency_key":"redeem-001"}'

# 10. Check final balances
curl http://localhost:8080/users/1
```

## Key Design Decisions

### Ledger-Based Balances
All balances are calculated from the transaction ledger, not stored separately. The `balance_after` field in each transaction provides a running total for auditing.

### Idempotency
All financial operations (purchase, wager, redeem) require idempotency keys to prevent duplicate transactions. Submitting the same key returns success without creating duplicates. Keys are automatically cleaned up after 24 hours.

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

## Testing the API

There are three ways to test the API endpoints. Choose the method that works best for you:

### Option 1: VS Code REST Client (Recommended for Development)

The easiest way to test if you're using VS Code:

1. Install the **REST Client** extension in VS Code
2. Open `api-requests.http` in the project
3. Click "Send Request" above any HTTP request
4. View responses directly in VS Code

**Pros:**
- No additional tools needed
- Fast and integrated with your editor
- All requests pre-configured
- Easy to modify and test

### Option 2: Postman (Best for GUI Users)

For those who prefer a graphical interface:

1. Download and install [Postman](https://www.postman.com/downloads/)
2. Import `postman-collection.json` from the project
3. All 17 API endpoints will be loaded
4. Click any request and hit "Send"
5. Use Collection Runner to execute all tests sequentially

**Pros:**
- User-friendly GUI
- Great visualization of responses
- Can save and organize requests
- Supports automated testing

### Option 3: curl (Best for Command Line)

For terminal/command line testing:

**Linux/Mac/Git Bash:**
```bash
# Use the provided script
bash curl-examples.sh

# Or run individual commands
curl http://localhost:8080/health
```

**Windows PowerShell:**
```powershell
# Test health endpoint
Invoke-RestMethod -Uri "http://localhost:8080/health"

# Purchase package
$body = @{ package_code = "starter_10k"; idempotency_key = "test-001" } | ConvertTo-Json
Invoke-RestMethod -Uri "http://localhost:8080/users/1/purchase" -Method Post -Body $body -ContentType "application/json"
```

**Pros:**
- No installation needed (built into most systems)
- Easy to script and automate
- Works on any platform
- Can be used in CI/CD pipelines

## Additional Features

- **Graceful Shutdown**: Handles SIGINT/SIGTERM signals properly
- **Background Cleanup**: Automatically cleans up old idempotency keys (24+ hours old) every hour
- **Connection Pooling**: Configured for optimal database performance
- **Input Validation**: Currency, transaction type, and limit parameters are validated
- **Logging**: Request logging middleware for debugging and monitoring

## Production Considerations

For production deployment, consider:
- Add authentication/authorization
- Implement rate limiting
- Add monitoring/metrics (Prometheus, Datadog, etc.)
- Set up structured logging (zerolog, zap)
- Configure TLS/HTTPS
- Add health checks for dependencies
- Implement circuit breakers for external services
- Set up automated backups for PostgreSQL
- Add comprehensive unit and integration tests
- Implement API versioning

## License

MIT
