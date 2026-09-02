CREATE TABLE attempts (
    id CHAR(36) NOT NULL,
    exam_id CHAR(36) NOT NULL,
    student_id CHAR(36) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'in_progress',
    start_idempotency_key VARCHAR(120) NOT NULL,
    started_at DATETIME(6) NOT NULL,
    deadline_at DATETIME(6) NOT NULL,
    submitted_at DATETIME(6) NULL,
    submission_receipt CHAR(36) NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    UNIQUE KEY uq_attempts_exam_student (exam_id, student_id),
    UNIQUE KEY uq_attempts_start_key (student_id, start_idempotency_key),
    UNIQUE KEY uq_attempts_receipt (submission_receipt),
    KEY idx_attempts_exam_status (exam_id, status),
    KEY idx_attempts_deadline_status (deadline_at, status),
    CONSTRAINT fk_attempts_exam FOREIGN KEY (exam_id) REFERENCES exams (id),
    CONSTRAINT fk_attempts_student FOREIGN KEY (student_id) REFERENCES users (id),
    CONSTRAINT chk_attempts_status CHECK (status IN ('in_progress', 'submitted', 'expired'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE attempt_answers (
    id CHAR(36) NOT NULL,
    attempt_id CHAR(36) NOT NULL,
    exam_question_id CHAR(36) NOT NULL,
    selected_option_id CHAR(36) NOT NULL,
    revision INT UNSIGNED NOT NULL DEFAULT 1,
    saved_at DATETIME(6) NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_attempt_answers_question (attempt_id, exam_question_id),
    KEY idx_attempt_answers_attempt (attempt_id),
    CONSTRAINT fk_attempt_answers_attempt FOREIGN KEY (attempt_id) REFERENCES attempts (id) ON DELETE CASCADE,
    CONSTRAINT fk_attempt_answers_question FOREIGN KEY (exam_question_id) REFERENCES exam_questions (id),
    CONSTRAINT fk_attempt_answers_option FOREIGN KEY (selected_option_id) REFERENCES exam_question_options (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE attempt_answer_save_requests (
    attempt_id CHAR(36) NOT NULL,
    idempotency_key VARCHAR(120) NOT NULL,
    exam_question_id CHAR(36) NOT NULL,
    selected_option_id CHAR(36) NOT NULL,
    answer_id CHAR(36) NOT NULL,
    revision INT UNSIGNED NOT NULL,
    saved_at DATETIME(6) NOT NULL,
    created_at DATETIME(6) NOT NULL,
    PRIMARY KEY (attempt_id, idempotency_key),
    KEY idx_answer_save_requests_answer (answer_id),
    CONSTRAINT fk_answer_save_requests_attempt FOREIGN KEY (attempt_id) REFERENCES attempts (id) ON DELETE CASCADE,
    CONSTRAINT fk_answer_save_requests_answer FOREIGN KEY (answer_id) REFERENCES attempt_answers (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE attempt_events (
    id CHAR(36) NOT NULL,
    attempt_id CHAR(36) NOT NULL,
    actor_id CHAR(36) NOT NULL,
    event_type VARCHAR(40) NOT NULL,
    metadata JSON NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    KEY idx_attempt_events_attempt_time (attempt_id, created_at),
    CONSTRAINT fk_attempt_events_attempt FOREIGN KEY (attempt_id) REFERENCES attempts (id) ON DELETE CASCADE,
    CONSTRAINT fk_attempt_events_actor FOREIGN KEY (actor_id) REFERENCES users (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
