-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS financial_accounts (
    id SERIAL PRIMARY KEY,
    account_id INTEGER NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL CHECK (type IN ('checking', 'savings', 'credit_card')),
    balance INTEGER NOT NULL DEFAULT 0, -- stored in cents
    csv_date_field VARCHAR(255) NOT NULL,
    csv_payee_field VARCHAR(255) NOT NULL,
    csv_expense_field VARCHAR(255) NOT NULL,
    csv_income_field VARCHAR(255) NOT NULL,
    csv_category_field VARCHAR(255), -- optional
    csv_balance_field VARCHAR(255), -- optional
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_financial_accounts_account_id ON financial_accounts(account_id);
CREATE INDEX IF NOT EXISTS idx_financial_accounts_type ON financial_accounts(type);

-- Trigger for updated_at (idempotent)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_financial_accounts_updated_at') THEN
        CREATE TRIGGER update_financial_accounts_updated_at 
            BEFORE UPDATE ON financial_accounts 
            FOR EACH ROW 
            EXECUTE FUNCTION update_updated_at_column();
    END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS update_financial_accounts_updated_at ON financial_accounts;
DROP INDEX IF EXISTS idx_financial_accounts_type;
DROP INDEX IF EXISTS idx_financial_accounts_account_id;
DROP TABLE IF EXISTS financial_accounts;
-- +goose StatementEnd

