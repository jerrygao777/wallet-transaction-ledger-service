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

# Wager - Payout Only GC (Free Spins/Bonus)
Write-Host "11a. Wager - Payout Only GC (payout 1000, no stake)" -ForegroundColor Yellow
$body = @{ payout_gc = 1000; idempotency_key = "wager-payout-gc-001" } | ConvertTo-Json
Invoke-RestMethod -Uri "http://localhost:8080/users/1/wager" -Method Post -Body $body -ContentType "application/json"
Write-Host ""

# Wager - Payout Only SC
Write-Host "11b. Wager - Payout Only SC (payout 5, no stake)" -ForegroundColor Yellow
$body = @{ payout_sc = 5; idempotency_key = "wager-payout-sc-001" } | ConvertTo-Json
Invoke-RestMethod -Uri "http://localhost:8080/users/1/wager" -Method Post -Body $body -ContentType "application/json"
Write-Host ""

# Wager - All Four Fields (complex settlement)
Write-Host "11c. Wager - All Currencies (GC stake+payout, SC stake+payout)" -ForegroundColor Yellow
$body = @{ stake_gc = 50; payout_gc = 75; stake_sc = 1; payout_sc = 2; idempotency_key = "wager-all-001" } | ConvertTo-Json
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

# Error Handling Tests
Write-Host "=== Error Handling Tests ===" -ForegroundColor Magenta
Write-Host ""

# Test Insufficient GC
Write-Host "15. Test Insufficient Gold Coins (should fail)" -ForegroundColor Yellow
try {
    $body = @{ stake_gc = 999999999; idempotency_key = "wager-insufficient-gc" } | ConvertTo-Json
    Invoke-RestMethod -Uri "http://localhost:8080/users/1/wager" -Method Post -Body $body -ContentType "application/json" -ErrorAction Stop
    Write-Host "FAILED: Should have returned error" -ForegroundColor Red
} catch {
    $errorResponse = $_.ErrorDetails.Message | ConvertFrom-Json
    Write-Host "PASSED: $($errorResponse.error)" -ForegroundColor Green
}
Write-Host ""

# Test Insufficient SC
Write-Host "16. Test Insufficient Sweep Coins (should fail)" -ForegroundColor Yellow
try {
    $body = @{ stake_sc = 999999; idempotency_key = "wager-insufficient-sc" } | ConvertTo-Json
    Invoke-RestMethod -Uri "http://localhost:8080/users/1/wager" -Method Post -Body $body -ContentType "application/json" -ErrorAction Stop
    Write-Host "FAILED: Should have returned error" -ForegroundColor Red
} catch {
    $errorResponse = $_.ErrorDetails.Message | ConvertFrom-Json
    Write-Host "PASSED: $($errorResponse.error)" -ForegroundColor Green
}
Write-Host ""

# Test Insufficient SC for Redeem
Write-Host "17. Test Insufficient SC for Redeem (should fail)" -ForegroundColor Yellow
try {
    $body = @{ amount_sc = 999999; idempotency_key = "redeem-insufficient-sc" } | ConvertTo-Json
    Invoke-RestMethod -Uri "http://localhost:8080/users/1/redeem" -Method Post -Body $body -ContentType "application/json" -ErrorAction Stop
    Write-Host "FAILED: Should have returned error" -ForegroundColor Red
} catch {
    $errorResponse = $_.ErrorDetails.Message | ConvertFrom-Json
    Write-Host "PASSED: $($errorResponse.error)" -ForegroundColor Green
}
Write-Host ""

# Test Negative GC Amount
Write-Host "18. Test Negative Gold Coins Amount (should fail)" -ForegroundColor Yellow
try {
    $body = @{ stake_gc = -100; idempotency_key = "wager-negative-gc" } | ConvertTo-Json
    Invoke-RestMethod -Uri "http://localhost:8080/users/1/wager" -Method Post -Body $body -ContentType "application/json" -ErrorAction Stop
    Write-Host "FAILED: Should have returned error" -ForegroundColor Red
} catch {
    $errorResponse = $_.ErrorDetails.Message | ConvertFrom-Json
    Write-Host "PASSED: $($errorResponse.error)" -ForegroundColor Green
}
Write-Host ""

