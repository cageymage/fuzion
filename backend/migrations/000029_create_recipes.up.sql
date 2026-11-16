-- id is Blizzard's recipe id from the profile API, which is not Wowhead's spell id.
CREATE TABLE recipes (
    id         integer PRIMARY KEY,
    profession text    NOT NULL,
    name       text    NOT NULL
);

CREATE TABLE character_recipes (
    character_id uuid    NOT NULL REFERENCES characters (id) ON DELETE CASCADE,
    recipe_id    integer NOT NULL REFERENCES recipes (id) ON DELETE CASCADE,
    PRIMARY KEY (character_id, recipe_id)
);

CREATE INDEX character_recipes_recipe_id_idx ON character_recipes (recipe_id);
