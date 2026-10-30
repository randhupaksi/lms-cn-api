ALTER TABLE users DROP CHECK chk_users_role, ADD CONSTRAINT chk_users_role CHECK (role IN ('superadmin', 'principal', 'curriculum', 'admin', 'teacher', 'student'));
