#!/bin/bash
# API Testing Examples using curl
# These commands work on Linux, Mac, and Git Bash on Windows

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

# Redeem Sweeps Coins
curl -X POST http://localhost:8080/users/1/redeem \
  -H "Content-Type: application/json" \
  -d '{"amount_sc":10,"idempotency_key":"redeem-001"}'

# Test Idempotency - Purchase Same Key (should return same result)
curl -X POST http://localhost:8080/users/1/purchase \
  -H "Content-Type: application/json" \
  -d '{"package_code":"starter_10k","idempotency_key":"purchase-001"}'

# Test Idempotency - Wager Same Key (should return success without duplicate)
curl -X POST http://localhost:8080/users/1/wager \
  -H "Content-Type: application/json" \
  -d '{"stake_gc":500,"payout_gc":900,"idempotency_key":"wager-gc-win-001"}'

# Test Idempotency - Redeem Same Key (should return success without duplicate)
curl -X POST http://localhost:8080/users/1/redeem \
  -H "Content-Type: application/json" \
  -d '{"amount_sc":10,"idempotency_key":"redeem-001"}'

# Test User 2
curl http://localhost:8080/users/2

# Test User 3
curl http://localhost:8080/users/3
