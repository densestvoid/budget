-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS rules (
    id SERIAL PRIMARY KEY,
    account_id INTEGER NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    new_payee VARCHAR(255),
    category_id INTEGER REFERENCES categories(id) ON DELETE SET NULL,
    priority INTEGER NOT NULL DEFAULT 0,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS rule_conditions (
    id SERIAL PRIMARY KEY,
    rule_id INTEGER NOT NULL REFERENCES rules(id) ON DELETE CASCADE,
    field VARCHAR(50) NOT NULL, -- 'payee' for now, could be extended to 'amount', 'date', etc.
    operator VARCHAR(20) NOT NULL, -- 'equals', 'contains', 'begins', 'ends'
    value VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_rules_account_id ON rules(account_id);
CREATE INDEX IF NOT EXISTS idx_rules_priority ON rules(priority);
CREATE INDEX IF NOT EXISTS idx_rules_active ON rules(active);
CREATE INDEX IF NOT EXISTS idx_rule_conditions_rule_id ON rule_conditions(rule_id);

CREATE TRIGGER update_rules_updated_at 
    BEFORE UPDATE ON rules 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS update_rules_updated_at ON rules;
DROP INDEX IF EXISTS idx_rule_conditions_rule_id;
DROP INDEX IF EXISTS idx_rules_active;
DROP INDEX IF EXISTS idx_rules_priority;
DROP INDEX IF EXISTS idx_rules_account_id;
DROP TABLE IF EXISTS rule_conditions;
DROP TABLE IF EXISTS rules;
-- +goose StatementEnd 