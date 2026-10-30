ALTER TABLE users DROP CHECK chk_users_role, ADD CONSTRAINT chk_users_role CHECK (role IN ('superadmin', 'principal', 'admin', 'teacher', 'student'));

CREATE TABLE leadership_actions (
    id CHAR(36) NOT NULL PRIMARY KEY,
    title VARCHAR(180) NOT NULL,
    description VARCHAR(2000) NOT NULL DEFAULT '',
    category VARCHAR(24) NOT NULL,
    priority VARCHAR(12) NOT NULL DEFAULT 'normal',
    status VARCHAR(20) NOT NULL DEFAULT 'open',
    assignee_id CHAR(36) NULL,
    due_at DATETIME(6) NULL,
    created_by CHAR(36) NOT NULL,
    created_at DATETIME(6) NOT NULL,
    updated_at DATETIME(6) NOT NULL,
    completed_at DATETIME(6) NULL,
    version INT NOT NULL DEFAULT 1,
    CONSTRAINT chk_leadership_action_category CHECK (category IN ('academic', 'exam', 'data_quality', 'general')),
    CONSTRAINT chk_leadership_action_priority CHECK (priority IN ('low', 'normal', 'high')),
    CONSTRAINT chk_leadership_action_status CHECK (status IN ('open', 'in_progress', 'blocked', 'completed', 'cancelled')),
    CONSTRAINT fk_leadership_action_assignee FOREIGN KEY (assignee_id) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT fk_leadership_action_creator FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE RESTRICT,
    KEY idx_leadership_actions_status_due (status, due_at),
    KEY idx_leadership_actions_assignee_status (assignee_id, status),
    KEY idx_leadership_actions_created (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE leadership_action_updates (
    id CHAR(36) NOT NULL PRIMARY KEY,
    action_id CHAR(36) NOT NULL,
    actor_id CHAR(36) NOT NULL,
    from_status VARCHAR(20) NULL,
    to_status VARCHAR(20) NOT NULL,
    note VARCHAR(1000) NOT NULL,
    created_at DATETIME(6) NOT NULL,
    CONSTRAINT fk_leadership_update_action FOREIGN KEY (action_id) REFERENCES leadership_actions(id) ON DELETE CASCADE,
    CONSTRAINT fk_leadership_update_actor FOREIGN KEY (actor_id) REFERENCES users(id) ON DELETE RESTRICT,
    KEY idx_leadership_updates_action_time (action_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
