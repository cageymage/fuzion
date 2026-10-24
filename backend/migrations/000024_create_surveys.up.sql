CREATE TABLE surveys (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    title       text        NOT NULL,
    description text        NOT NULL DEFAULT '',
    closes_at   timestamptz NOT NULL,
    created_by  uuid REFERENCES users (id) ON DELETE SET NULL,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE survey_questions (
    id        uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    survey_id uuid    NOT NULL REFERENCES surveys (id) ON DELETE CASCADE,
    position  integer NOT NULL,
    prompt    text    NOT NULL,
    type      text    NOT NULL CHECK (type IN ('single', 'multi')),
    required  boolean NOT NULL DEFAULT true,
    options   jsonb   NOT NULL CHECK (jsonb_typeof(options) = 'array'),
    UNIQUE (survey_id, position)
);

CREATE TABLE survey_responses (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    survey_id         uuid        NOT NULL REFERENCES surveys (id) ON DELETE CASCADE,
    user_id           uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    guild_name        text        NOT NULL DEFAULT '',
    discord_username  text        NOT NULL,
    in_discord_server boolean     NOT NULL,
    voided_at         timestamptz,
    voided_by         uuid REFERENCES users (id) ON DELETE SET NULL,
    void_reason       text        NOT NULL DEFAULT '',
    submitted_at      timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    UNIQUE (survey_id, user_id)
);

CREATE TABLE survey_answers (
    response_id uuid   NOT NULL REFERENCES survey_responses (id) ON DELETE CASCADE,
    question_id uuid   NOT NULL REFERENCES survey_questions (id) ON DELETE CASCADE,
    chosen      jsonb  NOT NULL CHECK (jsonb_typeof(chosen) = 'array'),
    PRIMARY KEY (response_id, question_id)
);
