-- +goose Up
-- +goose StatementBegin
-- Create ENUM types for recurrence units and transaction types
CREATE TYPE recurrence_unit_enum AS ENUM ('week', 'month', 'year');
CREATE TYPE transaction_type_enum AS ENUM ('expense', 'income');

CREATE TABLE IF NOT EXISTS recurring_transactions (
    id SERIAL PRIMARY KEY,
    account_id INTEGER NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    counterparty VARCHAR(255) NOT NULL,
    category_id INTEGER REFERENCES categories(id) ON DELETE SET NULL,
    expected_amount INTEGER NOT NULL CHECK (expected_amount != 0),
    tolerance INTEGER NOT NULL,
    start_date DATE NOT NULL,
    recurrence_unit recurrence_unit_enum NOT NULL,
    recurrence_value INTEGER NOT NULL CHECK (recurrence_value > 0),
    end_date DATE,
    type transaction_type_enum GENERATED ALWAYS AS (
        CASE WHEN expected_amount < 0 THEN 'expense'::transaction_type_enum 
             ELSE 'income'::transaction_type_enum 
        END
    ) STORED,
    archived BOOLEAN GENERATED ALWAYS AS (end_date IS NOT NULL) STORED,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_recurring_transactions_account_id ON recurring_transactions(account_id);
CREATE INDEX IF NOT EXISTS idx_recurring_transactions_counterparty ON recurring_transactions(counterparty);
CREATE INDEX IF NOT EXISTS idx_recurring_transactions_type ON recurring_transactions(type);

-- Trigger for updated_at (idempotent)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_recurring_transactions_updated_at') THEN
        CREATE TRIGGER update_recurring_transactions_updated_at 
            BEFORE UPDATE ON recurring_transactions 
            FOR EACH ROW 
            EXECUTE FUNCTION update_updated_at_column();
    END IF;
END $$;

-- Note: Views are created in migration 000006_create_recurring_transaction_views.sql
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS update_recurring_transactions_updated_at ON recurring_transactions;
DROP INDEX IF EXISTS idx_recurring_transactions_type;
DROP INDEX IF EXISTS idx_recurring_transactions_counterparty;
DROP INDEX IF EXISTS idx_recurring_transactions_account_id;
DROP TABLE IF EXISTS recurring_transactions;
DROP TYPE IF EXISTS transaction_type_enum;
DROP TYPE IF EXISTS recurrence_unit_enum;
-- +goose StatementEnd
