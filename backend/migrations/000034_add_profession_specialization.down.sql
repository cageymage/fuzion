ALTER TABLE professions
    DROP CONSTRAINT professions_specialization_has_source,
    DROP COLUMN specialization_source,
    DROP COLUMN specialization;
