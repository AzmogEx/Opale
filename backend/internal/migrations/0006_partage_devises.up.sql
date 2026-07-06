-- Opale — migration 0006 : espace partagé (EF-007) et multi-devises (EF-008).

-- ── Espaces partagés (EF-007) ─────────────────────────────────────────────────
-- Un espace regroupe des profils du même foyer ; seules les transactions
-- EXPLICITEMENT marquées « communes » y deviennent visibles entre membres.
CREATE TABLE spaces (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL CHECK (length(trim(name)) > 0),
    created_by  UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE space_members (
    space_id    UUID NOT NULL REFERENCES spaces(id) ON DELETE CASCADE,
    profile_id  UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    joined_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (space_id, profile_id)
);
CREATE INDEX idx_space_members_profile ON space_members(profile_id);

-- Une transaction peut être rattachée à un espace (= dépense commune).
ALTER TABLE transactions
    ADD COLUMN space_id UUID REFERENCES spaces(id) ON DELETE SET NULL;
CREATE INDEX idx_transactions_space ON transactions(space_id) WHERE space_id IS NOT NULL;

-- ── Taux de change (EF-008) ───────────────────────────────────────────────────
-- 1 unité de devise = rate_micro micro-euros (1 EUR = 1 000 000).
-- Entiers uniquement (ENF-007) ; saisie manuelle — aucune API externe par
-- défaut (confidentialité d'abord, ENF-005).
CREATE TABLE fx_rates (
    currency    TEXT PRIMARY KEY CHECK (currency ~ '^[A-Z]{3}$' AND currency <> 'EUR'),
    rate_micro  BIGINT NOT NULL CHECK (rate_micro > 0),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
