-- Null for hand-entered raids, which the Raid-Helper webhook never touches.
ALTER TABLE raids ADD COLUMN raid_helper_event_id text UNIQUE;

-- Raid-Helper's lastUpdated (Unix seconds) of the delivery last applied. A delivery
-- strictly older than this is skipped; equal ones are still applied.
ALTER TABLE raids ADD COLUMN raid_helper_last_updated bigint;

-- A deleted event keeps its row as a tombstone: Raid-Helper sends no ordering signal,
-- so a create or update arriving after the delete must not bring the raid back.
ALTER TABLE raids ADD COLUMN deleted_at timestamptz;
