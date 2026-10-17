ALTER TABLE applications
    ADD COLUMN playstyle text NOT NULL DEFAULT '' CHECK (playstyle IN ('', 'casual', 'raider'));
