# fuziongaming.gg hub + guild on forever.fuziongaming.gg

## Goal

`fuziongaming.gg` becomes a hub for all Fuzion content. The WoW Forever guild site moves to `forever.fuziongaming.gg`. A possible retail guild site later gets `retail.fuziongaming.gg`.

## Decisions

| Question | Decision | Why |
|---|---|---|
| Subdomain or path (`/forever`) | Subdomain | The app assumes it owns `/`: Render rewrites, `/api`, sitemap, OG tags, host-only cookies. A subdomain needs only config changes. A path would need a Vite `base`, a router `basename`, re-prefixed rewrites and a proxy in front of two apps. |
| Discord login across sites | One Discord application, one redirect URI per site, host-only session per site | Users are keyed by `discord_id`, so every site sees the same identity. Discord skips the consent screen after the first time. Roles stay per site. Real SSO (a central auth service, a domain-wide cookie) waits until a second site needs login. |
| Hub backend | No new API or database. The hub is a static site that proxies selected public endpoints to the existing `fuzion-api` ("option 2") | Costs nothing extra. The hub's backend stays undecided: it only knows its rewrite targets, so pointing it at a different backend later is a render.yaml change. |
| Site-keyed data (`site_id`, `user_site_roles`) | Not now | Only needed once a second site has editable content or a login ("option 3"). See [When to revisit](#when-to-revisit). |
| Where the hub code lives | `hub/` in this repo, next to `frontend/` | One render.yaml deploys everything, CI covers it, and it can copy the frontend's API types and styles. Revisit if the hub turns out to share nothing with the guild site. |

Nobody uses the site yet, so there is no cutover plan for live users and no long-lived apex redirect.

## PR split

Two stacked PRs:

1. **Guild moves to `forever.`** Mostly config and docs. Deployed and checked alone (login, Battle.net link and news previews work on the new host), so a broken login can only come from this PR.
2. **Hub scaffold + Legal.** A new app with its own test setup, CI job and Legal pages. It is a large diff of a different kind and is easier to review apart from the config change.

One PR would also work given no users. Splitting keeps "did the move break auth?" separate from reviewing a new app.

---

## PR 1 (#153): `chore: move the guild site to forever.fuziongaming.gg`

### Code

- [frontend/vite.config.ts](../frontend/vite.config.ts): change the `SITE_BASE_URL` fallback to `https://forever.fuziongaming.gg`.
- [render.yaml](../render.yaml):
  - `fuzion-web`: add `SITE_BASE_URL` (`sync: false`). The static build reads it for canonical, OG, robots.txt and sitemap.xml. Without it the build falls back to the default.
  - Update the example URLs in comments (`DISCORD_REDIRECT_URL`, `BATTLENET_REDIRECT_URL`, `SITE_BASE_URL`) to `forever.fuziongaming.gg`.
- [README.md](../README.md), [backend/README.md](../backend/README.md): update the prod host wherever it is named.
- [frontend/src/lib/siteFiles.test.ts](../frontend/src/lib/siteFiles.test.ts): switch the test input to `https://forever.fuziongaming.gg` so the fixtures match the real host. The behavior is unchanged.

No backend code changes. `SITE_BASE_URL` already drives application and news Discord links, news link previews, and the `index.html` shell the API wraps around news pages ([cmd/api/main.go:127](../backend/cmd/api/main.go#L127)).

### Dashboard steps (in this order)

Register the new hosts before switching env vars. Otherwise login fails with "state mismatch" in between.

1. **DNS**: CNAME `forever` → `fuzion-web.onrender.com`.
2. **Render, `fuzion-web`**: add custom domain `forever.fuziongaming.gg` and wait for the certificate. Leave the apex attached until PR 2 deploys the hub.
3. **Discord developer portal**: add redirect `https://forever.fuziongaming.gg/api/auth/callback`. Rename the application to "Fuzion Gaming", because the name shows on every site's consent screen. Credentials don't change.
4. **Blizzard developer portal**: add redirect `https://forever.fuziongaming.gg/api/auth/battlenet/callback`.
5. **Cloudflare Turnstile**: add `forever.fuziongaming.gg` to the widget's hostnames. Keys don't change.
6. **Render env**:
   - `fuzion-api`: `SITE_BASE_URL`, `DISCORD_REDIRECT_URL`, `BATTLENET_REDIRECT_URL` → `https://forever.fuziongaming.gg/...`
   - `fuzion-web`: `SITE_BASE_URL=https://forever.fuziongaming.gg`, then redeploy, since the value is baked in at build time.
7. **Raid-Helper** (if the webhook is set up): point the webhook URL at `https://forever.fuziongaming.gg/api/webhooks/raid-helper`.

No secrets are rotated. The database and other credentials stay as they are.

### Verify on `forever.fuziongaming.gg`

- [ ] `/api/health` returns `{"status":"ok"}`
- [ ] Discord login round-trips (everyone logs in again once, because cookies are per host)
- [ ] Battle.net link round-trips
- [ ] The application form submits (Turnstile passes)
- [ ] The Twitch embed loads (its `parent` comes from `window.location.hostname`, so nothing to change)
- [ ] View source on `/` and on a `/news/<id>` page: canonical and `og:url` use `forever.`
- [ ] `/robots.txt` and `/sitemap.xml` use `forever.`
- [ ] Publishing a news post sends a Discord announcement linking to `forever.`

---

## PR 2 (#154): `feat(hub): fuziongaming.gg hub site` (stacked on PR 1)

### Hub app (`hub/`)

- A Vite + React + TS app with the same toolchain as `frontend/` (Vitest + RTL + MSW and the same test conventions from CLAUDE.md).
- Start small: a landing page (who Fuzion is, links to Discord, Twitch, YouTube and `forever.fuziongaming.gg`) and a Legal page. Add the live-streams / suggested-video section only if wanted. It reads `GET /api/streams/live` and `GET /api/streams/suggested-video`, both public.
- Page and component tests ship with each page, per CLAUDE.md.
- Its own `SITE_BASE_URL` (`https://fuziongaming.gg`) for canonical and OG tags. Add a robots.txt and sitemap only if they're worth it for a page this small.

### Wiring

- **render.yaml**: new static service `fuzion-hub` (`rootDir: hub`). Only rewrite the API paths the hub uses, not all of `/api/*`. Otherwise the hub host would also expose the auth callbacks and officer endpoints, which only work on the guild host:
  ```yaml
  routes:
    - type: rewrite
      source: /api/streams/*
      destination: https://fuzion-api.onrender.com/api/streams/*
    - type: rewrite
      source: /*
      destination: /index.html
  ```
- **Root `package.json`**: `dev:hub` (Vite on port 5174, proxying `/api` to `127.0.0.1:8080` like the frontend), `test:hub`, and add the hub to `postinstall` and `test`. Decide whether `npm run dev` starts the hub too or it stays opt-in.
- **CI** ([.github/workflows/test.yml](../.github/workflows/test.yml)): a `hub` job matching `web` (`npm ci`, build, test).
- **Docs**: add `hub/` to [plans/project-structure.md](project-structure.md) and the README's deploy section. Mention the hub's Legal page in the CLAUDE.md Legal section.

### Legal

**Guild page** ([frontend/src/pages/Legal/Legal.tsx](../frontend/src/pages/Legal/Legal.tsx)):

- "The Fuzion site collects…" → name the host this policy covers, `forever.fuziongaming.gg`.
- Add a line saying `fuziongaming.gg` and other `*.fuziongaming.gg` sites are run by the same people, have their own notices, and that logging in with Discord on one site does not log you in on another.
- Tests in `Legal.test.tsx`, for example:
  - `should name forever.fuziongaming.gg as the site the privacy policy covers when the page renders`
  - `should say other fuziongaming.gg sites have their own notices when the page renders`

**Hub page** (`hub/src/pages/Legal/`):

- Privacy: the hub has no login, forms or uploads, so it collects no personal data itself. Third parties: Render (hosting) and, if streams are embedded, Twitch and YouTube, which may set their own cookies. Link to the guild page for what the guild site collects.
- Copyright / content notice: Fuzion branding, and a removal-request route (Discord).
- A Blizzard non-affiliation notice if the hub mentions WoW or shows game content.
- Attributions: only the assets the hub actually uses.
- Tests: section headings render, the named third parties appear in privacy, and the "no personal data collected" statement appears.

The PR description should say what changed on both Legal pages, per CLAUDE.md.

### Dashboard steps (after PR 2 merges)

1. **Render**: create `fuzion-hub` (the Blueprint sync picks it up from render.yaml). Set `SITE_BASE_URL=https://fuziongaming.gg`.
2. **Render**: remove the apex `fuziongaming.gg` from `fuzion-web` and add it, plus `www`, to `fuzion-hub`. Update the apex DNS records to the values Render shows.
3. **Discord developer portal**: remove the old `https://fuziongaming.gg/api/auth/callback` redirect. **Blizzard**: remove the old Battle.net redirect. **Turnstile**: remove the apex hostname.

### Verify

- [ ] `fuziongaming.gg` serves the hub. Links to `forever.` work.
- [ ] If the streams section is included, live streams and the suggested video load through the hub's rewrite.
- [ ] `fuziongaming.gg/api/auth/login` and `fuziongaming.gg/api/applications` are **not** proxied (they return the hub's SPA or a 404, not the API).
- [ ] Both Legal pages render the new wording.

---

## When to revisit

Move to site-keyed data ("option 3") at the first hub, or retail, feature that needs editable content, a login, or per-site roles. Scope at that point:

- A `sites` table (slug, hostname, per-site settings that are env vars today: webhooks, Raid-Helper secret, Blizzard guild).
- `site_id` only on the tables the new site shares (probably `news_posts`, `images`). Guild-only tables (roster, applications, raids, professions) wait until a second guild exists.
- `users` and `sessions` stay global. `is_admin` and `is_officer` move to `user_site_roles(user_id, site_id, …)`, and the first-admin auto-grant becomes per site.
- Middleware resolves the site from each site's rewrite path, because Render's rewrite means the API sees its own hostname.
- An existing-rows backfill to `forever`.