# Test Negative SC Amount
Write-Host "19. Test Negative Sweep Coins Amount (should fail)" -ForegroundColor Yellow
try {
    $body = @{ payout_sc = -50; idempotency_key = "wager-negative-sc" } | ConvertTo-Json
    Invoke-RestMethod -Uri "http://localhost:8080/users/1/wager" -Method Post -Body $body -ContentType "application/json" -ErrorAction Stop
    Write-Host "FAILED: Should have returned error" -ForegroundColor Red
} catch {
    $errorResponse = $_.ErrorDetails.Message | ConvertFrom-Json
    Write-Host "PASSED: $($errorResponse.error)" -ForegroundColor Green
}
Write-Host ""

# Test All Fields Zero
Write-Host "20. Test All Fields Zero (should fail)" -ForegroundColor Yellow
try {
    $body = @{ stake_gc = 0; payout_gc = 0; stake_sc = 0; payout_sc = 0; idempotency_key = "wager-all-zero" } | ConvertTo-Json
    Invoke-RestMethod -Uri "http://localhost:8080/users/1/wager" -Method Post -Body $body -ContentType "application/json" -ErrorAction Stop
    Write-Host "FAILED: Should have returned error" -ForegroundColor Red
} catch {
    $errorResponse = $_.ErrorDetails.Message | ConvertFrom-Json
    Write-Host "PASSED: $($errorResponse.error)" -ForegroundColor Green
}
Write-Host ""

# Test Negative Redeem Amount
Write-Host "21. Test Negative Redeem Amount (should fail)" -ForegroundColor Yellow
try {
    $body = @{ amount_sc = -10; idempotency_key = "redeem-negative" } | ConvertTo-Json
    Invoke-RestMethod -Uri "http://localhost:8080/users/1/redeem" -Method Post -Body $body -ContentType "application/json" -ErrorAction Stop
    Write-Host "FAILED: Should have returned error" -ForegroundColor Red
} catch {
    $errorResponse = $_.ErrorDetails.Message | ConvertFrom-Json
    Write-Host "PASSED: $($errorResponse.error)" -ForegroundColor Green
}
Write-Host ""

Write-Host "=== End Error Handling Tests ===" -ForegroundColor Magenta
Write-Host ""

# List Transactions
Write-Host "22. List User Transactions (First 10)" -ForegroundColor Yellow
$response = Invoke-RestMethod -Uri "http://localhost:8080/users/1/transactions?limit=10"
$response.items | Format-Table -Property id,type,currency,amount,balance_after
Write-Host ""

# Test Idempotency - Purchase
Write-Host "23. Test Idempotency - Purchase Same Key (should return same result)" -ForegroundColor Yellow
$body = @{ package_code = "starter_10k"; idempotency_key = "purchase-001" } | ConvertTo-Json
Invoke-RestMethod -Uri "http://localhost:8080/users/1/purchase" -Method Post -Body $body -ContentType "application/json"
Write-Host ""

# Test Idempotency - Wager
Write-Host "24. Test Idempotency - Wager Same Key (should return same transactions without duplicate)" -ForegroundColor Yellow
$body = @{ stake_gc = 500; payout_gc = 900; idempotency_key = "wager-gc-win-001" } | ConvertTo-Json
Invoke-RestMethod -Uri "http://localhost:8080/users/1/wager" -Method Post -Body $body -ContentType "application/json"
Write-Host ""

# Test Idempotency - Redeem
Write-Host "25. Test Idempotency - Redeem Same Key (should return same transaction without duplicate)" -ForegroundColor Yellow
$body = @{ amount_sc = 10; idempotency_key = "redeem-001" } | ConvertTo-Json
Invoke-RestMethod -Uri "http://localhost:8080/users/1/redeem" -Method Post -Body $body -ContentType "application/json"
Write-Host ""

# Final Balance Check
Write-Host "26. Final Balance Check (should be same as step 14)" -ForegroundColor Yellow
Invoke-RestMethod -Uri "http://localhost:8080/users/1"
Write-Host ""

