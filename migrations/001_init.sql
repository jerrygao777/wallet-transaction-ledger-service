-- Create users table
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    -- Wallet balances
    gold_balance BIGINT NOT NULL DEFAULT 0,
    sweeps_balance BIGINT NOT NULL DEFAULT 0,
    -- Wallet statistics
    total_gc_wagered BIGINT NOT NULL DEFAULT 0,
    total_gc_won BIGINT NOT NULL DEFAULT 0,
    total_sc_wagered BIGINT NOT NULL DEFAULT 0,
    total_sc_won BIGINT NOT NULL DEFAULT 0,
    total_sc_redeemed BIGINT NOT NULL DEFAULT 0
);

-- Create transactions table
CREATE TABLE transactions (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id),
    currency VARCHAR(2) NOT NULL CHECK (currency IN ('GC', 'SC')),
    type VARCHAR(20) NOT NULL CHECK (type IN ('purchase', 'wager_gc', 'win_gc', 'wager_sc', 'win_sc', 'redeem_sc')),
    amount BIGINT NOT NULL,
    balance_after BIGINT NOT NULL,
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create idempotency table for purchase deduplication
CREATE TABLE idempotency_keys (
    key VARCHAR(255) PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id),
    transaction_id INTEGER NOT NULL REFERENCES transactions(id),
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX idx_transactions_user_id ON transactions(user_id);
CREATE INDEX idx_transactions_user_currency ON transactions(user_id, currency);
CREATE INDEX idx_transactions_user_created ON transactions(user_id, created_at DESC, id DESC);
CREATE INDEX idx_transactions_type ON transactions(type);
CREATE INDEX idx_idempotency_created_at ON idempotency_keys(created_at);

-- Insert sample users for testing
INSERT INTO users (username) VALUES 
    ('alice'),
    ('bob'),
    ('charlie');
