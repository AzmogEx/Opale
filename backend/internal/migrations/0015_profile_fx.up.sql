-- Public reference rates remain shared; manual overrides belong to one profile.
CREATE TABLE profile_fx_history (
 profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
 currency TEXT NOT NULL,
 as_of DATE NOT NULL,
 rate_micro BIGINT NOT NULL CHECK(rate_micro>0),
 source TEXT NOT NULL DEFAULT 'manual',
 updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 PRIMARY KEY(profile_id,currency,as_of),
 CHECK(currency<>'EUR')
);
CREATE FUNCTION amount_eur(amount BIGINT, code TEXT, day DATE, owner UUID) RETURNS BIGINT LANGUAGE plpgsql STABLE AS $$
DECLARE rate BIGINT; exp INTEGER;
BEGIN
 exp:=currency_exponent(code);
 IF code='EUR' THEN RETURN amount; END IF;
 SELECT rate_micro INTO rate FROM profile_fx_history WHERE profile_id=owner AND currency=code AND as_of<=day ORDER BY as_of DESC LIMIT 1;
 IF rate IS NULL THEN RETURN amount_eur(amount,code,day); END IF;
 RETURN round(amount::numeric*rate*100/(power(10::numeric,exp)*1000000))::bigint;
END $$;
CREATE FUNCTION native_from_eur(amount BIGINT, code TEXT, day DATE, owner UUID) RETURNS BIGINT LANGUAGE plpgsql STABLE AS $$
DECLARE rate BIGINT; exp INTEGER;
BEGIN
 exp:=currency_exponent(code);
 IF code='EUR' THEN RETURN amount; END IF;
 SELECT rate_micro INTO rate FROM profile_fx_history WHERE profile_id=owner AND currency=code AND as_of<=day ORDER BY as_of DESC LIMIT 1;
 IF rate IS NULL THEN RETURN native_from_eur(amount,code,day); END IF;
 RETURN round(amount::numeric*power(10::numeric,exp)*1000000/(rate::numeric*100))::bigint;
END $$;
CREATE OR REPLACE VIEW financial_transactions AS
 SELECT t.*, a.currency, amount_eur(t.amount_cents,a.currency,t.occurred_on,t.profile_id) AS eur_cents
 FROM transactions t JOIN assets a ON a.profile_id=t.profile_id AND a.id=t.asset_id
 LEFT JOIN categories c ON c.id=t.category_id
 WHERE t.bank_status='booked' AND t.flow_kind IN ('expense_income','interest','fee') AND COALESCE(c.name,'')<>'Virements';
