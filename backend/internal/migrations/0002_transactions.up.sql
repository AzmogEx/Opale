-- Opale — migration 0002 : le quotidien (palier P3).
-- Catégories, transactions et règles d'apprentissage de la catégorisation.
-- Règle d'or inchangée : montants ENTIERS en centimes (BIGINT), signés
-- (+ revenu, − dépense). Jamais de float.

-- ── Catégories (EF-022) ───────────────────────────────────────────────────────
-- profile_id NULL = catégorie par défaut, visible par tous les profils.
-- parent_id prépare les sous-catégories (cahier des charges §9) — plat pour P3.
CREATE TABLE categories (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id  UUID REFERENCES profiles(id) ON DELETE CASCADE,
    parent_id   UUID REFERENCES categories(id) ON DELETE CASCADE,
    name        TEXT NOT NULL CHECK (length(trim(name)) > 0),
    icon        TEXT NOT NULL DEFAULT 'tag',          -- nom de symbole SF
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE NULLS NOT DISTINCT (profile_id, name)
);
CREATE INDEX idx_categories_profile ON categories(profile_id);

-- Catégories par défaut (globales — profile_id NULL).
INSERT INTO categories (name, icon) VALUES
    ('Revenus',      'arrow.down.circle'),
    ('Courses',      'cart'),
    ('Restaurants',  'fork.knife'),
    ('Transport',    'car'),
    ('Logement',     'house'),
    ('Abonnements',  'repeat'),
    ('Santé',        'cross.case'),
    ('Loisirs',      'gamecontroller'),
    ('Shopping',     'bag'),
    ('Voyages',      'airplane'),
    ('Impôts',       'building.columns'),
    ('Virements',    'arrow.left.arrow.right'),
    ('Autres',       'square.grid.2x2');

-- ── Transactions (EF-020/021) ─────────────────────────────────────────────────
CREATE TABLE transactions (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id   UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    asset_id     UUID NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    amount_cents BIGINT NOT NULL,                     -- signé : + revenu, − dépense
    occurred_on  DATE NOT NULL,
    label        TEXT NOT NULL,                       -- libellé lisible (nettoyé)
    raw_label    TEXT NOT NULL DEFAULT '',            -- libellé bancaire brut
    merchant_key TEXT NOT NULL DEFAULT '',            -- clé marchand normalisée (apprentissage)
    category_id  UUID REFERENCES categories(id) ON DELETE SET NULL,
    note         TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_transactions_profile_date ON transactions(profile_id, occurred_on DESC);
CREATE INDEX idx_transactions_asset ON transactions(asset_id);
CREATE INDEX idx_transactions_category ON transactions(category_id);
CREATE INDEX idx_transactions_merchant ON transactions(profile_id, merchant_key);
CREATE TRIGGER trg_transactions_updated BEFORE UPDATE ON transactions
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ── Règles marchands (EF-022 : apprentissage des corrections) ────────────────
-- Quand l'utilisateur corrige la catégorie d'une transaction, on mémorise
-- « clé marchand normalisée → catégorie » pour ce profil. Ces règles priment
-- sur les règles par mots-clés du catégoriseur.
CREATE TABLE merchant_rules (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id   UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    merchant_key TEXT NOT NULL CHECK (length(merchant_key) > 0),
    category_id  UUID NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (profile_id, merchant_key)
);
