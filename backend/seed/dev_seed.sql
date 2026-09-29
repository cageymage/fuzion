-- Sample content for local development so the home page renders with data.
-- Never run against a real environment.

TRUNCATE news_posts, raids, streams, characters, professions, raid_bosses, raid_tiers;

INSERT INTO news_posts (id, title, excerpt, category, image_url, author_name, published_at)
VALUES
    ('a1111111-1111-1111-1111-111111111111',
     'Fuzion defeated Queen Ansurek on Mythic',
     'Mythic Nerubar Palace, 8/8 down - a huge night for the team after weeks on the enrage.',
     'raid-progress', NULL, 'Officer', now() - interval '2 days'),
    ('a2222222-2222-2222-2222-222222222222',
     'Now recruiting: Restoration Druid & Fire Mage',
     'Two raid spots open for the Mythic roster.',
     'recruitment', NULL, 'Officer', now() - interval '4 days'),
    ('a3333333-3333-3333-3333-333333333333',
     'Welcome our newest officers',
     'Two promotions from the raid team.',
     'guild-news', NULL, 'Officer', now() - interval '6 days');

INSERT INTO raids (id, difficulty, instance_name, starts_at, progress_summary)
VALUES
    ('b1111111-1111-1111-1111-111111111111',
     'Mythic', 'Nerubar Palace', now() + interval '2 hours 14 minutes', '8/8 Heroic cleared');

INSERT INTO streams (id, streamer_name, game_name, viewer_count, thumbnail_url, channel_url, is_live, twitch_login)
VALUES
    ('c1111111-1111-1111-1111-111111111111',
     'Thundermane', 'World of Warcraft: Forever', 1240, NULL, 'https://twitch.tv/thundermane', true, 'thundermane'),
    ('c2222222-2222-2222-2222-222222222222',
     'Centrifuze', 'World of Warcraft', 87, 'https://static-cdn.jtvnw.net/previews-ttv/live_user_centrifuze-440x248.jpg', 'https://www.twitch.tv/centrifuze', true, 'centrifuze'),
    ('c3333333-3333-3333-3333-333333333333',
     'Moonveil', '', 0, NULL, 'https://twitch.tv/moonveil', false, 'moonveil');

INSERT INTO characters (id, name, secondary_name, realm, class, spec, role, is_main, raid_team)
VALUES
    ('d1111111-1111-1111-1111-111111111111', 'Thundermane', 'Ironhide', 'Emberreach', 'Warrior', 'Protection', 'tank', true, 'Team 1'),
    ('d2222222-2222-2222-2222-222222222222', 'Aeliana', 'Dawnsong', 'Emberreach', 'Priest', 'Holy', 'healer', true, 'Team 1'),
    ('d3333333-3333-3333-3333-333333333333', 'Brannor', 'Wildmane', 'Emberreach', 'Druid', 'Balance', 'dps', true, 'Team 1'),
    ('d4444444-4444-4444-4444-444444444444', 'Zephyrion', 'Frostwind', 'Emberreach', 'Mage', 'Frost', 'dps', true, 'Team 1'),
    ('d5555555-5555-5555-5555-555555555555', 'Yorick', 'Nightblade', 'Emberreach', 'Rogue', 'Assassination', 'dps', true, 'Team 2'),
    ('d6666666-6666-6666-6666-666666666666', 'Thunderalt', 'Ironhide', 'Emberreach', 'Paladin', 'Protection', 'tank', false, NULL);

-- Tier 1 of real WoW Forever launch content (icy-veins.com and
-- blizzardwatch.com, Sep 2026): three raids released together, all current
-- at once - Forever ships at least two raids per tier rather than one at a
-- time. Two fictional retired tiers are kept below as history examples.
-- sort_order is one global sequence, not reset per is_current group (nothing
-- in the app auto-renumbers it when isCurrent flips), and displays highest
-- first - a newly added raid should get the next highest number so it
-- naturally lands at the top without renumbering anything else.
INSERT INTO raid_tiers (id, name, is_current, sort_order)
VALUES
    ('e5555555-5555-5555-5555-555555555555', 'Sunken Reliquary', false, 0),
    ('e4444444-4444-4444-4444-444444444444', 'Shattered Spire', false, 1),
    ('e3333333-3333-3333-3333-333333333333', 'Onyxia''s Lair', true, 2),
    ('e2222222-2222-2222-2222-222222222222', 'Hyjal Summit', true, 3),
    ('e1111111-1111-1111-1111-111111111111', 'Barrow Deeps', true, 4);

