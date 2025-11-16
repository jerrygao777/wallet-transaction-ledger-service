#!/bin/bash
# API Testing Examples using curl
# These commands work on Linux, Mac, and Git Bash on Windows
# Note: For automatic cursor pagination, install jq (JSON processor)
#   - Mac: brew install jq
#   - Ubuntu/Debian: sudo apt-get install jq
#   - Windows: Download from https://stedolan.github.io/jq/

# Health Check
curl http://localhost:8080/health

# List Available Packages
curl http://localhost:8080/packages

# Get User with Balances
curl http://localhost:8080/users/1

# List User Transactions
curl "http://localhost:8080/users/1/transactions?limit=10"

# List User Transactions - Filter by Currency
curl "http://localhost:8080/users/1/transactions?currency=GC&limit=10"

# List User Transactions - Filter by Type
curl "http://localhost:8080/users/1/transactions?type=purchase&limit=10"

# Purchase Starter Package
curl -X POST http://localhost:8080/users/1/purchase \
  -H "Content-Type: application/json" \
  -d '{"package_code":"starter_10k","idempotency_key":"purchase-001"}'

# Purchase Grinder Package
curl -X POST http://localhost:8080/users/1/purchase \
  -H "Content-Type: application/json" \
  -d '{"package_code":"grinder_50k","idempotency_key":"purchase-002"}'

# Purchase HighRoller Package
curl -X POST http://localhost:8080/users/1/purchase \
  -H "Content-Type: application/json" \
  -d '{"package_code":"highroller_250k","idempotency_key":"purchase-003"}'

# Wager Gold Coins (Win)
curl -X POST http://localhost:8080/users/1/wager \
  -H "Content-Type: application/json" \
  -d '{"stake_gc":500,"payout_gc":900,"idempotency_key":"wager-gc-win-001"}'

# Wager Gold Coins (Lose)
curl -X POST http://localhost:8080/users/1/wager \
  -H "Content-Type: application/json" \
  -d '{"stake_gc":1000,"payout_gc":0,"idempotency_key":"wager-gc-lose-001"}'

# Wager Sweeps Coins (Win)
curl -X POST http://localhost:8080/users/1/wager \
  -H "Content-Type: application/json" \
  -d '{"stake_sc":5,"payout_sc":9,"idempotency_key":"wager-sc-win-001"}'

# Wager Sweeps Coins (Lose)
curl -X POST http://localhost:8080/users/1/wager \
  -H "Content-Type: application/json" \
  -d '{"stake_sc":2,"payout_sc":0,"idempotency_key":"wager-sc-lose-001"}'

# Wager - Payout Only GC
curl -X POST http://localhost:8080/users/1/wager \
  -H "Content-Type: application/json" \
  -d '{"payout_gc":1000,"idempotency_key":"wager-payout-gc-001"}'

# Wager - Payout Only SC
curl -X POST http://localhost:8080/users/1/wager \
  -H "Content-Type: application/json" \
  -d '{"payout_sc":5,"idempotency_key":"wager-payout-sc-001"}'

# Wager - All Currencies (complex settlement)
curl -X POST http://localhost:8080/users/1/wager \
  -H "Content-Type: application/json" \
  -d '{"stake_gc":50,"payout_gc":75,"stake_sc":1,"payout_sc":2,"idempotency_key":"wager-all-001"}'

# Redeem Sweeps Coins
curl -X POST http://localhost:8080/users/1/redeem \
  -H "Content-Type: application/json" \
  -d '{"amount_sc":10,"idempotency_key":"redeem-001"}'

# Test Idempotency - Purchase Same Key (should return same result)
curl -X POST http://localhost:8080/users/1/purchase \
  -H "Content-Type: application/json" \
  -d '{"package_code":"starter_10k","idempotency_key":"purchase-001"}'

# Test Idempotency - Wager Same Key (should return same transactions without creating duplicates)
curl -X POST http://localhost:8080/users/1/wager \
  -H "Content-Type: application/json" \
  -d '{"stake_gc":500,"payout_gc":900,"idempotency_key":"wager-gc-win-001"}'

# Test Idempotency - Redeem Same Key (should return same transaction without creating duplicate)
curl -X POST http://localhost:8080/users/1/redeem \
  -H "Content-Type: application/json" \
  -d '{"amount_sc":10,"idempotency_key":"redeem-001"}'

# Error Test - Insufficient Gold Coins (should fail)
curl -X POST http://localhost:8080/users/1/wager \
  -H "Content-Type: application/json" \
  -d '{"stake_gc":999999999,"idempotency_key":"wager-insufficient-gc"}'

