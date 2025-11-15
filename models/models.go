package models

import (
	"encoding/json"
	"time"
)

// Currency represents the type of currency
type Currency string

const (
	CurrencyGC Currency = "GC" // Gold Coins
	CurrencySC Currency = "SC" // Sweeps Coins
)

// TransactionType represents the type of transaction
type TransactionType string

const (
	TransactionTypePurchase TransactionType = "purchase"
	TransactionTypeWagerGC  TransactionType = "wager_gc"
	TransactionTypeWinGC    TransactionType = "win_gc"
	TransactionTypeWagerSC  TransactionType = "wager_sc"
	TransactionTypeWinSC    TransactionType = "win_sc"
	TransactionTypeRedeemSC TransactionType = "redeem_sc"
)

// User represents a user account
type User struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
}

// Transaction represents a ledger entry
type Transaction struct {
	ID           int             `json:"id"`
	UserID       int             `json:"user_id"`
	Currency     Currency        `json:"currency"`
	Type         TransactionType `json:"type"`
	Amount       int64           `json:"amount"`
	BalanceAfter int64           `json:"balance_after"`
	Metadata     json.RawMessage `json:"metadata,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
}

// UserWithBalances represents a user with their current balances and stats
type UserWithBalances struct {
	User
	GoldBalance     int64 `json:"gold_balance"`
	SweepsBalance   int64 `json:"sweeps_balance"`
	TotalGCWagered  int64 `json:"total_gc_wagered"`
	TotalGCWon      int64 `json:"total_gc_won"`
	TotalSCWagered  int64 `json:"total_sc_wagered"`
	TotalSCWon      int64 `json:"total_sc_won"`
	TotalSCRedeemed int64 `json:"total_sc_redeemed"`
}

// Package represents a purchasable package
type Package struct {
	Code       string `json:"code"`
	GoldCoins  int64  `json:"gold_coins"`
	SweepCoins int64  `json:"sweep_coins"`
}

// Available packages
var Packages = map[string]Package{
	"starter_10k": {
		Code:       "starter_10k",
		GoldCoins:  10000,
		SweepCoins: 10,
	},
	"grinder_50k": {
		Code:       "grinder_50k",
		GoldCoins:  50000,
		SweepCoins: 50,
	},
	"highroller_250k": {
		Code:       "highroller_250k",
		GoldCoins:  250000,
		SweepCoins: 250,
	},
}

// TransactionList represents a paginated list of transactions
type TransactionList struct {
	Items      []Transaction `json:"items"`
	NextCursor *string       `json:"next_cursor,omitempty"`
}
