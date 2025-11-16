-- Create users table
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
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
    transaction_ids INTEGER[] NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Indexes for performance
-- Composite index for GetCurrentBalance query (most frequently called in every transaction)
CREATE INDEX idx_transactions_user_currency_id ON transactions(user_id, currency, id DESC);

-- Composite index for ListTransactions pagination (covers most common queries)
CREATE INDEX idx_transactions_user_created ON transactions(user_id, created_at DESC, id DESC);

-- Composite indexes for filtered ListTransactions queries
CREATE INDEX idx_transactions_user_type_created ON transactions(user_id, type, created_at DESC, id DESC);
CREATE INDEX idx_transactions_user_currency_created ON transactions(user_id, currency, created_at DESC, id DESC);

-- Index for idempotency key cleanup job
CREATE INDEX idx_idempotency_created_at ON idempotency_keys(created_at);

-- Note: No separate index on user_id alone needed - covered by composite indexes above
-- Note: idempotency_keys.key already has index via PRIMARY KEY
-- Note: idempotency_keys.user_id doesn't need separate index (lookups always by key first)

-- Insert sample users for testing
INSERT INTO users (username) VALUES 
    ('alice'),
    ('bob'),
    ('charlie');
