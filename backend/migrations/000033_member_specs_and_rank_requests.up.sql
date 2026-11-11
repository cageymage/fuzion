-- Role is the member's to choose now, so a synced character starts with none instead
-- of a placeholder. Synced characters still holding the old placeholder (dps and no
-- spec) are cleared; hand-entered ones keep what the officer picked. A synced character
-- an officer set to dps without a spec looks the same as the placeholder and is cleared
-- too, and the down migration cannot tell them apart either.
ALTER TABLE characters ALTER COLUMN role DROP NOT NULL;
UPDATE characters SET role = NULL
WHERE guild_rank IS NOT NULL AND role = 'dps' AND spec IS NULL AND spec2 IS NULL;

-- Only the professions sync wrote this, and nothing read it.
ALTER TABLE professions DROP COLUMN source;

-- A member asking an officer to move a character into or out of an alt rank in game.
-- direction is fixed from the character's main/alt state when the request is filed.
CREATE TABLE rank_change_requests (
    id              uuid PRIMARY KEY,
    character_id    uuid        NOT NULL REFERENCES characters (id) ON DELETE CASCADE,
    requested_by    uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    direction       text        NOT NULL CHECK (direction IN ('to_main', 'to_alt')),
    status          text        NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'done', 'declined', 'cancelled')),
    note            text        NOT NULL DEFAULT '',
    resolution_note text        NOT NULL DEFAULT '',
    created_at      timestamptz NOT NULL,
    resolved_at     timestamptz,
    -- NULL on a done request means the roster sync closed it; cancelled requests also have none.
    resolved_by     uuid        REFERENCES users (id) ON DELETE SET NULL
);

CREATE UNIQUE INDEX rank_change_requests_one_open_per_character
    ON rank_change_requests (character_id) WHERE status = 'open';

-- The member list picks each character's newest request; the officer list shows recent history.
CREATE INDEX rank_change_requests_character_created ON rank_change_requests (character_id, created_at);
CREATE INDEX rank_change_requests_resolved ON rank_change_requests (resolved_at DESC)
    WHERE status IN ('done', 'declined');
