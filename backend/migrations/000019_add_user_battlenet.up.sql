ALTER TABLE users
    ADD COLUMN battlenet_id text UNIQUE,
    ADD COLUMN battlenet_tag text;
