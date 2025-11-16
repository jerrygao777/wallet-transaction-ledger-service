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
func (s *WalletService) Purchase(userID int, packageCode string, idempotencyKey string) ([]*models.Transaction, error) {
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
	existingTxIDs, err := s.repo.CheckIdempotencyKey(tx, idempotencyKey, userID)
	if err != nil {
		return nil, err
	}
	if len(existingTxIDs) > 0 {
		// Already processed, return existing transactions
		tx.Commit()
		result := make([]*models.Transaction, 0, len(existingTxIDs))
		for _, txID := range existingTxIDs {
			transaction, err := s.repo.GetTransaction(txID)
			if err != nil {
				return nil, err
			}
			result = append(result, transaction)
		}
		return result, nil
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

	// Track created transactions for idempotency and result
	result := []*models.Transaction{gcTx}
	txIDs := []int{gcTx.ID}

	// Create SC transaction only if package includes sweep coins
	if pkg.SweepCoins > 0 {
		scBalance, err := s.repo.GetCurrentBalance(tx, userID, models.CurrencySC)
		if err != nil {
			return nil, err
		}

		scTx := &models.Transaction{
			UserID:       userID,
			Currency:     models.CurrencySC,
			Type:         models.TransactionTypePurchase,
			Amount:       pkg.SweepCoins,
			BalanceAfter: scBalance + pkg.SweepCoins,
			Metadata:     metadataJSON,
		}

		err = s.repo.CreateTransaction(tx, scTx)
		if err != nil {
			return nil, err
		}

		result = append(result, scTx)
		txIDs = append(txIDs, scTx.ID)
	}

	// Save idempotency key with all transaction IDs
	err = s.repo.SaveIdempotencyKey(tx, idempotencyKey, userID, txIDs)
	if err != nil {
		return nil, err
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return result, nil
}

// Wager handles a wager with stake and payout
func (s *WalletService) Wager(userID int, stakeGC, payoutGC, stakeSC, payoutSC int64, idempotencyKey string) ([]*models.Transaction, error) {
	// Validate inputs
	if stakeGC < 0 || payoutGC < 0 || stakeSC < 0 || payoutSC < 0 {
		return nil, fmt.Errorf("amounts cannot be negative")
	}
	if stakeGC == 0 && payoutGC == 0 && stakeSC == 0 && payoutSC == 0 {
		return nil, fmt.Errorf("at least one amount must be greater than zero")
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
	existingTxIDs, err := s.repo.CheckIdempotencyKey(tx, idempotencyKey, userID)
	if err != nil {
		return nil, err
	}
	if len(existingTxIDs) > 0 {
		// Already processed, return existing transactions
		tx.Commit()
		var transactions []*models.Transaction
		for _, txID := range existingTxIDs {
			t, err := s.repo.GetTransaction(txID)
			if err != nil {
				return nil, err
			}
			transactions = append(transactions, t)
		}
		return transactions, nil
	}

	// Track created transactions and their IDs
	var transactions []*models.Transaction
	var txIDs []int

	// Handle Gold Coins stake
	if stakeGC > 0 {
		gcBalance, err := s.repo.GetCurrentBalance(tx, userID, models.CurrencyGC)
		if err != nil {
			return nil, err
		}

		if gcBalance < stakeGC {
			return nil, fmt.Errorf("%w: gold coins - have %d, need %d", ErrInsufficientFunds, gcBalance, stakeGC)
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
			return nil, err
		}
		transactions = append(transactions, wagerTx)
		txIDs = append(txIDs, wagerTx.ID)
	}

	// Handle Gold Coins payout (independent of stake)
	if payoutGC > 0 {
		gcBalance, err := s.repo.GetCurrentBalance(tx, userID, models.CurrencyGC)
		if err != nil {
			return nil, err
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
			return nil, err
		}
		transactions = append(transactions, winTx)
		txIDs = append(txIDs, winTx.ID)
	}

	// Handle Sweeps Coins stake
	if stakeSC > 0 {
		scBalance, err := s.repo.GetCurrentBalance(tx, userID, models.CurrencySC)
		if err != nil {
			return nil, err
		}

		if scBalance < stakeSC {
			return nil, fmt.Errorf("%w: sweeps coins - have %d, need %d", ErrInsufficientFunds, scBalance, stakeSC)
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
			return nil, err
		}
		transactions = append(transactions, wagerTx)
		txIDs = append(txIDs, wagerTx.ID)
	}

	// Handle Sweeps Coins payout (independent of stake)
	if payoutSC > 0 {
		scBalance, err := s.repo.GetCurrentBalance(tx, userID, models.CurrencySC)
		if err != nil {
			return nil, err
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
			return nil, err
		}
		transactions = append(transactions, winTx)
		txIDs = append(txIDs, winTx.ID)
	}

	// Save idempotency key with all transaction IDs
	err = s.repo.SaveIdempotencyKey(tx, idempotencyKey, userID, txIDs)
	if err != nil {
		return nil, err
	}

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return transactions, nil
}

// Redeem handles redeeming Sweeps Coins
func (s *WalletService) Redeem(userID int, amount int64, idempotencyKey string) (*models.Transaction, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("redemption amount must be positive")
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
	existingTxIDs, err := s.repo.CheckIdempotencyKey(tx, idempotencyKey, userID)
	if err != nil {
		return nil, err
	}
	if len(existingTxIDs) > 0 {
		// Already processed, return existing transaction
		tx.Commit()
		return s.repo.GetTransaction(existingTxIDs[0])
	}

	// Get current balance
	scBalance, err := s.repo.GetCurrentBalance(tx, userID, models.CurrencySC)
	if err != nil {
		return nil, err
	}

	if scBalance < amount {
		return nil, fmt.Errorf("%w: sweeps coins - have %d, need %d", ErrInsufficientFunds, scBalance, amount)
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
		return nil, err
	}

	// Save idempotency key
	err = s.repo.SaveIdempotencyKey(tx, idempotencyKey, userID, []int{redeemTx.ID})
	if err != nil {
		return nil, err
	}

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return redeemTx, nil
}
