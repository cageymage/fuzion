-- The member who owns the character, set when they link a Battle.net account that
-- has it (#34). A member's main and alts are the characters sharing their user_id.
-- Deleting the user keeps the character on the roster, just unowned.
ALTER TABLE characters ADD COLUMN user_id uuid REFERENCES users (id) ON DELETE SET NULL;

CREATE INDEX characters_user_id_idx ON characters (user_id);
