CREATE TABLE images (
    id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    content_type text       NOT NULL,
    width       integer     NOT NULL,
    height      integer     NOT NULL,
    full_bytes  bytea       NOT NULL,
    thumb_bytes bytea       NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now()
);
