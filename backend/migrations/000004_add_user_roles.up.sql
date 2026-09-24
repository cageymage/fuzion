ALTER TABLE users
    ADD COLUMN is_admin   boolean NOT NULL DEFAULT false,
    ADD COLUMN is_officer boolean NOT NULL DEFAULT false;
