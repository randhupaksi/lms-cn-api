CREATE TABLE users (
    id CHAR(36) NOT NULL,
    identifier VARCHAR(64) NOT NULL,
    full_name VARCHAR(160) NOT NULL,
    role VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    password_hash VARCHAR(255) NOT NULL,
    must_change_password BOOLEAN NOT NULL DEFAULT TRUE,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    UNIQUE KEY uq_users_identifier (identifier),
    KEY idx_users_role_status (role, status),
    CONSTRAINT chk_users_role CHECK (role IN ('admin', 'teacher', 'student')),
    CONSTRAINT chk_users_status CHECK (status IN ('active', 'inactive'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE auth_sessions (
    id CHAR(36) NOT NULL,
    user_id CHAR(36) NOT NULL,
    refresh_token_hash CHAR(64) NOT NULL,
    expires_at DATETIME(6) NOT NULL,
    last_used_at DATETIME(6) NOT NULL,
    revoked_at DATETIME(6) NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    UNIQUE KEY uq_auth_sessions_refresh_hash (refresh_token_hash),
    KEY idx_auth_sessions_user_active (user_id, revoked_at, expires_at),
    CONSTRAINT fk_auth_sessions_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE academic_years (
    id CHAR(36) NOT NULL,
    name VARCHAR(80) NOT NULL,
    starts_on DATE NOT NULL,
    ends_on DATE NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'inactive',
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    UNIQUE KEY uq_academic_years_name (name),
    KEY idx_academic_years_status (status),
    CONSTRAINT chk_academic_years_status CHECK (status IN ('active', 'inactive')),
    CONSTRAINT chk_academic_years_dates CHECK (ends_on >= starts_on)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE class_groups (
    id CHAR(36) NOT NULL,
    academic_year_id CHAR(36) NOT NULL,
    name VARCHAR(80) NOT NULL,
    grade_level SMALLINT NOT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    UNIQUE KEY uq_class_groups_year_name (academic_year_id, name),
    KEY idx_class_groups_year (academic_year_id),
    CONSTRAINT fk_class_groups_year FOREIGN KEY (academic_year_id) REFERENCES academic_years (id),
    CONSTRAINT chk_class_groups_grade CHECK (grade_level BETWEEN 1 AND 12)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE subjects (
    id CHAR(36) NOT NULL,
    code VARCHAR(32) NOT NULL,
    name VARCHAR(120) NOT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    UNIQUE KEY uq_subjects_code (code),
    UNIQUE KEY uq_subjects_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE courses (
    id CHAR(36) NOT NULL,
    academic_year_id CHAR(36) NOT NULL,
    class_group_id CHAR(36) NOT NULL,
    subject_id CHAR(36) NOT NULL,
    name VARCHAR(160) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    UNIQUE KEY uq_courses_context (academic_year_id, class_group_id, subject_id),
    KEY idx_courses_status (status),
    CONSTRAINT fk_courses_year FOREIGN KEY (academic_year_id) REFERENCES academic_years (id),
    CONSTRAINT fk_courses_class FOREIGN KEY (class_group_id) REFERENCES class_groups (id),
    CONSTRAINT fk_courses_subject FOREIGN KEY (subject_id) REFERENCES subjects (id),
    CONSTRAINT chk_courses_status CHECK (status IN ('active', 'archived'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE course_teachers (
    course_id CHAR(36) NOT NULL,
    teacher_id CHAR(36) NOT NULL,
    assigned_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    PRIMARY KEY (course_id, teacher_id),
    KEY idx_course_teachers_teacher (teacher_id, course_id),
    CONSTRAINT fk_course_teachers_course FOREIGN KEY (course_id) REFERENCES courses (id) ON DELETE CASCADE,
    CONSTRAINT fk_course_teachers_user FOREIGN KEY (teacher_id) REFERENCES users (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE course_students (
    course_id CHAR(36) NOT NULL,
    student_id CHAR(36) NOT NULL,
    enrolled_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    PRIMARY KEY (course_id, student_id),
    KEY idx_course_students_student (student_id, course_id),
    CONSTRAINT fk_course_students_course FOREIGN KEY (course_id) REFERENCES courses (id) ON DELETE CASCADE,
    CONSTRAINT fk_course_students_user FOREIGN KEY (student_id) REFERENCES users (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
