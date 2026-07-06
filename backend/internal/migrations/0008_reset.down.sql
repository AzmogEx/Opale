ALTER TABLE access_log DROP CONSTRAINT access_log_event_check;
ALTER TABLE access_log ADD CONSTRAINT access_log_event_check CHECK (event IN (
    'login_ok', 'login_failed', 'login_locked',
    'document_downloaded', 'export', 'logout'
));
