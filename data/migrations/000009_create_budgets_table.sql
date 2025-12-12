-- +goose Up
-- +goose StatementBegin
-- Create ENUM type for amount type
CREATE TYPE budget_amount_type_enum AS ENUM ('fixed', 'percentage');

CREATE TABLE IF NOT EXISTS budgets (
    id SERIAL PRIMARY KEY,
    account_id INTEGER NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    budget_plan_id INTEGER NOT NULL REFERENCES budget_plans(id) ON DELETE CASCADE,
    category_id INTEGER REFERENCES categories(id) ON DELETE SET NULL,
    amount_type budget_amount_type_enum NOT NULL,
    amount INTEGER NOT NULL, -- cents if fixed, percentage (0-10000 for 0.00-100.00%) if percentage
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    -- Unique constraint: one budget per category per budget plan
    UNIQUE(budget_plan_id, category_id)
);

-- Indexes for faster lookups
CREATE INDEX IF NOT EXISTS idx_budgets_account_id ON budgets(account_id);
CREATE INDEX IF NOT EXISTS idx_budgets_budget_plan_id ON budgets(budget_plan_id);
CREATE INDEX IF NOT EXISTS idx_budgets_category_id ON budgets(category_id);

-- Trigger for updated_at (idempotent)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_budgets_updated_at') THEN
        CREATE TRIGGER update_budgets_updated_at 
            BEFORE UPDATE ON budgets 
            FOR EACH ROW 
            EXECUTE FUNCTION update_updated_at_column();
    END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS update_budgets_updated_at ON budgets;
DROP INDEX IF EXISTS idx_budgets_category_id;
DROP INDEX IF EXISTS idx_budgets_budget_plan_id;
DROP INDEX IF EXISTS idx_budgets_account_id;
DROP TABLE IF EXISTS budgets;
DROP TYPE IF EXISTS budget_amount_type_enum;
-- +goose StatementEnd