INSERT INTO raid_bosses (id, tier_id, name, sort_order, killed_at)
VALUES
    -- Barrow Deeps (10-player), 8 bosses in kill order.
    ('fb000000-0000-0000-0000-000000000001', 'e1111111-1111-1111-1111-111111111111', 'Deepscar Matriarch', 0, NULL),
    ('fb000000-0000-0000-0000-000000000002', 'e1111111-1111-1111-1111-111111111111', 'Elder Tangleclaw', 1, NULL),
    ('fb000000-0000-0000-0000-000000000003', 'e1111111-1111-1111-1111-111111111111', 'Khalith the Dreadspinner', 2, NULL),
    ('fb000000-0000-0000-0000-000000000004', 'e1111111-1111-1111-1111-111111111111', 'Well of Sorrow', 3, NULL),
    ('fb000000-0000-0000-0000-000000000005', 'e1111111-1111-1111-1111-111111111111', 'Amethrax', 4, NULL),
    ('fb000000-0000-0000-0000-000000000006', 'e1111111-1111-1111-1111-111111111111', 'Del''lynar Songwood', 5, NULL),
    ('fb000000-0000-0000-0000-000000000007', 'e1111111-1111-1111-1111-111111111111', 'Ravus and Darlissa', 6, NULL),
    ('fb000000-0000-0000-0000-000000000008', 'e1111111-1111-1111-1111-111111111111', 'Sonya Darkhallow', 7, NULL),
    -- Hyjal Summit (20-player), 13 bosses in kill order.
    ('fc000000-0000-0000-0000-000000000001', 'e2222222-2222-2222-2222-222222222222', 'Bandalar', 0, NULL),
    ('fc000000-0000-0000-0000-000000000002', 'e2222222-2222-2222-2222-222222222222', 'Ancient of Decay', 1, NULL),
    ('fc000000-0000-0000-0000-000000000003', 'e2222222-2222-2222-2222-222222222222', 'Time-Lost Battalion', 2, NULL),
    ('fc000000-0000-0000-0000-000000000004', 'e2222222-2222-2222-2222-222222222222', 'Sylvestris Dusksong', 3, NULL),
    ('fc000000-0000-0000-0000-000000000005', 'e2222222-2222-2222-2222-222222222222', 'Old Gloomlurker', 4, NULL),
    ('fc000000-0000-0000-0000-000000000006', 'e2222222-2222-2222-2222-222222222222', 'Gharalis the Abyssal', 5, NULL),
    ('fc000000-0000-0000-0000-000000000007', 'e2222222-2222-2222-2222-222222222222', 'Kathris the Haunted', 6, NULL),
    ('fc000000-0000-0000-0000-000000000008', 'e2222222-2222-2222-2222-222222222222', 'Anara Chillwind', 7, NULL),
    ('fc000000-0000-0000-0000-000000000009', 'e2222222-2222-2222-2222-222222222222', 'Elder Minderel', 8, NULL),
    ('fc00000a-0000-0000-0000-000000000010', 'e2222222-2222-2222-2222-222222222222', 'Tracker Stillwind', 9, NULL),
    ('fc00000b-0000-0000-0000-000000000011', 'e2222222-2222-2222-2222-222222222222', 'Council of Thorns', 10, NULL),
    ('fc00000c-0000-0000-0000-000000000012', 'e2222222-2222-2222-2222-222222222222', 'Nythus the Dreadmbound', 11, NULL),
    ('fc00000d-0000-0000-0000-000000000013', 'e2222222-2222-2222-2222-222222222222', 'The Wild King', 12, NULL),
    -- Onyxia's Lair is a single-boss raid.
    ('f2222222-2222-2222-2222-222222222222', 'e3333333-3333-3333-3333-333333333333', 'Onyxia', 0, NULL),
    -- Shattered Spire (finished, ~1 month ago).
    ('f7777777-7777-7777-7777-777777777777', 'e4444444-4444-4444-4444-444444444444', 'Voidshard Sentinel', 0, now() - interval '30 days'),
    ('f8888888-8888-8888-8888-888888888888', 'e4444444-4444-4444-4444-444444444444', 'Thornqueen Ilyra', 1, now() - interval '28 days'),
    ('f9999999-9999-9999-9999-999999999999', 'e4444444-4444-4444-4444-444444444444', 'The Hollow King', 2, now() - interval '25 days'),
    -- Sunken Reliquary (finished, ~3 months ago).
    ('fa111111-1111-1111-1111-111111111111', 'e5555555-5555-5555-5555-555555555555', 'Tideborn Warden', 0, now() - interval '90 days'),
    ('fa222222-2222-2222-2222-222222222222', 'e5555555-5555-5555-5555-555555555555', 'Coral Matriarch', 1, now() - interval '88 days'),
    ('fa333333-3333-3333-3333-333333333333', 'e5555555-5555-5555-5555-555555555555', 'Abyssal Choir', 2, now() - interval '85 days'),
    ('fa444444-4444-4444-4444-444444444444', 'e5555555-5555-5555-5555-555555555555', 'Drowned King Maros', 3, now() - interval '80 days');
