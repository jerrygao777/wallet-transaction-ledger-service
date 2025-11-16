package service

import (
	"encoding/json"
	"fmt"
	"wallet-ledger/models"
	"wallet-ledger/repository"
)

type WalletService struct {
	repo *repository.Repository
}

func New(repo *repository.Repository) *WalletService {
	return &WalletService{repo: repo}
}

// GetUserWithBalances retrieves a user with balances and stats
func (s *WalletService) GetUserWithBalances(userID int) (*models.UserWithBalances, error) {
	return s.repo.GetUserWithBalances(userID)
}

// ListTransactions retrieves paginated transactions
func (s *WalletService) ListTransactions(userID int, cursor *string, limit int, txType *models.TransactionType, currency *models.Currency) (*models.TransactionList, error) {
	// Verify user exists
	_, err := s.repo.GetUser(userID)
	if err != nil {
		return nil, err
	}

	return s.repo.ListTransactions(userID, cursor, limit, txType, currency)
}

// Purchase handles purchasing a package with idempotency
func (s *WalletService) Purchase(userID int, packageCode string, idempotencyKey string) (*models.Transaction, error) {
	// Validate package
	pkg, ok := models.Packages[packageCode]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrInvalidPackage, packageCode)
	}

	// Verify user exists
	_, err := s.repo.GetUser(userID)
	if err != nil {
		return nil, err
	}

	// Start transaction
	tx, err := s.repo.BeginTx()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Check idempotency
	existingTxID, err := s.repo.CheckIdempotencyKey(tx, idempotencyKey, userID)
	if err != nil {
		return nil, err
	}
	if existingTxID != nil {
		// Already processed, return existing transaction
		tx.Commit()
		return s.repo.GetTransaction(*existingTxID)
	}

	// Create GC transaction
	gcBalance, err := s.repo.GetCurrentBalance(tx, userID, models.CurrencyGC)
	if err != nil {
		return nil, err
	}

	metadata := map[string]interface{}{
		"package_code": packageCode,
		"gc_amount":    pkg.GoldCoins,
		"sc_amount":    pkg.SweepCoins,
	}
	metadataJSON, _ := json.Marshal(metadata)

	gcTx := &models.Transaction{
		UserID:       userID,
		Currency:     models.CurrencyGC,
		Type:         models.TransactionTypePurchase,
		Amount:       pkg.GoldCoins,
		BalanceAfter: gcBalance + pkg.GoldCoins,
		Metadata:     metadataJSON,
	}

	err = s.repo.CreateTransaction(tx, gcTx)
	if err != nil {
		return nil, err
	}

	// Update wallet balances (GC purchased, SC given as bonus)
	err = s.repo.UpdateWalletBalance(tx, userID, models.CurrencyGC, pkg.GoldCoins)
	if err != nil {
		return nil, err
	}

	err = s.repo.UpdateWalletBalance(tx, userID, models.CurrencySC, pkg.SweepCoins)
	if err != nil {
		return nil, err
	}

	// Save idempotency key (using GC transaction ID as reference)
	err = s.repo.SaveIdempotencyKey(tx, idempotencyKey, userID, gcTx.ID)
	if err != nil {
		return nil, err
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return gcTx, nil
}

