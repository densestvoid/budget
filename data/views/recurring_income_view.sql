-- View that filters recurring_transactions_view for income only
CREATE OR REPLACE VIEW recurring_income_view AS
SELECT 
    rtv.*
FROM recurring_transactions_view rtv
INNER JOIN recurring_transactions rt ON rtv.recurring_transaction_id = rt.id
WHERE rt.type = 'income';

