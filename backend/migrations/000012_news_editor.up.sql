ALTER TABLE news_posts
    ADD COLUMN body           text        NOT NULL DEFAULT '',
    ADD COLUMN pinned         boolean     NOT NULL DEFAULT false,
    ADD COLUMN author_user_id uuid        REFERENCES users (id) ON DELETE SET NULL,
    ADD COLUMN updated_at     timestamptz NOT NULL DEFAULT now(),
    ALTER COLUMN published_at DROP NOT NULL;

ALTER TABLE news_posts DROP CONSTRAINT news_posts_category_check;
ALTER TABLE news_posts ADD CONSTRAINT news_posts_category_check
    CHECK (category IN ('raid-progress', 'recruitment', 'guild-news', 'patch-notes'));
