ALTER TABLE applications
    ADD COLUMN reviewed_by  uuid REFERENCES users (id),
    ADD COLUMN reviewed_at  timestamptz,
    ADD COLUMN review_note  text NOT NULL DEFAULT '';
