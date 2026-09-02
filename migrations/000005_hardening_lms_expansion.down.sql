DROP TABLE IF EXISTS assignment_submissions;
DROP TABLE IF EXISTS assignments;
DROP TABLE IF EXISTS material_progress;
DROP TABLE IF EXISTS course_materials;

ALTER TABLE exams
    DROP COLUMN randomize_options,
    DROP COLUMN randomize_questions;

ALTER TABLE questions
    DROP INDEX idx_questions_course_category,
    DROP COLUMN tags,
    DROP COLUMN category;
