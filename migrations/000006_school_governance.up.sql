ALTER TABLE users DROP CHECK chk_users_role, ADD CONSTRAINT chk_users_role CHECK (role IN ('superadmin', 'admin', 'teacher', 'student'));

CREATE TABLE school_settings (
    id INT NOT NULL PRIMARY KEY,
    school_name VARCHAR(160) NOT NULL,
    motto VARCHAR(255) NOT NULL DEFAULT '',
    contact_email VARCHAR(160) NOT NULL DEFAULT '',
    support_message VARCHAR(1000) NOT NULL DEFAULT '',
    passing_reference DECIMAL(5,2) NOT NULL DEFAULT 75,
    version INT NOT NULL DEFAULT 1,
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    CONSTRAINT chk_school_passing CHECK (passing_reference BETWEEN 0 AND 100)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT INTO school_settings (id, school_name, motto, support_message) VALUES (1, 'Citra Negara', 'Belajar, bertumbuh, berkarya.', 'Hubungi administrator sekolah untuk bantuan akses.');

CREATE TABLE school_setting_changes (
    id CHAR(36) NOT NULL PRIMARY KEY,
    actor_id CHAR(36) NOT NULL,
    version INT NOT NULL,
    reason VARCHAR(500) NOT NULL,
    snapshot JSON NOT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    CONSTRAINT fk_setting_change_actor FOREIGN KEY (actor_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE school_events (
    id CHAR(36) NOT NULL PRIMARY KEY,
    title VARCHAR(180) NOT NULL,
    description VARCHAR(2000) NOT NULL DEFAULT '',
    category VARCHAR(24) NOT NULL,
    starts_at DATETIME(6) NOT NULL,
    ends_at DATETIME(6) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_by CHAR(36) NOT NULL,
    version INT NOT NULL DEFAULT 1,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    KEY idx_school_events_dates (status, starts_at),
    CONSTRAINT fk_school_event_actor FOREIGN KEY (created_by) REFERENCES users(id),
    CONSTRAINT chk_school_event_dates CHECK (ends_at > starts_at),
    CONSTRAINT chk_school_event_status CHECK (status IN ('active', 'archived')),
    CONSTRAINT chk_school_event_category CHECK (category IN ('academic', 'holiday', 'meeting', 'announcement'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE governance_requests (
    id CHAR(36) NOT NULL PRIMARY KEY,
    exam_id CHAR(36) NOT NULL,
    requester_id CHAR(36) NOT NULL,
    reason VARCHAR(1000) NOT NULL,
    proposed_starts_at DATETIME(6) NOT NULL,
    proposed_ends_at DATETIME(6) NOT NULL,
    original_starts_at DATETIME(6) NOT NULL,
    original_ends_at DATETIME(6) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    reviewer_id CHAR(36) NULL,
    decision_note VARCHAR(1000) NOT NULL DEFAULT '',
    decided_at DATETIME(6) NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    pending_exam_id CHAR(36) GENERATED ALWAYS AS (CASE WHEN status = 'pending' THEN exam_id ELSE NULL END) STORED,
    UNIQUE KEY uq_pending_exam (pending_exam_id),
    KEY idx_governance_requests_status (status, created_at),
    CONSTRAINT fk_governance_exam FOREIGN KEY (exam_id) REFERENCES exams(id),
    CONSTRAINT fk_governance_requester FOREIGN KEY (requester_id) REFERENCES users(id),
    CONSTRAINT fk_governance_reviewer FOREIGN KEY (reviewer_id) REFERENCES users(id),
    CONSTRAINT chk_governance_dates CHECK (proposed_ends_at > proposed_starts_at),
    CONSTRAINT chk_governance_status CHECK (status IN ('pending', 'approved', 'rejected', 'cancelled'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
