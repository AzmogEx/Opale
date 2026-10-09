CREATE TABLE profile_journey (
 profile_id UUID PRIMARY KEY REFERENCES profiles(id) ON DELETE CASCADE,
 revision BIGINT NOT NULL DEFAULT 1 CHECK(revision>0),
 step INTEGER NOT NULL DEFAULT 0 CHECK(step BETWEEN 0 AND 6),
 reviewed TEXT[] NOT NULL DEFAULT '{}', skipped TEXT[] NOT NULL DEFAULT '{}',
 daily_budget_cents BIGINT CHECK(daily_budget_cents BETWEEN 0 AND 999999999999),
 budget_currency CHAR(3) NOT NULL DEFAULT 'EUR' CHECK(budget_currency IN ('EUR','USD','GBP','CHF','JPY','KWD','CAD','AUD')),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
