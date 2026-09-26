ALTER TABLE characters
    DROP CONSTRAINT characters_name_secondary_name_key,
    DROP CONSTRAINT characters_secondary_name_length,
    DROP CONSTRAINT characters_name_length,
    DROP COLUMN secondary_name,
    ADD CONSTRAINT characters_name_realm_key UNIQUE (name, realm);
