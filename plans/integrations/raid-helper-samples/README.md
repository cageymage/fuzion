# Raid-Helper webhook samples

Real webhook bodies captured from the Fuzion server on 2026-10-12 (Premium active), redacted, as valid JSON for use as fixtures in #31. Notes for how they were captured and what each one is are in `../raid-helper.md` (section 2.4b).

## Files

| File | Request type (`x-raidhelper-event`) | What produced it | Provenance |
|---|---|---|---|
| `webhookeventcreate.json` | `webhookeventcreate` | Creating an event; no signups | Real capture. Compared by eye with the update sent together with it: same fields, same `lastUpdated`, no signups. The two files have the same content |
| `webhookeventupdate-after-create.json` | `webhookeventupdate` | The update Raid-Helper sends together with the create, same moment | Real capture |
| `webhookeventupdate-signup.json` | `webhookeventupdate` | Title edited, and one Healer signup added | Real capture |
| `webhookeventupdate-absence.json` | `webhookeventupdate` | The same signup changed to Absence | Real capture |
| `webhookeventdelete.json` | `webhookeventdelete` | Deleting the event | Real capture (body identical to the absence update, see caveats) |
| `headers.json` | all three | The two request headers known to matter | Authorization value is a **fake test key** |

`comp.update` was never captured (no URL was registered for it) and is not needed for the Calendar, so there is no sample for it.

## Redactions

All personal and server-specific values were replaced; shapes and types were kept.

| Real value | Replaced with |
|---|---|
| Event (message) id | `1100000000000000001` |
| Server id | `1200000000000000001` |
| Channel id | `1300000000000000001` |
| Leader, creator and signup user id | `1400000000000000001` |
| Leader, creator and signup display name | `Raid Leader` |
| Channel name | `raid-planning` |
| Event title | `Test Raid Event` (`Test Raid Event (edited)` after the edit) |

Left unchanged: emote ids, template id (`wowforever`), timestamps, signup `id` values and `advancedSettings` (97 keys; `image` was the string `"none"`). The `classes` list is the full one the template sent (Tank, Melee, Ranged, Healer, Late, Bench, Tentative, Absence).

## Headers

`headers.json` holds the headers of a real request, per type (names shown lowercase, as webhook.site displays them; Go's `http.Header` canonicalizes them):
- `content-type`: `application/json`
- `user-agent`: `Raid-Helper/1.0`
- `x-raidhelper-event`: `webhookeventcreate`, `webhookeventupdate` or `webhookeventdelete`
- `authorization`: the bare webhook key, 30 alphanumeric characters (mixed case), no prefix. **The value here, `aB3dE5gH7jK9mN1pQ3sT5vW7yZ9bC1`, is fake**, safe to use as a test key. It is not Fuzion's real key.

Left out on purpose: `content-length` (real one was 13090 for the real body, but these fixtures are redacted and pretty-printed, so a test client should compute its own) and `host` (set by the client from the URL). The real request had no other headers: no signature, timestamp, delivery-id or retry header.

## Caveats for tests

- A create and its accompanying update have identical bodies, so a test that sends both should expect an idempotent upsert, with one row.
- The delete body is the whole event, identical to the last update (including `lastUpdated`). Bodies alone cannot tell the two apart. Tests must dispatch on the `x-raidhelper-event` header (or the URL path).
- `webhookeventupdate-absence.json` shows a signup with only 9 keys (no spec or role fields). `webhookeventupdate-signup.json` shows a Healer signup with spec and role fields. Parse spec and role fields as optional.
- `templateEmoteId` and `softresId` do not appear in any sample (documented but absent). `displayTitle`, `creator` and `announcements` do appear (undocumented).
