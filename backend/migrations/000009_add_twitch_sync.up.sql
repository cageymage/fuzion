ALTER TABLE streams ADD COLUMN twitch_login text;
ALTER TABLE streams ADD COLUMN profile_image_url text;

UPDATE streams
SET twitch_login = lower(substring(channel_url from 'twitch\.tv/([^/?#]+)'));

CREATE TABLE sync_log (
    id      bigserial   PRIMARY KEY,
    source  text        NOT NULL,
    status  text        NOT NULL CHECK (status IN ('ok', 'error')),
    ran_at  timestamptz NOT NULL DEFAULT now(),
    message text        NOT NULL DEFAULT ''
);

CREATE INDEX sync_log_source_ran_at_idx ON sync_log (source, ran_at DESC);
