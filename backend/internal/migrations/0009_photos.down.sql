ALTER TABLE documents DROP CONSTRAINT documents_kind_check;
ALTER TABLE documents ADD CONSTRAINT documents_kind_check CHECK (kind IN (
    'deed', 'contract', 'invoice', 'identity', 'insurance', 'tax', 'other'
));
