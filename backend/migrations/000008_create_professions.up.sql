CREATE TABLE professions (
    id           uuid    PRIMARY KEY,
    character_id uuid    NOT NULL REFERENCES characters (id) ON DELETE CASCADE,
    profession   text    NOT NULL,
    skill_level  integer NOT NULL CHECK (skill_level >= 0),
    UNIQUE (character_id, profession)
);
