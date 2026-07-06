-- Opale — migration 0007 : journal d'accès (ENF-004, §10 du cahier des
-- charges : « qui a consulté quoi »). Trace les événements sensibles :
-- connexions (réussies et échouées), téléchargements de documents, exports.

CREATE TABLE access_log (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- Nullable : un échec de connexion peut viser un profil inexistant.
    profile_id  UUID REFERENCES profiles(id) ON DELETE CASCADE,
    event       TEXT NOT NULL CHECK (event IN (
                    'login_ok', 'login_failed', 'login_locked',
                    'document_downloaded', 'export', 'logout'
                )),
    detail      TEXT NOT NULL DEFAULT '',
    at          TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_access_log_profile ON access_log(profile_id, at DESC);