# Error Test - Insufficient Sweep Coins (should fail)
curl -X POST http://localhost:8080/users/1/wager \
  -H "Content-Type: application/json" \
  -d '{"stake_sc":999999,"idempotency_key":"wager-insufficient-sc"}'

# Error Test - Insufficient SC for Redeem (should fail)
curl -X POST http://localhost:8080/users/1/redeem \
  -H "Content-Type: application/json" \
  -d '{"amount_sc":999999,"idempotency_key":"redeem-insufficient-sc"}'

# Error Test - Negative Gold Coins Amount (should fail)
curl -X POST http://localhost:8080/users/1/wager \
  -H "Content-Type: application/json" \
  -d '{"stake_gc":-100,"idempotency_key":"wager-negative-gc"}'

# Error Test - Negative Sweep Coins Amount (should fail)
curl -X POST http://localhost:8080/users/1/wager \
  -H "Content-Type: application/json" \
  -d '{"payout_sc":-50,"idempotency_key":"wager-negative-sc"}'

# Error Test - All Fields Zero (should fail)
curl -X POST http://localhost:8080/users/1/wager \
  -H "Content-Type: application/json" \
  -d '{"stake_gc":0,"payout_gc":0,"stake_sc":0,"payout_sc":0,"idempotency_key":"wager-all-zero"}'

# Error Test - Negative Redeem Amount (should fail)
curl -X POST http://localhost:8080/users/1/redeem \
  -H "Content-Type: application/json" \
  -d '{"amount_sc":-10,"idempotency_key":"redeem-negative"}'

# Test User 2
curl http://localhost:8080/users/2

# Test User 3
curl http://localhost:8080/users/3

# Cursor Pagination - Page 1 (limit 3) - Automatic cursor extraction
echo "Fetching Page 1..."
RESPONSE=$(curl -s "http://localhost:8080/users/1/transactions?limit=3")
echo "$RESPONSE" | jq .
NEXT_CURSOR=$(echo "$RESPONSE" | jq -r '.next_cursor // empty')

# Cursor Pagination - Page 2 (automatically uses cursor from Page 1)
if [ -n "$NEXT_CURSOR" ]; then
  echo -e "\nFetching Page 2 with cursor: $NEXT_CURSOR"
  curl -s "http://localhost:8080/users/1/transactions?limit=3&cursor=$NEXT_CURSOR" | jq .
else
  echo "No next_cursor found (last page)"
fi

# Cursor Pagination with Filter - GC transactions only
echo -e "\nFetching GC transactions (Page 1)..."
GC_RESPONSE=$(curl -s "http://localhost:8080/users/1/transactions?currency=GC&limit=2")
echo "$GC_RESPONSE" | jq .
GC_CURSOR=$(echo "$GC_RESPONSE" | jq -r '.next_cursor // empty')
if [ -n "$GC_CURSOR" ]; then
  echo "Next GC page cursor: $GC_CURSOR"
fi

# Cursor Pagination with Filter - Purchase type only
echo -e "\nFetching Purchase transactions (Page 1)..."
PURCHASE_RESPONSE=$(curl -s "http://localhost:8080/users/1/transactions?type=purchase&limit=2")
echo "$PURCHASE_RESPONSE" | jq .
PURCHASE_CURSOR=$(echo "$PURCHASE_RESPONSE" | jq -r '.next_cursor // empty')
if [ -n "$PURCHASE_CURSOR" ]; then
  echo "Next Purchase page cursor: $PURCHASE_CURSOR"
fi

# Cursor Pagination with Combined Filters - Purchase + SC (Page 1)
echo -e "\nFetching Purchase + SC transactions (Page 1)..."
COMBINED_RESPONSE=$(curl -s "http://localhost:8080/users/1/transactions?type=purchase&currency=SC&limit=1")
echo "$COMBINED_RESPONSE" | jq .
COMBINED_CURSOR=$(echo "$COMBINED_RESPONSE" | jq -r '.next_cursor // empty')

# Cursor Pagination - Navigate to next page with combined filters (automatic)
if [ -n "$COMBINED_CURSOR" ]; then
  echo -e "\nFetching Purchase + SC transactions (Page 2) with cursor: $COMBINED_CURSOR"
  curl -s "http://localhost:8080/users/1/transactions?type=purchase&currency=SC&limit=1&cursor=$COMBINED_CURSOR" | jq .
else
  echo "No next_cursor found (last page)"
fi
