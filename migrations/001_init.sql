-- ===========================================
-- INIT SCHEMA (tables only)
-- ===========================================

-- database: resume
CREATE DATABASE IF NOT EXISTS resume
  CHARACTER SET utf8mb4
  COLLATE utf8mb4_unicode_ci;
USE resume;

-- profiles
CREATE TABLE IF NOT EXISTS profiles (
  id          VARCHAR(36)  NOT NULL PRIMARY KEY,
  full_name   VARCHAR(255) NOT NULL,
  headline    VARCHAR(255) NOT NULL,
  summary     TEXT         NOT NULL,
  location    VARCHAR(255) NOT NULL,
  skills_csv  TEXT         NOT NULL,   -- contoh: "Go,gRPC,PostgreSQL,Redis,Vue,Docker"
  avatar_url  VARCHAR(500) NOT NULL,
  updated_at  TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP
                               ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;

-- projects
CREATE TABLE IF NOT EXISTS projects (
  id          VARCHAR(64)  NOT NULL PRIMARY KEY,
  name        VARCHAR(255) NOT NULL,
  description TEXT         NOT NULL,
  repo_url    VARCHAR(500) NOT NULL,
  demo_url    VARCHAR(500) NOT NULL,
  tags_csv    TEXT         NOT NULL,   -- contoh: "go,grpc,mc"
  created_at  TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at  TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP
                               ON UPDATE CURRENT_TIMESTAMP,
  KEY idx_projects_name (name)
) ENGINE=InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;

-- contact_messages
CREATE TABLE IF NOT EXISTS contact_messages (
  id         BIGINT AUTO_INCREMENT PRIMARY KEY,
  name       VARCHAR(255) NOT NULL,
  email      VARCHAR(255) NOT NULL,
  message    TEXT         NOT NULL,
  created_at TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;

-- hobbies
CREATE TABLE IF NOT EXISTS hobbies (
  id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  tag         VARCHAR(64)  NOT NULL,    -- contoh: 'coding', 'music'
  remarks     VARCHAR(255) NULL,
  description TEXT         NULL,
  created_at  TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at  TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP
                                   ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uq_hobbies_tag (tag)
) ENGINE=InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;
