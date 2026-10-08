-- Opale — migration 0010 : cours automatiques, snapshots mensuels,
-- allocation cible, alertes personnalisables, jetons push.

-- ── Cours automatiques (opt-in) ───────────────────────────────────────────────
-- quote_symbol : identifiant CoinGecko pour les cryptos (ex. « bitcoin »).
-- quote_quantity_micro : quantité détenue en millionièmes d'unité
-- (0,5 BTC = 500000). Valeur auto = quantité × cours — entiers (ENF-007).
ALTER TABLE assets
    ADD COLUMN quote_symbol TEXT NOT NULL DEFAULT '',
    ADD COLUMN quote_quantity_micro BIGINT NOT NULL DEFAULT 0 CHECK (quote_quantity_micro >= 0);

-- ── Snapshots mensuels du patrimoine (serveur) ────────────────────────────────
-- Un point par profil et par mois, pris automatiquement le 1er : la courbe
-- du patrimoine reste fidèle même quand personne n'ouvre l'app.
CREATE TABLE monthly_snapshots (
    profile_id        UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    month             DATE NOT NULL, -- toujours le 1er du mois
    assets_cents      BIGINT NOT NULL,
    liabilities_cents BIGINT NOT NULL,
    net_worth_cents   BIGINT NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (profile_id, month)
);

-- ── Allocation cible du portefeuille ──────────────────────────────────────────
-- Cible en points de base par classe d'actifs ; la dérive se calcule contre
-- la répartition réelle des valorisations.
CREATE TABLE allocation_targets (
    profile_id  UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    class       TEXT NOT NULL CHECK (class IN ('stocks', 'real_estate', 'crypto', 'cash', 'other')),
    target_bps  INTEGER NOT NULL CHECK (target_bps >= 0 AND target_bps <= 10000),
    PRIMARY KEY (profile_id, class)
);

-- ── Alertes personnalisables ──────────────────────────────────────────────────
CREATE TABLE custom_alerts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id      UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    kind            TEXT NOT NULL CHECK (kind IN ('cash_below', 'net_worth_below', 'expenses_month_above')),
    threshold_cents BIGINT NOT NULL,
    enabled         BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_custom_alerts_profile ON custom_alerts(profile_id);

-- ── Jetons de notification push (APNs) ────────────────────────────────────────
CREATE TABLE push_tokens (
    token      TEXT PRIMARY KEY,
    profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    platform   TEXT NOT NULL DEFAULT 'ios',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_push_tokens_profile ON push_tokens(profile_id);
