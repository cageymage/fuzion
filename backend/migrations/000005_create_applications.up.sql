CREATE TABLE applications (
    id             uuid        PRIMARY KEY,
    applicant_name text        NOT NULL,
    character_name text        NOT NULL,
    class          text        NOT NULL,
    role           text        NOT NULL CHECK (role IN ('tank', 'healer', 'dps')),
    availability   text        NOT NULL,
    discord_handle text        NOT NULL,
    notes          text        NOT NULL DEFAULT '',
    status         text        NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'accepted', 'declined')),
    submitted_at   timestamptz NOT NULL DEFAULT now()
);
