BEGIN;

-- postgres does not have builtin support for uuids as primary key
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE SCHEMA IF NOT EXISTS faceittha;

CREATE TABLE IF NOT EXISTS faceittha.users (
    id UUID DEFAULT uuid_generate_v4() NOT NULL PRIMARY KEY,
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    nickname TEXT NOT NULL,
    email TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    country TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

-- needed for proper replication in CDC
ALTER TABLE faceittha.users REPLICA IDENTITY FULL;

-- indexes used for efficient querying user by country and created_at fields
CREATE INDEX IF NOT EXISTS idx_users_country ON faceittha.users (country);
CREATE INDEX IF NOT EXISTS idx_users_created_at ON faceittha.users (created_at);

COMMIT;