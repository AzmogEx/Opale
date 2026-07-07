-- Opale — migration 0009 : photos d'objets (EF-035).
-- Une photo est un document du coffre (chiffrée comme le reste, N3) de
-- type « photo », rattachée à l'actif.
ALTER TABLE documents DROP CONSTRAINT documents_kind_check;
ALTER TABLE documents ADD CONSTRAINT documents_kind_check CHECK (kind IN (
    'deed', 'contract', 'invoice', 'identity', 'insurance', 'tax', 'photo', 'other'
));
