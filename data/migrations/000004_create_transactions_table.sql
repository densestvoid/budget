-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS transactions (
    id SERIAL PRIMARY KEY,
    account_id INTEGER NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    financial_account_id INTEGER NOT NULL REFERENCES financial_accounts(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    original_payee VARCHAR(255) NOT NULL, -- value from import, used for deduplication
    payee VARCHAR(255) NOT NULL,          -- user-override, can be edited for clarity
    category_id INTEGER REFERENCES categories(id) ON DELETE SET NULL,
    amount INTEGER NOT NULL, -- store as cents
    reviewed BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_transactions_account_id ON transactions(account_id);
CREATE INDEX IF NOT EXISTS idx_transactions_financial_account_id ON transactions(financial_account_id);
CREATE INDEX IF NOT EXISTS idx_transactions_category_id ON transactions(category_id);
CREATE INDEX IF NOT EXISTS idx_transactions_date ON transactions(date);

-- Trigger for updated_at (idempotent)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_transactions_updated_at') THEN
        CREATE TRIGGER update_transactions_updated_at 
            BEFORE UPDATE ON transactions 
            FOR EACH ROW 
            EXECUTE FUNCTION update_updated_at_column();
    END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS update_transactions_updated_at ON transactions;
DROP INDEX IF EXISTS idx_transactions_date;
DROP INDEX IF EXISTS idx_transactions_category_id;
DROP INDEX IF EXISTS idx_transactions_financial_account_id;
DROP INDEX IF EXISTS idx_transactions_account_id;
DROP TABLE IF EXISTS transactions;
-- +goose StatementEnd 