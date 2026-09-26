-- Sample content for local development so the home page renders with data.
-- Never run against a real environment.

TRUNCATE news_posts, raids, streams, characters;

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

INSERT INTO streams (id, streamer_name, game_name, viewer_count, thumbnail_url, channel_url, is_live)
VALUES
    ('c1111111-1111-1111-1111-111111111111',
     'Thundermane', 'World of Warcraft: Forever', 1240, NULL, 'https://twitch.tv/thundermane', true);

INSERT INTO characters (id, name, secondary_name, realm, class, spec, role, is_main, raid_team)
VALUES
    ('d1111111-1111-1111-1111-111111111111', 'Thundermane', 'Ironhide', 'Emberreach', 'Warrior', 'Protection', 'tank', true, 'Team 1'),
    ('d2222222-2222-2222-2222-222222222222', 'Aeliana', 'Dawnsong', 'Emberreach', 'Priest', 'Holy', 'healer', true, 'Team 1'),
    ('d3333333-3333-3333-3333-333333333333', 'Brannor', 'Wildmane', 'Emberreach', 'Druid', 'Balance', 'dps', true, 'Team 1'),
    ('d4444444-4444-4444-4444-444444444444', 'Zephyrion', 'Frostwind', 'Emberreach', 'Mage', 'Frost', 'dps', true, 'Team 1'),
    ('d5555555-5555-5555-5555-555555555555', 'Yorick', 'Nightblade', 'Emberreach', 'Rogue', 'Assassination', 'dps', true, 'Team 2'),
    ('d6666666-6666-6666-6666-666666666666', 'Thunderalt', 'Ironhide', 'Emberreach', 'Paladin', 'Protection', 'tank', false, NULL);
