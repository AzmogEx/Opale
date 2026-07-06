-- Opale — migration 0005 : le confort (palier P7).
-- Module entrepreneur (EF-036) et synchro bancaire GoCardless (EF-071).

-- ── Détails entreprise (EF-036) ───────────────────────────────────────────────
-- Extension 1-1 d'un actif company_share. La valorisation de la SOCIÉTÉ
-- entière vit dans les valuations de l'actif ; la part détenue (bps) et le
-- compte courant d'associé permettent au moteur de calculer « ma part ».
CREATE TABLE company_details (
    asset_id               UUID PRIMARY KEY REFERENCES assets(id) ON DELETE CASCADE,
    profile_id             UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    siren                  TEXT NOT NULL DEFAULT '',
    ownership_bps          INT  NOT NULL DEFAULT 10000
                           CHECK (ownership_bps > 0 AND ownership_bps <= 10000),
    cca_cents              BIGINT NOT NULL DEFAULT 0 CHECK (cca_cents >= 0),      -- compte courant d'associé
    annual_dividends_cents BIGINT NOT NULL DEFAULT 0 CHECK (annual_dividends_cents >= 0),
    monthly_salary_cents   BIGINT NOT NULL DEFAULT 0 CHECK (monthly_salary_cents >= 0),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_company_profile ON company_details(profile_id);
CREATE TRIGGER trg_company_updated BEFORE UPDATE ON company_details
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ── Liens bancaires GoCardless (EF-071) ───────────────────────────────────────
-- Une réquisition DSP2 par banque connectée ; les mouvements synchronisés
-- atterrissent sur l'actif lié (dédup par l'import P3).
CREATE TABLE bank_links (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id      UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    asset_id        UUID NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    requisition_id  TEXT NOT NULL,
    institution_id  TEXT NOT NULL,
    institution_name TEXT NOT NULL DEFAULT '',
    -- created (lien envoyé) → linked (comptes accessibles)
    status          TEXT NOT NULL DEFAULT 'created' CHECK (status IN ('created', 'linked')),
    last_synced_at  TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_bank_links_profile ON bank_links(profile_id);
