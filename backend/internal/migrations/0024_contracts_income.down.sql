DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM financial_contracts) OR EXISTS(SELECT 1 FROM variable_incomes) THEN
  RAISE EXCEPTION 'Export or explicitly remove contracts and variable incomes before downgrade';
 END IF;
END $$;
DROP TABLE variable_incomes,contract_push_deliveries,contract_price_dismissals,contract_prices,financial_contracts;
