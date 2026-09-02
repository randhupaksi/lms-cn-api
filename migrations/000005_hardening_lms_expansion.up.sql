ALTER TABLE questions
    ADD COLUMN category VARCHAR(80) NULL AFTER default_points,
    ADD COLUMN tags JSON NULL AFTER category,
    ADD KEY idx_questions_course_category (course_id, category);

ALTER TABLE exams
    ADD COLUMN randomize_questions BOOLEAN NOT NULL DEFAULT FALSE AFTER allow_back_navigation,
    ADD COLUMN randomize_options BOOLEAN NOT NULL DEFAULT FALSE AFTER randomize_questions;

CREATE TABLE course_materials (
    id CHAR(36) NOT NULL,
    course_id CHAR(36) NOT NULL,
    author_id CHAR(36) NOT NULL,
    title VARCHAR(180) NOT NULL,
    description TEXT NULL,
    content TEXT NOT NULL,
    position SMALLINT UNSIGNED NOT NULL DEFAULT 1,
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    published_at DATETIME(6) NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    KEY idx_materials_course_status_position (course_id, status, position),
    CONSTRAINT fk_materials_course FOREIGN KEY (course_id) REFERENCES courses (id) ON DELETE CASCADE,
    CONSTRAINT fk_materials_author FOREIGN KEY (author_id) REFERENCES users (id),
    CONSTRAINT chk_materials_status CHECK (status IN ('draft', 'published', 'archived'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE material_progress (
    material_id CHAR(36) NOT NULL,
    student_id CHAR(36) NOT NULL,
    completed_at DATETIME(6) NOT NULL,
    PRIMARY KEY (material_id, student_id),
    KEY idx_material_progress_student (student_id, completed_at),
    CONSTRAINT fk_material_progress_material FOREIGN KEY (material_id) REFERENCES course_materials (id) ON DELETE CASCADE,
    CONSTRAINT fk_material_progress_student FOREIGN KEY (student_id) REFERENCES users (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE assignments (
    id CHAR(36) NOT NULL,
    course_id CHAR(36) NOT NULL,
    author_id CHAR(36) NOT NULL,
    title VARCHAR(180) NOT NULL,
    instructions TEXT NOT NULL,
    due_at DATETIME(6) NOT NULL,
    max_score DECIMAL(10,2) NOT NULL DEFAULT 100,
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    published_at DATETIME(6) NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    KEY idx_assignments_course_status_due (course_id, status, due_at),
    CONSTRAINT fk_assignments_course FOREIGN KEY (course_id) REFERENCES courses (id) ON DELETE CASCADE,
    CONSTRAINT fk_assignments_author FOREIGN KEY (author_id) REFERENCES users (id),
    CONSTRAINT chk_assignments_status CHECK (status IN ('draft', 'published', 'closed', 'archived')),
    CONSTRAINT chk_assignments_score CHECK (max_score > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE assignment_submissions (
    id CHAR(36) NOT NULL,
    assignment_id CHAR(36) NOT NULL,
    student_id CHAR(36) NOT NULL,
    content TEXT NOT NULL,
    attachment_url VARCHAR(1000) NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'submitted',
    score DECIMAL(10,2) NULL,
    feedback TEXT NULL,
    submitted_at DATETIME(6) NOT NULL,
    graded_at DATETIME(6) NULL,
    graded_by CHAR(36) NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    UNIQUE KEY uq_assignment_submission_student (assignment_id, student_id),
    KEY idx_assignment_submissions_status (assignment_id, status, submitted_at),
    CONSTRAINT fk_assignment_submissions_assignment FOREIGN KEY (assignment_id) REFERENCES assignments (id) ON DELETE CASCADE,
    CONSTRAINT fk_assignment_submissions_student FOREIGN KEY (student_id) REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT fk_assignment_submissions_grader FOREIGN KEY (graded_by) REFERENCES users (id) ON DELETE SET NULL,
    CONSTRAINT chk_assignment_submissions_status CHECK (status IN ('submitted', 'graded', 'returned')),
    CONSTRAINT chk_assignment_submission_score CHECK (score IS NULL OR score >= 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

