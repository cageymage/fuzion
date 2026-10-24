Real Raid-Helper webhook bodies, captured on the Fuzion server on 2026-10-12 and redacted
(issue #8). They are copies of `plans/integrations/raid-helper-samples/`, which documents
how each was captured and what was replaced, except that `channelId` here is the real
WoW Forever raid-planning channel, so the handler mirrors them (issue #159). All five describe the same event; the
`lastUpdated` values rise from create to signup to absence, and the delete repeats the
absence update's value.