// Wager handles a wager with stake and payout
func (s *WalletService) Wager(userID int, stakeGC, payoutGC, stakeSC, payoutSC int64, idempotencyKey string) error {
	// Validate inputs
	if stakeGC < 0 || payoutGC < 0 || stakeSC < 0 || payoutSC < 0 {
		return fmt.Errorf("amounts cannot be negative")
	}
	if stakeGC == 0 && payoutGC == 0 && stakeSC == 0 && payoutSC == 0 {
		return fmt.Errorf("at least one amount must be greater than zero")
	}

	// Verify user exists
	_, err := s.repo.GetUser(userID)
	if err != nil {
		return err
	}

	// Start transaction
	tx, err := s.repo.BeginTx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Check idempotency
	existingTxID, err := s.repo.CheckIdempotencyKey(tx, idempotencyKey, userID)
	if err != nil {
		return err
	}
	if existingTxID != nil {
		// Already processed, return success
		tx.Commit()
		return nil
	}

	// Handle Gold Coins stake
	if stakeGC > 0 {
		gcBalance, err := s.repo.GetCurrentBalance(tx, userID, models.CurrencyGC)
		if err != nil {
			return err
		}

		if gcBalance < stakeGC {
			return fmt.Errorf("%w: gold coins - have %d, need %d", ErrInsufficientFunds, gcBalance, stakeGC)
		}

		// Create wager transaction
		wagerTx := &models.Transaction{
			UserID:       userID,
			Currency:     models.CurrencyGC,
			Type:         models.TransactionTypeWagerGC,
			Amount:       stakeGC,
			BalanceAfter: gcBalance - stakeGC,
		}
		err = s.repo.CreateTransaction(tx, wagerTx)
		if err != nil {
			return err
		}

		// Update wallet balance and stats
		err = s.repo.UpdateWalletBalance(tx, userID, models.CurrencyGC, -stakeGC)
		if err != nil {
			return err
		}
		err = s.repo.UpdateWalletStat(tx, userID, "total_gc_wagered", stakeGC)
		if err != nil {
			return err
		}
	}

	// Handle Gold Coins payout (independent of stake)
	if payoutGC > 0 {
		gcBalance, err := s.repo.GetCurrentBalance(tx, userID, models.CurrencyGC)
		if err != nil {
			return err
		}

		winTx := &models.Transaction{
			UserID:       userID,
			Currency:     models.CurrencyGC,
			Type:         models.TransactionTypeWinGC,
			Amount:       payoutGC,
			BalanceAfter: gcBalance + payoutGC,
		}
		err = s.repo.CreateTransaction(tx, winTx)
		if err != nil {
			return err
		}

		// Update wallet balance and stats
		err = s.repo.UpdateWalletBalance(tx, userID, models.CurrencyGC, payoutGC)
		if err != nil {
			return err
		}
		err = s.repo.UpdateWalletStat(tx, userID, "total_gc_won", payoutGC)
		if err != nil {
			return err
		}
	}

	// Handle Sweeps Coins stake
	if stakeSC > 0 {
		scBalance, err := s.repo.GetCurrentBalance(tx, userID, models.CurrencySC)
		if err != nil {
			return err
		}

		if scBalance < stakeSC {
			return fmt.Errorf("%w: sweeps coins - have %d, need %d", ErrInsufficientFunds, scBalance, stakeSC)
		}

		// Create wager transaction
		wagerTx := &models.Transaction{
			UserID:       userID,
			Currency:     models.CurrencySC,
			Type:         models.TransactionTypeWagerSC,
			Amount:       stakeSC,
			BalanceAfter: scBalance - stakeSC,
		}
		err = s.repo.CreateTransaction(tx, wagerTx)
		if err != nil {
			return err
		}

		// Update wallet balance and stats
		err = s.repo.UpdateWalletBalance(tx, userID, models.CurrencySC, -stakeSC)
		if err != nil {
			return err
		}
		err = s.repo.UpdateWalletStat(tx, userID, "total_sc_wagered", stakeSC)
		if err != nil {
			return err
		}
	}

	// Handle Sweeps Coins payout (independent of stake)
	if payoutSC > 0 {
		scBalance, err := s.repo.GetCurrentBalance(tx, userID, models.CurrencySC)
		if err != nil {
			return err
		}

		winTx := &models.Transaction{
			UserID:       userID,
			Currency:     models.CurrencySC,
			Type:         models.TransactionTypeWinSC,
			Amount:       payoutSC,
			BalanceAfter: scBalance + payoutSC,
		}
		err = s.repo.CreateTransaction(tx, winTx)
		if err != nil {
			return err
		}

		// Update wallet balance and stats
		err = s.repo.UpdateWalletBalance(tx, userID, models.CurrencySC, payoutSC)
		if err != nil {
			return err
		}
		err = s.repo.UpdateWalletStat(tx, userID, "total_sc_won", payoutSC)
		if err != nil {
			return err
		}
	}

	// Save idempotency key (use userID as transaction reference for wagers)
	err = s.repo.SaveIdempotencyKey(tx, idempotencyKey, userID, userID)
	if err != nil {
		return err
	}

	// Commit transaction
	return tx.Commit()
}

// Redeem handles redeeming Sweeps Coins
func (s *WalletService) Redeem(userID int, amount int64, idempotencyKey string) error {
	if amount <= 0 {
		return fmt.Errorf("redemption amount must be positive")
	}

	// Verify user exists
	_, err := s.repo.GetUser(userID)
	if err != nil {
		return err
	}

	// Start transaction
	tx, err := s.repo.BeginTx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Check idempotency
	existingTxID, err := s.repo.CheckIdempotencyKey(tx, idempotencyKey, userID)
	if err != nil {
		return err
	}
	if existingTxID != nil {
		// Already processed, return success
		tx.Commit()
		return nil
	}

	// Get current balance
	scBalance, err := s.repo.GetCurrentBalance(tx, userID, models.CurrencySC)
	if err != nil {
		return err
	}

	if scBalance < amount {
		return fmt.Errorf("%w: sweeps coins - have %d, need %d", ErrInsufficientFunds, scBalance, amount)
	}

	// Create redeem transaction
	redeemTx := &models.Transaction{
		UserID:       userID,
		Currency:     models.CurrencySC,
		Type:         models.TransactionTypeRedeemSC,
		Amount:       amount,
		BalanceAfter: scBalance - amount,
	}

	err = s.repo.CreateTransaction(tx, redeemTx)
	if err != nil {
		return err
	}

	// Update wallet balance and stats
	err = s.repo.UpdateWalletBalance(tx, userID, models.CurrencySC, -amount)
	if err != nil {
		return err
	}
	err = s.repo.UpdateWalletStat(tx, userID, "total_sc_redeemed", amount)
	if err != nil {
		return err
	}

	// Save idempotency key
	err = s.repo.SaveIdempotencyKey(tx, idempotencyKey, userID, redeemTx.ID)
	if err != nil {
		return err
	}

	// Commit transaction
	return tx.Commit()
}
