CREATE TABLE results (
    id CHAR(36) NOT NULL,
    attempt_id CHAR(36) NOT NULL,
    exam_id CHAR(36) NOT NULL,
    student_id CHAR(36) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    score DECIMAL(10,2) NOT NULL DEFAULT 0,
    max_score DECIMAL(10,2) NOT NULL DEFAULT 0,
    graded_at DATETIME(6) NOT NULL,
    published_at DATETIME(6) NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    UNIQUE KEY uq_results_attempt (attempt_id),
    KEY idx_results_exam_status (exam_id, status),
    KEY idx_results_student_status (student_id, status),
    CONSTRAINT fk_results_attempt FOREIGN KEY (attempt_id) REFERENCES attempts (id),
    CONSTRAINT fk_results_exam FOREIGN KEY (exam_id) REFERENCES exams (id),
    CONSTRAINT fk_results_student FOREIGN KEY (student_id) REFERENCES users (id),
    CONSTRAINT chk_results_status CHECK (status IN ('draft', 'reviewed', 'published')),
    CONSTRAINT chk_results_score CHECK (score >= 0 AND max_score >= 0 AND score <= max_score)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE audit_logs (
    id CHAR(36) NOT NULL,
    actor_id CHAR(36) NULL,
    action VARCHAR(80) NOT NULL,
    entity_type VARCHAR(64) NOT NULL,
    entity_id CHAR(36) NULL,
    metadata JSON NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    KEY idx_audit_logs_entity (entity_type, entity_id, created_at),
    KEY idx_audit_logs_actor_time (actor_id, created_at),
    CONSTRAINT fk_audit_logs_actor FOREIGN KEY (actor_id) REFERENCES users (id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
