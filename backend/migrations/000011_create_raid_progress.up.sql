CREATE TABLE raid_tiers (
    id         uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    name       text        NOT NULL,
    is_current boolean     NOT NULL DEFAULT false,
    sort_order integer     NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE raid_bosses (
    id         uuid    PRIMARY KEY DEFAULT gen_random_uuid(),
    tier_id    uuid    NOT NULL REFERENCES raid_tiers (id) ON DELETE CASCADE,
    name       text    NOT NULL,
    sort_order integer NOT NULL,
    killed_at  timestamptz,
    UNIQUE (tier_id, sort_order)
);
