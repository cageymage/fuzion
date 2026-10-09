-- The recruiting-channel message announcing the application, kept so the
-- message can be edited when an officer reviews it. Null for applications
-- submitted before this existed or while notifications were disabled.
ALTER TABLE applications
    ADD COLUMN discord_message_id text;
