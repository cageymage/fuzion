ALTER TABLE news_posts ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE raids ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE streams ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE users ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE characters ALTER COLUMN id SET DEFAULT gen_random_uuid();
