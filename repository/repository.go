package repository

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
	"wallet-ledger/models"
)

type Repository struct {
	db *sql.DB
}

func New(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// BeginTx starts a new database transaction
func (r *Repository) BeginTx() (*sql.Tx, error) {
	return r.db.Begin()
}

// GetUser retrieves a user by ID
func (r *Repository) GetUser(userID int) (*models.User, error) {
	var user models.User
	err := r.db.QueryRow(`
		SELECT id, username, created_at 
		FROM users 
		WHERE id = $1
	`, userID).Scan(&user.ID, &user.Username, &user.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserWithBalances retrieves a user with their balances and stats
func (r *Repository) GetUserWithBalances(userID int) (*models.UserWithBalances, error) {
	user, err := r.GetUser(userID)
	if err != nil {
		return nil, err
	}

	result := &models.UserWithBalances{
		User: *user,
	}

	// Calculate Gold Coin balance
	err = r.db.QueryRow(`
		SELECT COALESCE(balance_after, 0)
		FROM transactions
		WHERE user_id = $1 AND currency = 'GC'
		ORDER BY id DESC
		LIMIT 1
	`, userID).Scan(&result.GoldBalance)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Calculate Sweeps Coin balance
	err = r.db.QueryRow(`
		SELECT COALESCE(balance_after, 0)
		FROM transactions
		WHERE user_id = $1 AND currency = 'SC'
		ORDER BY id DESC
		LIMIT 1
	`, userID).Scan(&result.SweepsBalance)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Calculate stats
	err = r.db.QueryRow(`
		SELECT 
			COALESCE(SUM(CASE WHEN type = 'wager_gc' THEN amount ELSE 0 END), 0) as total_gc_wagered,
			COALESCE(SUM(CASE WHEN type = 'win_gc' THEN amount ELSE 0 END), 0) as total_gc_won,
			COALESCE(SUM(CASE WHEN type = 'wager_sc' THEN amount ELSE 0 END), 0) as total_sc_wagered,
			COALESCE(SUM(CASE WHEN type = 'win_sc' THEN amount ELSE 0 END), 0) as total_sc_won,
			COALESCE(SUM(CASE WHEN type = 'redeem_sc' THEN amount ELSE 0 END), 0) as total_sc_redeemed
		FROM transactions
		WHERE user_id = $1
	`, userID).Scan(
		&result.TotalGCWagered,
		&result.TotalGCWon,
		&result.TotalSCWagered,
		&result.TotalSCWon,
		&result.TotalSCRedeemed,
	)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// GetCurrentBalance returns the current balance for a user and currency
func (r *Repository) GetCurrentBalance(tx *sql.Tx, userID int, currency models.Currency) (int64, error) {
	var balance int64
	query := `
		SELECT COALESCE(balance_after, 0)
		FROM transactions
		WHERE user_id = $1 AND currency = $2
		ORDER BY id DESC
		LIMIT 1
		FOR UPDATE
	`

	err := tx.QueryRow(query, userID, currency).Scan(&balance)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return balance, err
}

// CreateTransaction creates a new transaction record
func (r *Repository) CreateTransaction(tx *sql.Tx, t *models.Transaction) error {
	metadataBytes, err := json.Marshal(t.Metadata)
	if err != nil {
		return err
	}

	return tx.QueryRow(`
		INSERT INTO transactions (user_id, currency, type, amount, balance_after, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`, t.UserID, t.Currency, t.Type, t.Amount, t.BalanceAfter, metadataBytes, time.Now()).
		Scan(&t.ID, &t.CreatedAt)
}

// CheckIdempotencyKey checks if an idempotency key was already used
func (r *Repository) CheckIdempotencyKey(tx *sql.Tx, key string, userID int) (*int, error) {
	var transactionID int
	err := tx.QueryRow(`
		SELECT transaction_id 
		FROM idempotency_keys 
		WHERE key = $1 AND user_id = $2
	`, key, userID).Scan(&transactionID)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &transactionID, nil
}

// SaveIdempotencyKey saves an idempotency key
func (r *Repository) SaveIdempotencyKey(tx *sql.Tx, key string, userID int, transactionID int) error {
	_, err := tx.Exec(`
		INSERT INTO idempotency_keys (key, user_id, transaction_id, created_at)
		VALUES ($1, $2, $3, $4)
	`, key, userID, transactionID, time.Now())
	return err
}

// GetTransaction retrieves a transaction by ID
func (r *Repository) GetTransaction(transactionID int) (*models.Transaction, error) {
	var t models.Transaction
	var metadataBytes []byte

	err := r.db.QueryRow(`
		SELECT id, user_id, currency, type, amount, balance_after, metadata, created_at
		FROM transactions
		WHERE id = $1
	`, transactionID).Scan(
		&t.ID, &t.UserID, &t.Currency, &t.Type, &t.Amount, &t.BalanceAfter, &metadataBytes, &t.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	if len(metadataBytes) > 0 {
		t.Metadata = json.RawMessage(metadataBytes)
	}

	return &t, nil
}

// ListTransactions retrieves paginated transactions for a user with optional filters
func (r *Repository) ListTransactions(userID int, cursor *string, limit int, txType *models.TransactionType, currency *models.Currency) (*models.TransactionList, error) {
	// Build query
	query := `SELECT id, user_id, currency, type, amount, balance_after, metadata, created_at FROM transactions WHERE user_id = $1`
	args := []interface{}{userID}
	argCount := 1

	// Apply filters
	if txType != nil {
		argCount++
		query += fmt.Sprintf(" AND type = $%d", argCount)
		args = append(args, *txType)
	}
	if currency != nil {
		argCount++
		query += fmt.Sprintf(" AND currency = $%d", argCount)
		args = append(args, *currency)
	}

	// Apply cursor pagination
	if cursor != nil && *cursor != "" {
		cursorID, cursorTime, err := decodeCursor(*cursor)
		if err == nil {
			argCount++
			query += fmt.Sprintf(" AND (created_at, id) < ($%d, $%d)", argCount, argCount+1)
			args = append(args, cursorTime, cursorID)
			argCount++
		}
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC, id DESC LIMIT $%d", argCount+1)
	args = append(args, limit+1) // Fetch one extra to determine if there's a next page

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []models.Transaction
	for rows.Next() {
		var t models.Transaction
		var metadataBytes []byte

		err := rows.Scan(&t.ID, &t.UserID, &t.Currency, &t.Type, &t.Amount, &t.BalanceAfter, &metadataBytes, &t.CreatedAt)
		if err != nil {
			return nil, err
		}

		if len(metadataBytes) > 0 {
			t.Metadata = json.RawMessage(metadataBytes)
		}

		transactions = append(transactions, t)
	}

	// Determine next cursor
	var nextCursor *string
	if len(transactions) > limit {
		lastTx := transactions[limit-1]
		cursorStr := encodeCursor(lastTx.ID, lastTx.CreatedAt)
		nextCursor = &cursorStr
		transactions = transactions[:limit]
	}

	return &models.TransactionList{
		Items:      transactions,
		NextCursor: nextCursor,
	}, nil
}

// encodeCursor creates a cursor from transaction ID and timestamp
func encodeCursor(id int, createdAt time.Time) string {
	cursorData := fmt.Sprintf("%d:%d", id, createdAt.Unix())
	return base64.URLEncoding.EncodeToString([]byte(cursorData))
}

// decodeCursor decodes a cursor into transaction ID and timestamp
func decodeCursor(cursor string) (int, time.Time, error) {
	data, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return 0, time.Time{}, err
	}

	parts := strings.Split(string(data), ":")
	if len(parts) != 2 {
		return 0, time.Time{}, fmt.Errorf("invalid cursor format")
	}

	id, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, time.Time{}, err
	}

	timestamp, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return 0, time.Time{}, err
	}

	return id, time.Unix(timestamp, 0), nil
}

// CleanupOldIdempotencyKeys removes idempotency keys older than 24 hours
func (r *Repository) CleanupOldIdempotencyKeys() error {
	_, err := r.db.Exec(`
		DELETE FROM idempotency_keys 
		WHERE created_at < NOW() - INTERVAL '24 hours'
	`)
	return err
}
