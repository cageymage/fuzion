-- Alts are not linked to a specific main character row on purpose: once
-- Battle.net character claiming (#34) lands, a member's alts and main are
-- grouped implicitly by sharing the same user_id. Adding a hand-maintained
-- main_character_id now would just be a second, competing source of truth.
-- Wanted eventually as a UI nicety before #34 ships, but not in scope here.
CREATE TABLE characters (
    id         uuid PRIMARY KEY,
    name       text        NOT NULL,
    realm      text        NOT NULL DEFAULT 'Emberreach',
    class      text        NOT NULL CHECK (class IN (
                   'Warrior', 'Paladin', 'Hunter', 'Rogue', 'Priest',
                   'Shaman', 'Mage', 'Warlock', 'Druid'
               )),
    spec       text,
    -- Forever allows role/class combinations vanilla classic didn't (e.g. a
    -- class that couldn't tank or heal there might be able to here), so role
    -- is deliberately its own unrestricted-by-class check, not a per-class enum.
    role       text        NOT NULL CHECK (role IN ('tank', 'healer', 'dps')),
    is_main    boolean     NOT NULL DEFAULT true,
    raid_team  text,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (name, realm)
);
