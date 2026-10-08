-- These constraints are also part of 0012. Keep them when rolling back this
-- reinforcement; 0012 down owns their removal. Pending dates remain nullable
-- because filling missing dates would fabricate historical observations.
SELECT 1;
