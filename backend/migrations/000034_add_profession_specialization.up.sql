-- specialization_source says who may change it: the Blizzard sync owns 'blizzard' values
-- and leaves 'manual' ones alone unless Blizzard reports a specialization itself.
ALTER TABLE professions
    ADD COLUMN specialization text,
    ADD COLUMN specialization_source text CHECK (specialization_source IN ('blizzard', 'manual')),
    ADD CONSTRAINT professions_specialization_has_source
        CHECK ((specialization IS NULL) = (specialization_source IS NULL));
