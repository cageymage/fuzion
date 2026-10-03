-- Forever characters can split talent points across two trees, and the role can
-- differ per spec. Two specs is the ceiling, so these are columns rather than a
-- child table. The pair is all-or-nothing: a second role without a second spec
-- (or the reverse) has no meaning.
ALTER TABLE characters
    ADD COLUMN spec2 text,
    ADD COLUMN role2 text CHECK (role2 IN ('tank', 'healer', 'dps')),
    ADD CONSTRAINT characters_second_spec_pair CHECK ((spec2 IS NULL) = (role2 IS NULL));
