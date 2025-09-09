-- +goose Up
-- 在此写入迁移 SQL（执行时会运行）
-- 示例：
CREATE TABLE users (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
-- 在此写入回滚 SQL（执行 down 时会运行）
-- 示例：
DROP TABLE users;

