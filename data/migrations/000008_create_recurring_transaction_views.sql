-- +goose Up
-- +goose StatementBegin
-- Create base view for recurring transactions (idempotent)
CREATE OR REPLACE VIEW recurring_transactions_view AS
WITH candidate_matches AS (
    SELECT 
        rt.id AS recurring_transaction_id,
        rt.type,
        rt.start_date,
        rt.recurrence_unit,
        rt.recurrence_value,
        rt.end_date,
        rt.expected_amount,
        t.id AS transaction_id,
        t.date AS transaction_date,
        t.amount AS transaction_amount
    FROM recurring_transactions rt
    INNER JOIN transactions t ON rt.account_id = t.account_id
        AND rt.financial_account_id = t.financial_account_id
    WHERE 
        -- Counterparty match (case-insensitive contains)
        LOWER(t.payee) LIKE '%' || LOWER(rt.counterparty) || '%'
        -- Category match (if recurring transaction has category set)
        AND (rt.category_id IS NULL OR t.category_id = rt.category_id)
        -- Amount within tolerance
        -- Expenses: expected_amount is negative, transactions are negative
        -- Income: expected_amount is positive, transactions are positive
        -- Match when the absolute difference is within tolerance
        AND ABS(t.amount - rt.expected_amount) <= rt.tolerance
        -- Type must match: expenses match negative transactions, income matches positive transactions
        AND (
            (rt.type = 'expense' AND t.amount < 0) OR
            (rt.type = 'income' AND t.amount > 0)
        )
        -- Transaction must be on or after start_date
        AND t.date >= rt.start_date
        -- If recurring transaction is archived (end_date IS NOT NULL), transaction must be on or before end_date
        AND (rt.end_date IS NULL OR t.date <= rt.end_date)
),
occurrence_periods AS (
    -- For each candidate match, calculate which occurrence period the transaction falls into
    SELECT 
        cm.recurring_transaction_id,
        cm.transaction_id,
        cm.transaction_date,
        cm.start_date,
        cm.recurrence_unit,
        cm.recurrence_value,
        cm.end_date,
        -- Calculate the occurrence number (0-based) that this transaction might belong to
        -- For weekly: number of weeks since start_date
        -- For monthly: number of months since start_date  
        -- For yearly: number of years since start_date
        CASE 
            WHEN cm.recurrence_unit = 'week' THEN
                FLOOR(EXTRACT(EPOCH FROM (cm.transaction_date::TIMESTAMP - cm.start_date::TIMESTAMP)) / (7.0 * 86400.0) / cm.recurrence_value::NUMERIC)::INTEGER
            WHEN cm.recurrence_unit = 'month' THEN
                FLOOR(((EXTRACT(YEAR FROM cm.transaction_date)::NUMERIC - EXTRACT(YEAR FROM cm.start_date)::NUMERIC) * 12.0 
                 + EXTRACT(MONTH FROM cm.transaction_date)::NUMERIC - EXTRACT(MONTH FROM cm.start_date)::NUMERIC) / cm.recurrence_value::NUMERIC)::INTEGER
            WHEN cm.recurrence_unit = 'year' THEN
                FLOOR((EXTRACT(YEAR FROM cm.transaction_date)::NUMERIC - EXTRACT(YEAR FROM cm.start_date)::NUMERIC) / cm.recurrence_value::NUMERIC)::INTEGER
            ELSE 0
        END AS occurrence_num,
        -- Calculate the expected date for this occurrence period
        -- Use the occurrence_num to calculate period_start
        (CASE 
            WHEN cm.recurrence_unit = 'week' THEN
                cm.start_date + (INTERVAL '1 week' * (FLOOR(EXTRACT(EPOCH FROM (cm.transaction_date::TIMESTAMP - cm.start_date::TIMESTAMP)) / (7.0 * 86400.0) / cm.recurrence_value::NUMERIC)::INTEGER * cm.recurrence_value))
            WHEN cm.recurrence_unit = 'month' THEN
                -- Add the calculated number of months to start_date
                cm.start_date + (
                    INTERVAL '1 month' * (FLOOR(((EXTRACT(YEAR FROM cm.transaction_date)::NUMERIC - EXTRACT(YEAR FROM cm.start_date)::NUMERIC) * 12.0 
                     + EXTRACT(MONTH FROM cm.transaction_date)::NUMERIC - EXTRACT(MONTH FROM cm.start_date)::NUMERIC) / cm.recurrence_value::NUMERIC)::INTEGER * cm.recurrence_value)
                )
            WHEN cm.recurrence_unit = 'year' THEN
                -- Add the calculated number of years to start_date
                cm.start_date + (
                    INTERVAL '1 year' * (FLOOR((EXTRACT(YEAR FROM cm.transaction_date)::NUMERIC - EXTRACT(YEAR FROM cm.start_date)::NUMERIC) / cm.recurrence_value::NUMERIC)::INTEGER * cm.recurrence_value)
                )
            ELSE cm.start_date
        END)::DATE AS period_start,
        -- Calculate the next expected date (end of period)
        (CASE 
            WHEN cm.recurrence_unit = 'week' THEN
                cm.start_date + (INTERVAL '1 week' * ((FLOOR(EXTRACT(EPOCH FROM (cm.transaction_date::TIMESTAMP - cm.start_date::TIMESTAMP)) / (7.0 * 86400.0) / cm.recurrence_value::NUMERIC)::INTEGER + 1) * cm.recurrence_value))
            WHEN cm.recurrence_unit = 'month' THEN
                cm.start_date + (
                    INTERVAL '1 month' * ((FLOOR(((EXTRACT(YEAR FROM cm.transaction_date)::NUMERIC - EXTRACT(YEAR FROM cm.start_date)::NUMERIC) * 12.0 
                      + EXTRACT(MONTH FROM cm.transaction_date)::NUMERIC - EXTRACT(MONTH FROM cm.start_date)::NUMERIC) / cm.recurrence_value::NUMERIC)::INTEGER + 1) * cm.recurrence_value)
                )
            WHEN cm.recurrence_unit = 'year' THEN
                cm.start_date + (
                    INTERVAL '1 year' * ((FLOOR((EXTRACT(YEAR FROM cm.transaction_date)::NUMERIC - EXTRACT(YEAR FROM cm.start_date)::NUMERIC) / cm.recurrence_value::NUMERIC)::INTEGER + 1) * cm.recurrence_value)
                )
            ELSE cm.start_date + INTERVAL '1 day'
        END)::DATE AS period_end
    FROM candidate_matches cm
),
valid_period_matches AS (
    -- Filter to only valid period matches where transaction falls within the period
    SELECT 
        op.recurring_transaction_id,
        op.transaction_id,
        op.transaction_date,
        op.period_start,
        EXTRACT(YEAR FROM op.period_start)::INTEGER AS year,
        EXTRACT(MONTH FROM op.period_start)::INTEGER AS month
    FROM occurrence_periods op
    WHERE 
        -- Transaction must occur on or after the period start
        op.transaction_date >= op.period_start
        -- And before the period end (exclusive)
        AND op.transaction_date < op.period_end
        -- Period start must be on or after recurring transaction start_date (sanity check)
        AND op.period_start >= op.start_date
        -- If recurring transaction is archived, period start must be on or before end_date
        AND (op.end_date IS NULL OR op.period_start <= op.end_date)
),
ranked_matches AS (
    -- Rank all matches: for each recurring transaction, rank transactions by preference (date/ID)
    -- Also rank recurring transactions for each transaction (by recurring_transaction_id for deterministic ordering)
    SELECT 
        vpm.*,
        ROW_NUMBER() OVER (
            PARTITION BY vpm.recurring_transaction_id, vpm.year, vpm.month 
            ORDER BY vpm.transaction_date ASC, vpm.transaction_id ASC
        ) AS recurring_transaction_preference_rank,
        ROW_NUMBER() OVER (
            PARTITION BY vpm.transaction_id 
            ORDER BY vpm.recurring_transaction_id ASC, vpm.transaction_date ASC, vpm.transaction_id ASC
        ) AS transaction_preference_rank
    FROM valid_period_matches vpm
),
greedy_assignments AS (
    -- Greedy assignment: process recurring transactions in order, assign each its first available transaction
    -- A transaction is "available" if it hasn't been assigned to a recurring transaction with a lower ID
    SELECT 
        rm.recurring_transaction_id,
        rm.transaction_id,
        rm.transaction_date,
        rm.year,
        rm.month,
        rm.recurring_transaction_preference_rank,
        -- Check if this transaction is already "claimed" by a recurring transaction with lower ID
        -- A transaction is claimed if there's a recurring transaction with lower ID that has this as its first preference
        CASE 
            WHEN EXISTS (
                SELECT 1 
                FROM ranked_matches rm2 
                WHERE rm2.transaction_id = rm.transaction_id
                AND rm2.recurring_transaction_id < rm.recurring_transaction_id
                AND rm2.recurring_transaction_preference_rank = 1
            ) THEN true
            ELSE false
        END AS transaction_claimed
    FROM ranked_matches rm
)
-- Final selection: each recurring transaction gets its first unclaimed transaction
SELECT DISTINCT ON (ga.recurring_transaction_id, ga.year, ga.month)
    ga.recurring_transaction_id,
    ga.transaction_id,
    ga.transaction_date,
    ga.year,
    ga.month
FROM greedy_assignments ga
WHERE NOT ga.transaction_claimed
ORDER BY ga.recurring_transaction_id ASC, ga.year ASC, ga.month ASC, ga.recurring_transaction_preference_rank ASC;

-- Create expense view (depends on recurring_transactions_view)
CREATE OR REPLACE VIEW recurring_expenses_view AS
SELECT 
    rtv.*
FROM recurring_transactions_view rtv
INNER JOIN recurring_transactions rt ON rtv.recurring_transaction_id = rt.id
WHERE rt.type = 'expense';

-- Create income view (depends on recurring_transactions_view)
CREATE OR REPLACE VIEW recurring_income_view AS
SELECT 
    rtv.*
FROM recurring_transactions_view rtv
INNER JOIN recurring_transactions rt ON rtv.recurring_transaction_id = rt.id
WHERE rt.type = 'income';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP VIEW IF EXISTS recurring_income_view CASCADE;
DROP VIEW IF EXISTS recurring_expenses_view CASCADE;
DROP VIEW IF EXISTS recurring_transactions_view CASCADE;
-- +goose StatementEnd



