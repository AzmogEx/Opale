-- Opale — migration 0001 : socle du modèle de données (palier P0).
-- Couvre le cœur "patrimoine" : profils, sessions, actifs, passifs, valorisations.
-- Règle d'or : tous les montants sont des ENTIERS en centimes (BIGINT) — jamais de float.

-- Extension pour gen_random_uuid() (natif PostgreSQL >= 13 via pgcrypto).
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Fonction utilitaire : met à jour automatiquement updated_at.
CREATE OR REPLACE FUNCTION set_updated_at() RETURNS trigger AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ── Profils (EF-001) ──────────────────────────────────────────────────────────
-- Multi-profil léger : chaque profil est une cloison de données privée.
CREATE TABLE profiles (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT NOT NULL CHECK (length(trim(name)) > 0),
    pin_hash        TEXT NOT NULL,                       -- bcrypt du code/PIN (EF-002)
    privacy_default TEXT NOT NULL DEFAULT 'N1'           -- niveau de confidentialité par défaut (§6.5)
                    CHECK (privacy_default IN ('N1', 'N2', 'N3')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TRIGGER trg_profiles_updated BEFORE UPDATE ON profiles
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ── Sessions (EF-002) ─────────────────────────────────────────────────────────
-- Jetons opaques de session (révocables), liés à un profil. Le jeton n'est
-- stocké que sous forme hachée.
CREATE TABLE sessions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id  UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    token_hash  TEXT NOT NULL UNIQUE,                    -- sha-256 du jeton porteur
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_sessions_profile ON sessions(profile_id);
CREATE INDEX idx_sessions_expires ON sessions(expires_at);

-- ── Actifs (EF-030, EF-072) ───────────────────────────────────────────────────
-- Entité de premier rang : comptes, livrets, placements, immobilier, objets, etc.
CREATE TABLE assets (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id  UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    name        TEXT NOT NULL CHECK (length(trim(name)) > 0),
    kind        TEXT NOT NULL CHECK (kind IN (
                    'checking',        -- compte courant
                    'savings',         -- livret / épargne
                    'life_insurance',  -- assurance-vie
                    'pea',             -- plan d'épargne en actions
                    'cto',             -- compte-titres ordinaire
                    'crypto',
                    'real_estate',     -- immobilier
                    'precious_metal',  -- or, métaux
                    'vehicle',         -- voiture, moto
                    'valuable',        -- montre, œuvre, collection
                    'company_share',   -- parts de société
                    'other'
                )),
    currency    CHAR(3) NOT NULL DEFAULT 'EUR',
    note        TEXT NOT NULL DEFAULT '',
    archived    BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_assets_profile ON assets(profile_id);
CREATE TRIGGER trg_assets_updated BEFORE UPDATE ON assets
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ── Passifs (EF-031) ──────────────────────────────────────────────────────────
CREATE TABLE liabilities (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id  UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    name        TEXT NOT NULL CHECK (length(trim(name)) > 0),
    kind        TEXT NOT NULL CHECK (kind IN (
                    'mortgage',        -- crédit immobilier
                    'auto_loan',       -- crédit auto
                    'consumer_loan',   -- crédit conso
                    'other'
                )),
    currency    CHAR(3) NOT NULL DEFAULT 'EUR',
    note        TEXT NOT NULL DEFAULT '',
    archived    BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_liabilities_profile ON liabilities(profile_id);
CREATE TRIGGER trg_liabilities_updated BEFORE UPDATE ON liabilities
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ── Valorisations (EF-032) ────────────────────────────────────────────────────
-- Snapshot daté de la valeur d'un actif OU d'un passif → courbe du patrimoine net.
-- value_cents : montant en centimes (toujours positif ; le passif est soustrait
-- au moment du calcul du patrimoine net).
CREATE TABLE valuations (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id    UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    asset_id      UUID REFERENCES assets(id) ON DELETE CASCADE,
    liability_id  UUID REFERENCES liabilities(id) ON DELETE CASCADE,
    value_cents   BIGINT NOT NULL CHECK (value_cents >= 0),
    as_of         DATE NOT NULL,
    note          TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- Exactement un sujet : soit un actif, soit un passif.
    CONSTRAINT valuation_one_subject CHECK (
        (asset_id IS NOT NULL AND liability_id IS NULL) OR
        (asset_id IS NULL AND liability_id IS NOT NULL)
    )
);
CREATE INDEX idx_valuations_asset ON valuations(asset_id, as_of DESC);
CREATE INDEX idx_valuations_liability ON valuations(liability_id, as_of DESC);
CREATE INDEX idx_valuations_profile ON valuations(profile_id, as_of DESC);
