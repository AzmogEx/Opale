-- Opale — migration 0008 : nouveaux événements du journal d'accès pour la
-- gestion du profil (réinitialisation des données, suppression).

ALTER TABLE access_log DROP CONSTRAINT access_log_event_check;
ALTER TABLE access_log ADD CONSTRAINT access_log_event_check CHECK (event IN (
    'login_ok', 'login_failed', 'login_locked',
    'document_downloaded', 'export', 'logout',
    'data_reset', 'profile_deleted'
));
