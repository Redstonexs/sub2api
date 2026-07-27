-- Create group_view_grants table to record which regular users may view which
-- group's quota dashboard card. Separate from user_allowed_groups (API key binding).

CREATE TABLE IF NOT EXISTS group_view_grants (
    id                  BIGSERIAL PRIMARY KEY,
    user_id             BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    group_id            BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    granted_by_user_id  BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    granted_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMPTZ NULL
);

-- Partial unique index: a (user_id, group_id) pair can only have one active grant.
CREATE UNIQUE INDEX IF NOT EXISTS idx_group_view_grants_user_group_active
    ON group_view_grants(user_id, group_id)
    WHERE deleted_at IS NULL;

-- Index for looking up all grants for a group (e.g. revoking group access).
CREATE INDEX IF NOT EXISTS idx_group_view_grants_group_id
    ON group_view_grants(group_id);
