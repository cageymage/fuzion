-- Only the Blizzard roster sync sets this, to choose the male or female race icon. NULL
-- means Blizzard has not been asked yet or has no profile for the character.
ALTER TABLE characters ADD COLUMN gender text CHECK (gender IN ('male', 'female'));
