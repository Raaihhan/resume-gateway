-- database: resume
CREATE DATABASE IF NOT EXISTS resume CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
USE resume;

-- profile (single row)
CREATE TABLE IF NOT EXISTS profiles (
                                        id          VARCHAR(36)  NOT NULL PRIMARY KEY,
    full_name   VARCHAR(255) NOT NULL,
    headline    VARCHAR(255) NOT NULL,
    summary     TEXT         NOT NULL,
    location    VARCHAR(255) NOT NULL,
    skills_csv  TEXT         NOT NULL, -- "Go,gRPC,PostgreSQL,Redis,Vue,Docker"
    avatar_url  VARCHAR(500) NOT NULL,
    updated_at  TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
    );

-- projects
CREATE TABLE IF NOT EXISTS projects (
                                        id          VARCHAR(64)  NOT NULL PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    description TEXT         NOT NULL,
    repo_url    VARCHAR(500) NOT NULL,
    demo_url    VARCHAR(500) NOT NULL,
    tags_csv    TEXT         NOT NULL,  -- "go,grpc,mc"
    created_at  TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
    );
CREATE INDEX idx_projects_name ON projects (name);

-- contact messages
CREATE TABLE IF NOT EXISTS contact_messages (
                                                id         BIGINT AUTO_INCREMENT PRIMARY KEY,
                                                name       VARCHAR(255) NOT NULL,
    email      VARCHAR(255) NOT NULL,
    message    TEXT         NOT NULL,
    created_at TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP
    );

-- seed 1 profile + 2 projects (opsional)
INSERT INTO profiles (id,full_name,headline,summary,location,skills_csv,avatar_url)
VALUES ('profile-1','Muhammad Raihan Nur Rizqi Amin',
        'Backend Developer (Go) | gRPC | Vue',
        'Backend dev fokus Go/gRPC, microservices & integration.',
        'Semarang, Indonesia',
        'Go,gRPC,PostgreSQL,Redis,Vue,Docker',
        'https://avatars.githubusercontent.com/u/xxxx')
    ON DUPLICATE KEY UPDATE full_name=VALUES(full_name);

INSERT INTO projects (id,name,description,repo_url,demo_url,tags_csv) VALUES
    ('p1','Money Changer Module','Microservice NDS BRI (MC)','','','go,grpc,mc')
    ON DUPLICATE KEY UPDATE name=VALUES(name);

INSERT INTO projects (id,name,description,repo_url,demo_url,tags_csv) VALUES
    ('p2','ATM Vault Inquiry','Inquiry vault ATM (internal)','','','go,grpc,atm')
    ON DUPLICATE KEY UPDATE name=VALUES(name);
