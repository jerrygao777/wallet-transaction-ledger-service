# PowerShell Test Script for Wallet Transaction Ledger Service
# Run this script after starting the service with: docker-compose up -d

Write-Host "=== Wallet Transaction Ledger Service - PowerShell Test Script ===" -ForegroundColor Cyan
Write-Host ""

# Health Check
Write-Host "1. Health Check" -ForegroundColor Yellow
Invoke-RestMethod -Uri "http://localhost:8080/health"
Write-Host ""

# List Available Packages
Write-Host "2. List Available Packages" -ForegroundColor Yellow
Invoke-RestMethod -Uri "http://localhost:8080/packages"
Write-Host ""

# Get User with Balances
Write-Host "3. Get User 1 (Initial State)" -ForegroundColor Yellow
Invoke-RestMethod -Uri "http://localhost:8080/users/1"
Write-Host ""

# Purchase Starter Package
Write-Host "4. Purchase Starter Package" -ForegroundColor Yellow
$body = @{ package_code = "starter_10k"; idempotency_key = "purchase-001" } | ConvertTo-Json
Invoke-RestMethod -Uri "http://localhost:8080/users/1/purchase" -Method Post -Body $body -ContentType "application/json"
Write-Host ""

# Purchase Grinder Package
Write-Host "5. Purchase Grinder Package" -ForegroundColor Yellow
$body = @{ package_code = "grinder_50k"; idempotency_key = "purchase-002" } | ConvertTo-Json
Invoke-RestMethod -Uri "http://localhost:8080/users/1/purchase" -Method Post -Body $body -ContentType "application/json"
Write-Host ""

# Purchase HighRoller Package
Write-Host "6. Purchase HighRoller Package" -ForegroundColor Yellow
$body = @{ package_code = "highroller_250k"; idempotency_key = "purchase-003" } | ConvertTo-Json
Invoke-RestMethod -Uri "http://localhost:8080/users/1/purchase" -Method Post -Body $body -ContentType "application/json"
Write-Host ""

# Check Balance After Purchases
Write-Host "7. Check Balance After Purchases" -ForegroundColor Yellow
Invoke-RestMethod -Uri "http://localhost:8080/users/1"
Write-Host ""

# Wager Gold Coins (Win)
Write-Host "8. Wager Gold Coins (Win: stake 500, payout 900)" -ForegroundColor Yellow
$body = @{ stake_gc = 500; payout_gc = 900; idempotency_key = "wager-gc-win-001" } | ConvertTo-Json
Invoke-RestMethod -Uri "http://localhost:8080/users/1/wager" -Method Post -Body $body -ContentType "application/json"
Write-Host ""

# Wager Gold Coins (Lose)
Write-Host "9. Wager Gold Coins (Lose: stake 1000, payout 0)" -ForegroundColor Yellow
$body = @{ stake_gc = 1000; payout_gc = 0; idempotency_key = "wager-gc-lose-001" } | ConvertTo-Json
Invoke-RestMethod -Uri "http://localhost:8080/users/1/wager" -Method Post -Body $body -ContentType "application/json"
Write-Host ""

# Wager Sweeps Coins (Win)
Write-Host "10. Wager Sweeps Coins (Win: stake 5, payout 9)" -ForegroundColor Yellow
$body = @{ stake_sc = 5; payout_sc = 9; idempotency_key = "wager-sc-win-001" } | ConvertTo-Json
Invoke-RestMethod -Uri "http://localhost:8080/users/1/wager" -Method Post -Body $body -ContentType "application/json"
Write-Host ""

# Wager Sweeps Coins (Lose)
Write-Host "11. Wager Sweeps Coins (Lose: stake 2, payout 0)" -ForegroundColor Yellow
$body = @{ stake_sc = 2; payout_sc = 0; idempotency_key = "wager-sc-lose-001" } | ConvertTo-Json
Invoke-RestMethod -Uri "http://localhost:8080/users/1/wager" -Method Post -Body $body -ContentType "application/json"
Write-Host ""

# Check Balance After Wagers
Write-Host "12. Check Balance After Wagers" -ForegroundColor Yellow
Invoke-RestMethod -Uri "http://localhost:8080/users/1"
Write-Host ""

# Redeem Sweeps Coins
Write-Host "13. Redeem 10 Sweeps Coins" -ForegroundColor Yellow
$body = @{ amount_sc = 10; idempotency_key = "redeem-001" } | ConvertTo-Json
Invoke-RestMethod -Uri "http://localhost:8080/users/1/redeem" -Method Post -Body $body -ContentType "application/json"
Write-Host ""

# Check Balance After Redeem
Write-Host "14. Check Balance After Redeem" -ForegroundColor Yellow
Invoke-RestMethod -Uri "http://localhost:8080/users/1"
Write-Host ""

# List Transactions
Write-Host "15. List User Transactions (First 10)" -ForegroundColor Yellow
$response = Invoke-RestMethod -Uri "http://localhost:8080/users/1/transactions?limit=10"
$response.items | Format-Table -Property id,type,currency,amount,balance_after
Write-Host ""

# Test Idempotency - Purchase
Write-Host "16. Test Idempotency - Purchase Same Key (should return same result)" -ForegroundColor Yellow
$body = @{ package_code = "starter_10k"; idempotency_key = "purchase-001" } | ConvertTo-Json
Invoke-RestMethod -Uri "http://localhost:8080/users/1/purchase" -Method Post -Body $body -ContentType "application/json"
Write-Host ""

# Test Idempotency - Wager
Write-Host "17. Test Idempotency - Wager Same Key (should return success without duplicate)" -ForegroundColor Yellow
$body = @{ stake_gc = 500; payout_gc = 900; idempotency_key = "wager-gc-win-001" } | ConvertTo-Json
Invoke-RestMethod -Uri "http://localhost:8080/users/1/wager" -Method Post -Body $body -ContentType "application/json"
Write-Host ""

# Test Idempotency - Redeem
Write-Host "18. Test Idempotency - Redeem Same Key (should return success without duplicate)" -ForegroundColor Yellow
$body = @{ amount_sc = 10; idempotency_key = "redeem-001" } | ConvertTo-Json
Invoke-RestMethod -Uri "http://localhost:8080/users/1/redeem" -Method Post -Body $body -ContentType "application/json"
Write-Host ""

# Final Balance Check
Write-Host "19. Final Balance Check (should be same as step 14)" -ForegroundColor Yellow
Invoke-RestMethod -Uri "http://localhost:8080/users/1"
Write-Host ""

# List Transactions with Filter - Gold Coins Only
Write-Host "20. List Transactions - Gold Coins Only" -ForegroundColor Yellow
$response = Invoke-RestMethod -Uri "http://localhost:8080/users/1/transactions?currency=GC&limit=20"
$response.items | Format-Table -Property id,type,currency,amount,balance_after
Write-Host ""

# List Transactions with Filter - Sweeps Coins Only
Write-Host "21. List Transactions - Sweeps Coins Only" -ForegroundColor Yellow
$response = Invoke-RestMethod -Uri "http://localhost:8080/users/1/transactions?currency=SC&limit=20"
$response.items | Format-Table -Property id,type,currency,amount,balance_after
Write-Host ""

# Test User 2
Write-Host "22. Get User 2" -ForegroundColor Yellow
Invoke-RestMethod -Uri "http://localhost:8080/users/2"
Write-Host ""

# Test User 3
Write-Host "23. Get User 3" -ForegroundColor Yellow
Invoke-RestMethod -Uri "http://localhost:8080/users/3"
Write-Host ""

Write-Host "=== All Tests Completed ===" -ForegroundColor Green
