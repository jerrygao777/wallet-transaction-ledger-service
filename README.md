# Wallet and Transaction Ledger Service

A production-ready backend service that manages user wallets for two currencies: Gold Coins (GC) and Sweeps Coins (SC). All balance changes are tracked in an immutable transaction ledger with full atomicity guarantees.

## Tech Stack

- **Language**: Go 1.21+
- **Framework**: Chi (lightweight HTTP router)
- **Database**: PostgreSQL 15
- **Architecture**: Clean layered architecture (handlers → services → repositories)

## Key Features

- **Dual Currency System**: Gold Coins (play money) and Sweeps Coins (redeemable)
- **Sweepstakes Casino Compliance**: All purchases require Gold Coins; Sweep Coins awarded as bonus
- **Immutable Transaction Ledger**: Complete audit trail for all balance changes
- **Transaction-Based Architecture**: All balances and statistics calculated from the ledger
- **Idempotency Protection**: All financial operations prevent duplicates via idempotency keys
- **Atomic Operations**: All multi-step operations wrapped in database transactions
- **Cursor-based Pagination**: Efficient transaction history queries
- **Type-safe Error Handling**: Custom error types with proper error wrapping

## 🚀 Quick Start

### Prerequisites

**All you need:**
- [Docker Desktop](https://www.docker.com/products/docker-desktop/) installed and running

No Go, PostgreSQL, or other dependencies required - everything runs in containers.

### Launch the Application

```bash
# Clone the repository
git clone https://github.com/jerrygao777/wallet-transaction-ledger-service.git
cd wallet-transaction-ledger-service

# Start the application
docker-compose up --build
```

The application will:
- Start PostgreSQL database
- Run migrations and seed test data (users: alice, bob, charlie)
- Compile and launch the Go API server
- Be available at **http://localhost:8080**

### Stop the Application

```bash
# Stop and remove all containers/volumes
docker-compose down -v
```


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
[
  {
    "id": 1,
    "user_id": 1,
    "currency": "GC",
    "type": "purchase",
    "amount": 10000,
    "balance_after": 10000,
    "metadata": {"package_code": "starter_10k"},
    "created_at": "2025-11-14T10:00:00Z"
  },
  {
    "id": 2,
    "user_id": 1,
    "currency": "SC",
    "type": "purchase",
    "amount": 10,
    "balance_after": 10,
    "metadata": {"package_code": "starter_10k"},
    "created_at": "2025-11-14T10:00:00Z"
  }
]

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

**Supported Scenarios:**
- Stake only, payout only, or both
- Single currency or multi-currency settlements
- Any combination of the four fields

**Note:** `idempotency_key` is required.

**Example:**
```bash
curl -X POST http://localhost:8080/users/1/wager \
  -H "Content-Type: application/json" \
  -d '{"stake_gc":500,"payout_gc":900,"idempotency_key":"wager-001"}'
```

**Response:**
```json
[
  {
    "id": 3,
    "user_id": 1,
    "currency": "GC",
    "type": "wager_gc",
    "amount": 500,
    "balance_after": 9500,
    "created_at": "2025-11-14T10:05:00Z"
  },
  {
    "id": 4,
    "user_id": 1,
    "currency": "GC",
    "type": "win_gc",
    "amount": 900,
    "balance_after": 10400,
    "created_at": "2025-11-14T10:05:00Z"
  }
]

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

**Note:** `idempotency_key` is required.

**Example:**
```bash
curl -X POST http://localhost:8080/users/1/redeem \
  -H "Content-Type: application/json" \
  -d '{"amount_sc":10,"idempotency_key":"redeem-001"}'
```

**Response:**
```json
{
  "id": 5,
  "user_id": 1,
  "currency": "SC",
  "type": "redeem_sc",
  "amount": 10,
  "balance_after": 0,
  "created_at": "2025-11-14T10:10:00Z"
}

## 📋 Example Test Workflow

**Complete end-to-end test sequence** (available in Postman collection):

1. **Health Check** - Verify service is running
2. **Get User** - Check alice's initial state (user ID: 1)
3. **Purchase Package** - Buy starter_10k (10,000 GC + 10 SC)
4. **View Balance** - Verify purchase credited correctly
5. **Place Wager** - Simulate game round (stake 500 GC, win 900 GC)
6. **View Transactions** - See transaction history with pagination
7. **Redeem SC** - Convert 5 SC to cash equivalent
8. **Test Idempotency** - Retry same request, verify no duplicate
9. **View Final State** - Check all balances and statistics updated

All requests are pre-configured in the Postman collection with proper idempotency keys.

---

## 🏗️ Architecture & Design

### Transaction-Based Architecture

All balances and statistics are calculated dynamically from the immutable transaction ledger:

**Transaction Ledger (transactions table)** - Single source of truth:
- Every balance change recorded as a transaction
- `balance_after` field provides point-in-time snapshots
- Current balances calculated from latest transaction per currency
- Statistics aggregated from transaction history

**Calculation Strategy**:
- Balances: Read `balance_after` from most recent transaction for each currency
- Statistics: Aggregate `amount` fields filtered by transaction type (wager/win/redeem)
- All reads executed within database transactions for consistency

**Benefits**:
- Complete audit trail with no denormalization
- Simplified code - no balance synchronization logic
- Guaranteed consistency between balances and ledger

### Clean Layered Architecture

```
handlers/     → HTTP routing, request validation, response formatting
service/      → Business logic, transaction orchestration, idempotency handling
repository/   → Database access, query execution
models/       → Domain types and constants
```

### Concurrency Control: User-Level Serialization

**Problem**: Concurrent operations on the same user can cause race conditions and balance inconsistencies.

**Solution**: Per-user request serialization using mutex locks:
- Each user has their own mutex lock (managed via `sync.Map`)
- All financial operations (Purchase, Wager, Redeem) acquire the user's lock before execution
- Requests for the same user are serialized (executed one at a time)
- Requests for different users run in parallel (no global bottleneck)

**Benefits**:
- **Eliminates race conditions** - No two requests for same user run concurrently
- **Simpler code** - No need for constraint violation handling or retry logic
- **Better reasoning** - Sequential execution matches real-world casino gameplay
- **Natural backpressure** - Requests queue if user is busy
- **Prevents balance corruption** - No concurrent balance calculations

### Key Design Principles

**Idempotency Protection**
- All financial operations require unique idempotency keys
- Duplicate requests return original transactions without creating new records
- Safe to retry on timeout, connection error, or server crash
- PRIMARY KEY constraint prevents duplicate operations at database level
- Keys auto-expire after 24 hours

**Atomicity & Consistency**
- All operations wrapped in database transactions
- Per-user request serialization eliminates race conditions
- All-or-nothing guarantee for multi-step operations

**Immutable Ledger**
- Transactions never modified after creation
- Corrections are new offsetting transactions
- Complete audit trail maintained

**Efficient Pagination**
- Cursor-based pagination (encoded ID + timestamp)
- Scales to millions of transactions per user

## Database Schema

### Users Table
```sql
id          SERIAL PRIMARY KEY
username    VARCHAR(255) UNIQUE
created_at  TIMESTAMP
```
*Note: Balances and statistics are calculated from the transactions table, not stored here.*

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
key               VARCHAR(255) PRIMARY KEY
user_id           INTEGER REFERENCES users(id)
transaction_ids   INTEGER[]
created_at        TIMESTAMP
```
*Note: `key` is globally unique. `transaction_ids` supports multi-transaction operations (e.g., purchases).*

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

## 📊 Project Structure

```
wallet-ledger/
├── main.go                    # Entry point, server initialization
├── handlers/handlers.go       # HTTP routing and request handling
├── service/service.go         # Business logic and orchestration
├── service/errors.go          # Custom error types
├── repository/repository.go   # Database access layer
├── models/models.go           # Domain types and constants
├── migrations/001_init.sql    # Database schema
├── docker-compose.yml         # Container orchestration
├── Dockerfile                 # Multi-stage Go build
└── postman-collection.json    # Pre-configured API tests
```

**Layer Separation Benefits:**
- Easy unit testing with mock interfaces
- Clear separation of concerns
- Maintainable and scalable codebase
- Database can be swapped without touching business logic

## 🧪 Testing the API

### Method 1: Postman (Recommended)

**The easiest way to explore and test all endpoints:**

1. Download and install [Postman](https://www.postman.com/downloads/)
2. Import `postman-collection.json` from the project root
3. All 17 API endpoints will be loaded with pre-configured requests
4. Click any request and hit **Send**
5. Use **Collection Runner** to execute all tests sequentially

### Method 2: VS Code REST Client

**For developers who prefer staying in the editor:**

1. Install the **REST Client** extension in VS Code
2. Open `api-requests.http` in the project
3. Click "Send Request" above any HTTP request
4. View responses inline in VS Code

### Method 3: Command Line Scripts

**For automated testing:**

**Windows PowerShell:**
```powershell
# Run the comprehensive test suite (32 tests including idempotency)
.\test-examples.ps1

# If you get permission error, run once:
# Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
```

**Linux/Mac:**
```bash
# Run the provided test script
bash curl-examples.sh

# Or individual requests
curl http://localhost:8080/health
curl http://localhost:8080/users/1
```

## ✨ Additional Features

- **Health Check Endpoint**: Verifies database connectivity
- **Graceful Shutdown**: Handles SIGINT/SIGTERM signals properly
- **Background Cleanup**: Auto-removes expired idempotency keys (24+ hours old) every hour
- **Connection Pooling**: Optimized PostgreSQL connection management
- **Request Logging**: HTTP middleware for debugging and monitoring
- **Type-safe Errors**: Custom error types with proper wrapping (`errors.Is()` compatible)

## 🔍 Database Inspection

You can inspect the database directly using Docker Desktop or command line:

**Using Docker Desktop:**
1. Open Docker Desktop
2. Click on `wtls-db-1` container
3. Go to "Exec" tab
4. Run: `psql -U postgres -d wallet_ledger`

**Common PostgreSQL Commands:**

```bash
# List all tables
\dt

# Describe table structure
\d users
\d transactions
\d idempotency_keys

# View recent transactions
SELECT * FROM transactions ORDER BY id DESC LIMIT 10;

# Check user balances
SELECT * FROM users;

# View idempotency keys
SELECT key, user_id, transaction_ids, created_at FROM idempotency_keys;

# Aggregate statistics
SELECT currency, type, COUNT(*), SUM(amount) 
FROM transactions 
GROUP BY currency, type;

# Exit psql
\q
```

**Using Command Line:**

```bash
# From your terminal (not in Docker)
docker exec -it wtls-db-1 psql -U postgres -d wallet_ledger -c "SELECT * FROM transactions LIMIT 5;"
```
