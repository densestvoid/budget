-- View that filters recurring_transactions_view for expenses only
CREATE OR REPLACE VIEW recurring_expenses_view AS
SELECT 
    rtv.*
FROM recurring_transactions_view rtv
INNER JOIN recurring_transactions rt ON rtv.recurring_transaction_id = rt.id
WHERE rt.type = 'expense';

