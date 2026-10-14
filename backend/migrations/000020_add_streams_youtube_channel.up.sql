ALTER TABLE streams ADD COLUMN youtube_channel_id text;

-- One row per channel: a member on both platforms gets two rows, so the Twitch and
-- YouTube jobs never write live status to the same row.
ALTER TABLE streams ADD CONSTRAINT streams_single_platform
    CHECK (twitch_login IS NULL OR youtube_channel_id IS NULL);
