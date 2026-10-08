-- Race and level are officer-entered and nullable until an officer sets them.
-- Skyborne can be either faction, so each faction's variant is its own race
-- value; that keeps faction derivable from race without a separate column.
ALTER TABLE characters
    ADD COLUMN race text CHECK (race IN (
        'Human', 'Dwarf', 'Night Elf', 'Gnome', 'Skyborne (High Order)',
        'Orc', 'Undead', 'Tauren', 'Troll', 'Skyborne (Windshaper)'
    )),
    ADD COLUMN level smallint CHECK (level BETWEEN 1 AND 60);
