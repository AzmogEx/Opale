-- Opale — migration 0003 : le pilotage (palier P4).
-- Enveloppes budgétaires (EF-028) et objectifs de vie (EF-042).

-- ── Enveloppes (EF-028) ───────────────────────────────────────────────────────
-- Une enveloppe = un budget mensuel alloué à une catégorie. Le « dépensé »
-- n'est jamais stocké : il est recalculé depuis les transactions du mois
-- (source de vérité unique, déterministe).
CREATE TABLE envelopes (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id           UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    category_id          UUID NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    monthly_budget_cents BIGINT NOT NULL CHECK (monthly_budget_cents > 0),
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (profile_id, category_id)
);
CREATE INDEX idx_envelopes_profile ON envelopes(profile_id);
CREATE TRIGGER trg_envelopes_updated BEFORE UPDATE ON envelopes
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ── Objectifs (EF-042) ────────────────────────────────────────────────────────
-- Un objectif de vie : montant cible, échéance optionnelle, et un actif
-- « source » optionnel dont la dernière valorisation mesure la progression
-- (ex. objectif « Apport immobilier » suivi sur le Livret A).
CREATE TABLE goals (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id   UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    name         TEXT NOT NULL CHECK (length(trim(name)) > 0),
    icon         TEXT NOT NULL DEFAULT 'target',
    target_cents BIGINT NOT NULL CHECK (target_cents > 0),
    target_date  DATE,
    asset_id     UUID REFERENCES assets(id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_goals_profile ON goals(profile_id);
CREATE TRIGGER trg_goals_updated BEFORE UPDATE ON goals
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
