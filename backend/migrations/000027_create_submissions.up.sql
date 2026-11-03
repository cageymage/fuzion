-- Art and screenshots members submit for officers to review (#175). Unlike images,
-- the exact original is kept and only officers can read it. byte_size is the
-- original's length, stored so quotas and the storage ceiling are cheap sums.
CREATE TABLE submissions (
    id                      uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                 uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    original_bytes          bytea       NOT NULL,
    byte_size               integer     NOT NULL CHECK (byte_size > 0),
    content_type            text        NOT NULL,
    original_filename       text        NOT NULL,
    thumb_bytes             bytea       NOT NULL,
    width                   integer     NOT NULL CHECK (width > 0),
    height                  integer     NOT NULL CHECK (height > 0),
    description             text        NOT NULL DEFAULT '',
    attestation_accepted_at timestamptz NOT NULL,
    attestation_version     text        NOT NULL,
    content_hash            text        NOT NULL,
    created_at              timestamptz NOT NULL,
    UNIQUE (user_id, content_hash)
);

CREATE INDEX submissions_user_id_created_at_idx ON submissions (user_id, created_at);
CREATE INDEX submissions_created_at_idx ON submissions (created_at DESC, id);
