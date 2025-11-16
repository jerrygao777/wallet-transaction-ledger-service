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

	"github.com/lib/pq"
)

type Repository struct {
	db *sql.DB
}

func New(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Ping checks if the database connection is alive
func (r *Repository) Ping() error {
	return r.db.Ping()
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

// GetUserWithBalances retrieves a user with their balances calculated from transactions
func (r *Repository) GetUserWithBalances(userID int) (*models.UserWithBalances, error) {
	var result models.UserWithBalances
	err := r.db.QueryRow(`
		SELECT id, username, created_at
		FROM users
		WHERE id = $1
	`, userID).Scan(&result.ID, &result.Username, &result.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, err
	}

	// Calculate balances from transactions
	var gcBalance, scBalance sql.NullInt64
	err = r.db.QueryRow(`
		SELECT 
			COALESCE(SUM(CASE WHEN currency = 'GC' THEN 
				CASE 
					WHEN type IN ('purchase', 'win_gc') THEN amount
					WHEN type = 'wager_gc' THEN -amount
				END
			END), 0) as gc_balance,
			COALESCE(SUM(CASE WHEN currency = 'SC' THEN 
				CASE 
					WHEN type IN ('purchase', 'win_sc') THEN amount
					WHEN type IN ('wager_sc', 'redeem_sc') THEN -amount
				END
			END), 0) as sc_balance
		FROM transactions
		WHERE user_id = $1
	`, userID).Scan(&gcBalance, &scBalance)

	if err != nil {
		return nil, err
	}

	if gcBalance.Valid {
		result.GoldBalance = gcBalance.Int64
	}
	if scBalance.Valid {
		result.SweepsBalance = scBalance.Int64
	}

	// Calculate statistics from transactions
	var gcWagered, gcWon, scWagered, scWon, scRedeemed sql.NullInt64
	err = r.db.QueryRow(`
		SELECT 
			COALESCE(SUM(CASE WHEN type = 'wager_gc' THEN amount END), 0) as gc_wagered,
			COALESCE(SUM(CASE WHEN type = 'win_gc' THEN amount END), 0) as gc_won,
			COALESCE(SUM(CASE WHEN type = 'wager_sc' THEN amount END), 0) as sc_wagered,
			COALESCE(SUM(CASE WHEN type = 'win_sc' THEN amount END), 0) as sc_won,
			COALESCE(SUM(CASE WHEN type = 'redeem_sc' THEN amount END), 0) as sc_redeemed
		FROM transactions
		WHERE user_id = $1
	`, userID).Scan(&gcWagered, &gcWon, &scWagered, &scWon, &scRedeemed)

	if err != nil {
		return nil, err
	}

	if gcWagered.Valid {
		result.TotalGCWagered = gcWagered.Int64
	}
	if gcWon.Valid {
		result.TotalGCWon = gcWon.Int64
	}
	if scWagered.Valid {
		result.TotalSCWagered = scWagered.Int64
	}
	if scWon.Valid {
		result.TotalSCWon = scWon.Int64
	}
	if scRedeemed.Valid {
		result.TotalSCRedeemed = scRedeemed.Int64
	}

	return &result, nil
}

// GetCurrentBalance returns the current balance for a user and currency from transactions
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
	var metadataValue interface{}

	// Only include metadata if it's not nil/empty
	if len(t.Metadata) > 0 {
		metadataBytes, err := json.Marshal(t.Metadata)
		if err != nil {
			return err
		}
		metadataValue = metadataBytes
	} else {
		// Use nil for NULL in database
		metadataValue = nil
	}

	return tx.QueryRow(`
		INSERT INTO transactions (user_id, currency, type, amount, balance_after, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`, t.UserID, t.Currency, t.Type, t.Amount, t.BalanceAfter, metadataValue, time.Now()).
		Scan(&t.ID, &t.CreatedAt)
}

// CheckIdempotencyKey checks if an idempotency key was already used
func (r *Repository) CheckIdempotencyKey(tx *sql.Tx, key string, userID int) ([]int, error) {
	var transactionIDs pq.Int64Array
	err := tx.QueryRow(`
		SELECT transaction_ids 
		FROM idempotency_keys 
		WHERE key = $1 AND user_id = $2
		FOR UPDATE
	`, key, userID).Scan(&transactionIDs)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	// Convert pq.Int64Array to []int
	result := make([]int, len(transactionIDs))
	for i, v := range transactionIDs {
		result[i] = int(v)
	}
	return result, nil
}

// SaveIdempotencyKey saves an idempotency key
func (r *Repository) SaveIdempotencyKey(tx *sql.Tx, key string, userID int, transactionIDs []int) error {
	_, err := tx.Exec(`
		INSERT INTO idempotency_keys (key, user_id, transaction_ids, created_at)
		VALUES ($1, $2, $3, $4)
	`, key, userID, pq.Array(transactionIDs), time.Now())
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
			timeArg := argCount
			argCount++
			idArg := argCount
			// Proper cursor pagination: (timestamp, id) < (cursor_timestamp, cursor_id)
			// This handles cases where multiple transactions have the same timestamp
			query += fmt.Sprintf(" AND (created_at < $%d OR (created_at = $%d AND id < $%d))", timeArg, timeArg, idArg)
			args = append(args, cursorTime, cursorID)
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
	// Use RFC3339Nano to preserve microsecond precision
	cursorData := fmt.Sprintf("%d:%s", id, createdAt.Format(time.RFC3339Nano))
	return base64.URLEncoding.EncodeToString([]byte(cursorData))
}

// decodeCursor decodes a cursor into transaction ID and timestamp
func decodeCursor(cursor string) (int, time.Time, error) {
	data, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return 0, time.Time{}, err
	}

	// Split only on the first colon to separate ID from timestamp (timestamp contains colons)
	parts := strings.SplitN(string(data), ":", 2)
	if len(parts) != 2 {
		return 0, time.Time{}, fmt.Errorf("invalid cursor format")
	}

	id, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, time.Time{}, err
	}

	timestamp, err := time.Parse(time.RFC3339Nano, parts[1])
	if err != nil {
		return 0, time.Time{}, err
	}

	return id, timestamp, nil
}

// CleanupOldIdempotencyKeys removes idempotency keys older than 24 hours
func (r *Repository) CleanupOldIdempotencyKeys() error {
	_, err := r.db.Exec(`
		DELETE FROM idempotency_keys 
		WHERE created_at < NOW() - INTERVAL '24 hours'
	`)
	return err
}
