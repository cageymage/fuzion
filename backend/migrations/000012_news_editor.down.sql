DELETE FROM news_posts WHERE published_at IS NULL OR category = 'patch-notes';

ALTER TABLE news_posts DROP CONSTRAINT news_posts_category_check;
ALTER TABLE news_posts ADD CONSTRAINT news_posts_category_check
    CHECK (category IN ('raid-progress', 'recruitment', 'guild-news'));

ALTER TABLE news_posts
    ALTER COLUMN published_at SET NOT NULL,
    DROP COLUMN updated_at,
    DROP COLUMN author_user_id,
    DROP COLUMN pinned,
    DROP COLUMN body;
