CREATE TABLE questions (
    id CHAR(36) NOT NULL,
    course_id CHAR(36) NOT NULL,
    author_id CHAR(36) NOT NULL,
    type VARCHAR(32) NOT NULL DEFAULT 'single_choice',
    stem TEXT NOT NULL,
    default_points DECIMAL(8,2) NOT NULL DEFAULT 1.00,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    version INT UNSIGNED NOT NULL DEFAULT 1,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    KEY idx_questions_course_status (course_id, status),
    KEY idx_questions_author (author_id),
    CONSTRAINT fk_questions_course FOREIGN KEY (course_id) REFERENCES courses (id),
    CONSTRAINT fk_questions_author FOREIGN KEY (author_id) REFERENCES users (id),
    CONSTRAINT chk_questions_type CHECK (type IN ('single_choice')),
    CONSTRAINT chk_questions_status CHECK (status IN ('active', 'archived')),
    CONSTRAINT chk_questions_points CHECK (default_points > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE question_options (
    id CHAR(36) NOT NULL,
    question_id CHAR(36) NOT NULL,
    content TEXT NOT NULL,
    is_correct BOOLEAN NOT NULL DEFAULT FALSE,
    position SMALLINT UNSIGNED NOT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    UNIQUE KEY uq_question_options_position (question_id, position),
    KEY idx_question_options_question (question_id),
    CONSTRAINT fk_question_options_question FOREIGN KEY (question_id) REFERENCES questions (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE exams (
    id CHAR(36) NOT NULL,
    course_id CHAR(36) NOT NULL,
    author_id CHAR(36) NOT NULL,
    title VARCHAR(180) NOT NULL,
    description TEXT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    starts_at DATETIME(6) NOT NULL,
    ends_at DATETIME(6) NOT NULL,
    duration_minutes SMALLINT UNSIGNED NOT NULL,
    max_attempts SMALLINT UNSIGNED NOT NULL DEFAULT 1,
    allow_back_navigation BOOLEAN NOT NULL DEFAULT TRUE,
    result_policy VARCHAR(24) NOT NULL DEFAULT 'after_publish',
    published_at DATETIME(6) NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    KEY idx_exams_course_status_schedule (course_id, status, starts_at, ends_at),
    KEY idx_exams_author (author_id),
    CONSTRAINT fk_exams_course FOREIGN KEY (course_id) REFERENCES courses (id),
    CONSTRAINT fk_exams_author FOREIGN KEY (author_id) REFERENCES users (id),
    CONSTRAINT chk_exams_status CHECK (status IN ('draft', 'published', 'closed', 'archived')),
    CONSTRAINT chk_exams_schedule CHECK (ends_at > starts_at),
    CONSTRAINT chk_exams_duration CHECK (duration_minutes > 0),
    CONSTRAINT chk_exams_attempts CHECK (max_attempts = 1),
    CONSTRAINT chk_exams_result_policy CHECK (result_policy IN ('after_publish'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE exam_participants (
    exam_id CHAR(36) NOT NULL,
    student_id CHAR(36) NOT NULL,
    assigned_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    PRIMARY KEY (exam_id, student_id),
    KEY idx_exam_participants_student (student_id, exam_id),
    CONSTRAINT fk_exam_participants_exam FOREIGN KEY (exam_id) REFERENCES exams (id) ON DELETE CASCADE,
    CONSTRAINT fk_exam_participants_student FOREIGN KEY (student_id) REFERENCES users (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE exam_questions (
    id CHAR(36) NOT NULL,
    exam_id CHAR(36) NOT NULL,
    source_question_id CHAR(36) NOT NULL,
    source_version INT UNSIGNED NOT NULL,
    type VARCHAR(32) NOT NULL,
    stem TEXT NOT NULL,
    position SMALLINT UNSIGNED NOT NULL,
    points DECIMAL(8,2) NOT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    UNIQUE KEY uq_exam_questions_position (exam_id, position),
    UNIQUE KEY uq_exam_questions_source (exam_id, source_question_id),
    CONSTRAINT fk_exam_questions_exam FOREIGN KEY (exam_id) REFERENCES exams (id) ON DELETE CASCADE,
    CONSTRAINT fk_exam_questions_source FOREIGN KEY (source_question_id) REFERENCES questions (id),
    CONSTRAINT chk_exam_questions_type CHECK (type IN ('single_choice')),
    CONSTRAINT chk_exam_questions_points CHECK (points > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE exam_question_options (
    id CHAR(36) NOT NULL,
    exam_question_id CHAR(36) NOT NULL,
    source_option_id CHAR(36) NOT NULL,
    content TEXT NOT NULL,
    is_correct BOOLEAN NOT NULL,
    position SMALLINT UNSIGNED NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_exam_question_options_position (exam_question_id, position),
    CONSTRAINT fk_exam_question_options_question FOREIGN KEY (exam_question_id) REFERENCES exam_questions (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
