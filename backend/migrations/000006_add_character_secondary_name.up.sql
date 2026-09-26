-- Forever characters have a required main and secondary name, each 2-12
-- characters, and the pair is what must be unique. Existing rows predate the
-- secondary name, so they get a placeholder an officer can correct via PATCH.
ALTER TABLE characters ADD COLUMN secondary_name text;
UPDATE characters SET secondary_name = 'Unset';
ALTER TABLE characters
    ALTER COLUMN secondary_name SET NOT NULL,
    ADD CONSTRAINT characters_name_length CHECK (char_length(name) BETWEEN 2 AND 12),
    ADD CONSTRAINT characters_secondary_name_length CHECK (char_length(secondary_name) BETWEEN 2 AND 12),
    DROP CONSTRAINT characters_name_realm_key,
    ADD CONSTRAINT characters_name_secondary_name_key UNIQUE (name, secondary_name);
