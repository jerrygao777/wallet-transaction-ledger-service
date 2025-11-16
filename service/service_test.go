package service

import (
	"errors"
	"testing"
)

// Testing Strategy:
// These unit tests focus on the service layer's business logic validation that
// occurs BEFORE any repository/database calls. This includes:
// - Input validation (negative amounts, zeros, invalid package codes)
// - Business rules (packages must have GC, at least one amount > 0)
// - Error wrapping and error type validation
// - User-level lock behavior
//
// The validation logic in Purchase, Wager, and Redeem returns early before
// touching the repository, so we can test with repo: nil.
//
// For full end-to-end testing with database interactions, see the integration
// tests in test-examples.ps1 (39 integration tests + 3 concurrency tests).

// Test Purchase - Invalid Package (validation logic)
func TestPurchase_InvalidPackage(t *testing.T) {
	// This test validates business logic before any repository calls
	// The service will check package validity first
	service := &WalletService{repo: nil}

	_, err := service.Purchase(1, "invalid_package", "key-001")

	if !errors.Is(err, ErrInvalidPackage) {
		t.Errorf("expected ErrInvalidPackage, got %v", err)
	}
}

// Test Wager - Negative Amounts (validation logic)
func TestWager_NegativeAmounts(t *testing.T) {
	// Test input validation before any repository calls
	service := &WalletService{repo: nil}

	tests := []struct {
		name     string
		stakeGC  int64
		payoutGC int64
		stakeSC  int64
		payoutSC int64
	}{
		{"negative stake GC", -100, 0, 0, 0},
		{"negative payout GC", 0, -100, 0, 0},
		{"negative stake SC", 0, 0, -10, 0},
		{"negative payout SC", 0, 0, 0, -10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.Wager(1, tt.stakeGC, tt.payoutGC, tt.stakeSC, tt.payoutSC, "key-001")

			if !errors.Is(err, ErrInvalidInput) {
				t.Errorf("expected ErrInvalidInput, got %v", err)
			}

			if err == nil || err.Error() != "amounts cannot be negative: invalid input" {
				t.Errorf("expected 'amounts cannot be negative' message, got %v", err)
			}
		})
	}
}

// Test Wager - All Zeros (validation logic)
func TestWager_AllZeros(t *testing.T) {
	// Test validation: at least one amount must be > 0
	service := &WalletService{repo: nil}

	_, err := service.Wager(1, 0, 0, 0, 0, "key-001")

	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}

	if err == nil || err.Error() != "at least one amount must be greater than zero: invalid input" {
		t.Errorf("expected 'at least one amount must be greater than zero' message, got %v", err)
	}
}

// Test Redeem - Negative Amount (validation logic)
func TestRedeem_NegativeAmount(t *testing.T) {
	// Test input validation before any repository calls
	service := &WalletService{repo: nil}

	_, err := service.Redeem(1, -10, "key-001")

	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}

	if err == nil || err.Error() != "redemption amount must be positive: invalid input" {
		t.Errorf("expected 'redemption amount must be positive' message, got %v", err)
	}
}

// Test Redeem - Zero Amount (validation logic)
func TestRedeem_ZeroAmount(t *testing.T) {
	// Test validation: amount must be positive
	service := &WalletService{repo: nil}

	_, err := service.Redeem(1, 0, "key-001")

	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

// Test User-Level Lock Serialization
func TestUserLevelLocks(t *testing.T) {
	service := &WalletService{}

	// Get lock for user 1
	lock1a := service.getUserLock(1)
	lock1b := service.getUserLock(1)

	// Should return same mutex for same user
	if lock1a != lock1b {
		t.Error("expected same lock instance for same user")
	}

	// Get lock for user 2
	lock2 := service.getUserLock(2)

	// Should return different mutex for different user
	if lock1a == lock2 {
		t.Error("expected different lock instances for different users")
	}
}
