-- Opale — migration 0004 : la profondeur (palier P6).
-- Centre immobilier (EF-033), objets de valeur (EF-035), coffre-fort (EF-064)
-- et plan de transmission (EF-063). Montants : ENTIERS en centimes (BIGINT).

-- ── Détails immobiliers (EF-033) ──────────────────────────────────────────────
-- Extension 1-1 d'un actif real_estate : de quoi calculer rendement,
-- cashflow et plus-value latente (calculs faits par le moteur, jamais ici).
CREATE TABLE property_details (
    asset_id                   UUID PRIMARY KEY REFERENCES assets(id) ON DELETE CASCADE,
    profile_id                 UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    purchase_price_cents       BIGINT NOT NULL DEFAULT 0 CHECK (purchase_price_cents >= 0),
    purchase_date              DATE,
    monthly_rent_cents         BIGINT NOT NULL DEFAULT 0 CHECK (monthly_rent_cents >= 0),
    monthly_charges_cents      BIGINT NOT NULL DEFAULT 0 CHECK (monthly_charges_cents >= 0),
    property_tax_yearly_cents  BIGINT NOT NULL DEFAULT 0 CHECK (property_tax_yearly_cents >= 0),
    -- Crédit adossé au bien (mensualité saisie ; le restant dû vient des
    -- valorisations du passif lié).
    liability_id               UUID REFERENCES liabilities(id) ON DELETE SET NULL,
    monthly_loan_payment_cents BIGINT NOT NULL DEFAULT 0 CHECK (monthly_loan_payment_cents >= 0),
    updated_at                 TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_property_profile ON property_details(profile_id);
CREATE TRIGGER trg_property_updated BEFORE UPDATE ON property_details
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ── Détails objets de valeur (EF-035) ─────────────────────────────────────────
-- Extension 1-1 d'un actif precious_metal / vehicle / valuable.
CREATE TABLE object_details (
    asset_id             UUID PRIMARY KEY REFERENCES assets(id) ON DELETE CASCADE,
    profile_id           UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    category             TEXT NOT NULL DEFAULT '',   -- montre, voiture, or, œuvre…
    brand                TEXT NOT NULL DEFAULT '',   -- marque / auteur / référence
    purchase_price_cents BIGINT NOT NULL DEFAULT 0 CHECK (purchase_price_cents >= 0),
    purchase_date        DATE,
    insured              BOOLEAN NOT NULL DEFAULT false,
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_object_profile ON object_details(profile_id);
CREATE TRIGGER trg_object_updated BEFORE UPDATE ON object_details
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ── Coffre-fort patrimonial (EF-064) ──────────────────────────────────────────
-- Documents stockés CHIFFRÉS côté serveur (AES-256-GCM, clé hors base :
-- OPALE_VAULT_KEY). content = nonce || ciphertext. Niveau N3 : un document
-- ne quitte JAMAIS le homelab (EIA-032).
CREATE TABLE documents (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id  UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    asset_id    UUID REFERENCES assets(id) ON DELETE SET NULL,  -- pièce liée à un actif (facture, acte…)
    name        TEXT NOT NULL CHECK (length(trim(name)) > 0),
    kind        TEXT NOT NULL DEFAULT 'other' CHECK (kind IN (
                    'deed',       -- acte (notarié, propriété)
                    'contract',   -- contrat (AV, assurance, prêt)
                    'invoice',    -- facture
                    'identity',   -- pièce d'identité
                    'insurance',  -- attestation d'assurance
                    'tax',        -- avis d'imposition
                    'other'
                )),
    mime        TEXT NOT NULL DEFAULT 'application/octet-stream',
    size_bytes  BIGINT NOT NULL CHECK (size_bytes > 0),
    sha256      TEXT NOT NULL,          -- empreinte du contenu EN CLAIR (intégrité)
    content     BYTEA NOT NULL,         -- nonce || ciphertext (AES-256-GCM)
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_documents_profile ON documents(profile_id);
CREATE INDEX idx_documents_asset ON documents(asset_id);

-- ── Contacts de transmission (EF-063) ─────────────────────────────────────────
-- Qui contacter « si un jour » : notaire, banquier, assureur, proche de confiance.
CREATE TABLE contacts (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id  UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    name        TEXT NOT NULL CHECK (length(trim(name)) > 0),
    role        TEXT NOT NULL DEFAULT 'other' CHECK (role IN (
                    'notary', 'banker', 'insurer', 'accountant', 'trusted', 'other'
                )),
    phone       TEXT NOT NULL DEFAULT '',
    email       TEXT NOT NULL DEFAULT '',
    note        TEXT NOT NULL DEFAULT '',   -- ex. « détient le testament », n° de contrat…
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_contacts_profile ON contacts(profile_id);
CREATE TRIGGER trg_contacts_updated BEFORE UPDATE ON contacts
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
