-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS budget_plans (
    id SERIAL PRIMARY KEY,
    account_id INTEGER NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Unique constraint: only one active plan per account
CREATE UNIQUE INDEX IF NOT EXISTS idx_budget_plans_account_active 
    ON budget_plans(account_id) 
    WHERE is_active = TRUE;

-- Indexes for faster lookups
CREATE INDEX IF NOT EXISTS idx_budget_plans_account_id ON budget_plans(account_id);
CREATE INDEX IF NOT EXISTS idx_budget_plans_is_active ON budget_plans(is_active);

-- Trigger for updated_at (idempotent)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_budget_plans_updated_at') THEN
        CREATE TRIGGER update_budget_plans_updated_at 
            BEFORE UPDATE ON budget_plans 
            FOR EACH ROW 
            EXECUTE FUNCTION update_updated_at_column();
    END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS update_budget_plans_updated_at ON budget_plans;
DROP INDEX IF EXISTS idx_budget_plans_is_active;
DROP INDEX IF EXISTS idx_budget_plans_account_id;
DROP INDEX IF EXISTS idx_budget_plans_account_active;
DROP TABLE IF EXISTS budget_plans;
-- +goose StatementEnd



