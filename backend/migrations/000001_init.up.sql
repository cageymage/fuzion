CREATE TABLE news_posts (
    id           uuid PRIMARY KEY,
    title        text        NOT NULL,
    excerpt      text        NOT NULL,
    category     text        NOT NULL CHECK (category IN ('raid-progress', 'recruitment', 'guild-news')),
    image_url    text,
    author_name  text        NOT NULL,
    published_at timestamptz NOT NULL
);

CREATE INDEX news_posts_published_at_idx ON news_posts (published_at DESC);

CREATE TABLE raids (
    id               uuid PRIMARY KEY,
    difficulty       text        NOT NULL,
    instance_name    text        NOT NULL,
    starts_at        timestamptz NOT NULL,
    progress_summary text        NOT NULL
);

CREATE INDEX raids_starts_at_idx ON raids (starts_at);

CREATE TABLE streams (
    id            uuid PRIMARY KEY,
    streamer_name text    NOT NULL,
    game_name     text    NOT NULL,
    viewer_count  integer NOT NULL DEFAULT 0,
    thumbnail_url text,
    channel_url   text    NOT NULL,
    is_live       boolean NOT NULL DEFAULT false
);
