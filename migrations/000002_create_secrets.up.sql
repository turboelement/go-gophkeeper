CREATE TABLE secrets (
    id                UUID PRIMARY KEY,
    user_id           UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type              TEXT NOT NULL,
    title             TEXT NOT NULL,
    metadata          JSONB,
    encrypted_payload BYTEA NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at        TIMESTAMPTZ
);

CREATE INDEX idx_secrets_user_id ON secrets(user_id);
CREATE INDEX idx_secrets_deleted_at ON secrets(deleted_at) WHERE deleted_at IS NULL;
