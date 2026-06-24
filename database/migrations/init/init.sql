CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TYPE gender_type AS ENUM ('male', 'female');

CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  first_name VARCHAR(50) NOT NULL,
  second_name VARCHAR(50) NOT NULL,
  biography TEXT,
  birthdate DATE,
  city VARCHAR(50),
  gender gender_type,
  password_hash VARCHAR(255) NOT NULL
);

CREATE INDEX idx_names_prefix ON users (
  first_name varchar_pattern_ops,
  second_name varchar_pattern_ops
);

CREATE TABLE auth_tokens (
  id BIGSERIAL PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,
  expires_at TIMESTAMPTZ NOT NULL,
  is_revoked BOOLEAN NOT NULL DEFAULT FALSE
);
