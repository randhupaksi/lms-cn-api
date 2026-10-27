ALTER TABLE users DROP CHECK chk_users_role, ADD CONSTRAINT chk_users_role CHECK (role IN ('admin', 'teacher', 'student'));
DROP TABLE governance_requests;
DROP TABLE school_events;
DROP TABLE school_setting_changes;
DROP TABLE school_settings;
