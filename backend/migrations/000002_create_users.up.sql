CREATE TABLE users (
    id           uuid PRIMARY KEY,
    discord_id   text        NOT NULL UNIQUE,
    username     text        NOT NULL,
    avatar_url   text,
    created_at   timestamptz NOT NULL DEFAULT now(),
    last_seen_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE sessions (
    token      text        PRIMARY KEY,
    user_id    uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    expires_at timestamptz NOT NULL
);

CREATE INDEX sessions_user_id_idx ON sessions (user_id);
