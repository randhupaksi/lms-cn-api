CREATE TABLE course_material_attachments (
    id CHAR(36) NOT NULL,
    material_id CHAR(36) NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    content_type VARCHAR(100) NOT NULL,
    size_bytes BIGINT UNSIGNED NOT NULL,
    checksum_sha256 CHAR(64) NOT NULL,
    file_data LONGBLOB NOT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    UNIQUE KEY uq_material_attachments_checksum (material_id, checksum_sha256),
    KEY idx_material_attachments_material (material_id, created_at),
    CONSTRAINT fk_material_attachments_material
        FOREIGN KEY (material_id) REFERENCES course_materials (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
