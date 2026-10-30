ALTER TABLE users DROP CHECK chk_users_role, ADD CONSTRAINT chk_users_role CHECK (role IN ('superadmin', 'admin', 'teacher', 'student'));

DROP TABLE leadership_action_updates;
DROP TABLE leadership_actions;