# List Transactions with Filter - Gold Coins Only
Write-Host "27. List Transactions - Gold Coins Only" -ForegroundColor Yellow
$response = Invoke-RestMethod -Uri "http://localhost:8080/users/1/transactions?currency=GC&limit=20"
$response.items | Format-Table -Property id,type,currency,amount,balance_after
Write-Host ""

# List Transactions with Filter - Sweeps Coins Only
Write-Host "28. List Transactions - Sweeps Coins Only" -ForegroundColor Yellow
$response = Invoke-RestMethod -Uri "http://localhost:8080/users/1/transactions?currency=SC&limit=20"
$response.items | Format-Table -Property id,type,currency,amount,balance_after
Write-Host ""

# Test User 2
Write-Host "29. Get User 2" -ForegroundColor Yellow
Invoke-RestMethod -Uri "http://localhost:8080/users/2"
Write-Host ""

# Test User 3
Write-Host "30. Get User 3" -ForegroundColor Yellow
Invoke-RestMethod -Uri "http://localhost:8080/users/3"
Write-Host ""

# Cursor Pagination Tests
Write-Host "31. Cursor Pagination - Page 1 (limit 3)" -ForegroundColor Yellow
$page1 = Invoke-RestMethod -Uri "http://localhost:8080/users/1/transactions?limit=3"
Write-Host "Items: $($page1.items.Count), Next Cursor: $($page1.next_cursor)"
$page1.items | Format-Table -Property id,type,currency,amount
Write-Host ""

Write-Host "32. Cursor Pagination - Page 2 (using cursor)" -ForegroundColor Yellow
if ($page1.next_cursor) {
    $page2 = Invoke-RestMethod -Uri "http://localhost:8080/users/1/transactions?limit=3&cursor=$($page1.next_cursor)"
    Write-Host "Items: $($page2.items.Count), Next Cursor: $($page2.next_cursor)"
    $page2.items | Format-Table -Property id,type,currency,amount
}
Write-Host ""

Write-Host "33. Cursor Pagination - Page 3 (using cursor)" -ForegroundColor Yellow
if ($page2.next_cursor) {
    $page3 = Invoke-RestMethod -Uri "http://localhost:8080/users/1/transactions?limit=3&cursor=$($page2.next_cursor)"
    Write-Host "Items: $($page3.items.Count), Next Cursor: $($page3.next_cursor)"
    $page3.items | Format-Table -Property id,type,currency,amount
}
Write-Host ""

Write-Host "34. Cursor Pagination with Filter - GC transactions only" -ForegroundColor Yellow
$gcPage1 = Invoke-RestMethod -Uri "http://localhost:8080/users/1/transactions?currency=GC&limit=2"
Write-Host "GC Page 1 - Items: $($gcPage1.items.Count), Next Cursor: $($gcPage1.next_cursor)"
$gcPage1.items | Format-Table -Property id,type,currency,amount
Write-Host ""

Write-Host "35. Cursor Pagination with Filter - Purchase type only" -ForegroundColor Yellow
$purchasePage = Invoke-RestMethod -Uri "http://localhost:8080/users/1/transactions?type=purchase&limit=2"
Write-Host "Purchase Page 1 - Items: $($purchasePage.items.Count), Next Cursor: $($purchasePage.next_cursor)"
$purchasePage.items | Format-Table -Property id,type,currency,amount
Write-Host ""

Write-Host "36. Verify No Duplicates Across Pages" -ForegroundColor Yellow
$allIds = @()
$cursor = $null
$pageNum = 1
do {
    $url = "http://localhost:8080/users/1/transactions?limit=2"
    if ($cursor) { $url += "&cursor=$cursor" }
    $page = Invoke-RestMethod -Uri $url
    Write-Host "Page $pageNum : IDs = $($page.items.id -join ', ')"
    $allIds += $page.items.id
    $cursor = $page.next_cursor
    $pageNum++
} while ($cursor)
$duplicates = $allIds | Group-Object | Where-Object { $_.Count -gt 1 }
if ($duplicates) {
    Write-Host "FAILED: Found duplicate IDs!" -ForegroundColor Red
} else {
    Write-Host "PASSED: No duplicate IDs across all pages" -ForegroundColor Green
}
Write-Host ""

Write-Host "=== All Tests Completed ===" -ForegroundColor Green
