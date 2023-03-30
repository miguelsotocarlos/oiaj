CREATE TABLE IF NOT EXISTS oia_user (
    id BIGSERIAL PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    is_email_verified BOOLEAN NOT NULL DEFAULT FALSE,
    username TEXT NOT NULL UNIQUE,
    cms_user_id BIGINT NOT NULL,
    password_hash BYTEA NOT NULL,
    score REAL NOT NULL DEFAULT 0
)

;;

CREATE TABLE IF NOT EXISTS oia_tokens (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT,
    secret BYTEA,
    CONSTRAINT fk_user_id
        FOREIGN KEY(user_id)
            REFERENCES oia_user(id)
)

;;

CREATE TABLE IF NOT EXISTS oia_submissions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT,
    task_id BIGINT,
    details TEXT,
    subtask_details TEXT,
    CONSTRAINT fk_user_id
        FOREIGN KEY(user_id)
            REFERENCES oia_user(id)
)

;;

CREATE TABLE IF NOT EXISTS oia_task (
    id BIGSERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    name TEXT NOT NULL,
    statement BYTEA NOT NULL,
    max_score REAL NOT NULL,
    submission_format TEXT[] NOT NULL,
    tags TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    multiplier REAL NOT NULL DEFAULT 1
)

;;

CREATE TABLE IF NOT EXISTS oia_task_score (
    user_id BIGINT,
    task_id BIGINT,
    score REAL,
    base_score REAL,
    PRIMARY KEY (user_id, task_id),
    CONSTRAINT fk_user_id
        FOREIGN KEY(user_id)
            REFERENCES oia_user(id)
)