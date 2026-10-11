# Fuzion: Design Notes

Visual and accessibility decisions for the frontend. The tokens live in `frontend/src/app/theme.css`; this page records what they are for and the rules that keep them consistent. Stack and layout are in [project-structure.md](project-structure.md).

## Type

Fonts come from Google Fonts (see `frontend/index.html`): **Cinzel** (display) for headings, the wordmark, chips and table headers, and **Crimson Pro** (body) for everything else. Crimson Pro has a small x-height, so it reads smaller than its pixel size suggests; the scale below is deliberately larger than a typical site.

All sizes are in `rem` so they follow the browser's default font size.

| Token | Size | Used for |
|---|---|---|
| `--text-label` | 0.875rem (14px) | Uppercase Cinzel labels: chips, badges, filters, table headers |
| `--text-meta` | 1.0625rem (17px) | Secondary text: meta lines, notices, form fields, footer |
| `--text-ui` | 1.125rem (18px) | Nav links, card titles, inline links |
| `--text-body` | 1.1875rem (19px) | Page default, `line-height: 1.6` |

Headings: page titles 38px, home headline 52px, section headings 1.625rem, wordmark 1.5rem. Cinzel stays on headings and labels only; news card titles and raid boss names use sentence-case Crimson Pro semibold so Cinzel keeps working as an accent. Do not add new `px` font sizes; pick a token.

## Color

| Token | Value | Notes |
|---|---|---|
| `--bg` | `#15100b` | Page background |
| `--surface` | `#201811` | Cards |
| `--border` | `#3c2d1e` | Card and table borders, progress track |
| `--border-strong` | `#7d6542` | Form controls and disabled buttons (about 3:1 against the background) |
| `--text` | `#f1e6d6` | Main text |
| `--muted` | `#a8977f` | Secondary text (about 6:1) |
| `--gold`, `--gold-hover`, `--gold-soft` | `#d3a24d`, `#e8c27a`, `#3a2c17` | Accent, links, primary actions |
| `--live` | `#c23b3b` | Live badges, dots and danger borders. **Not for text** |
| `--danger` | `#e06666` | Red text (errors, danger buttons), about 5:1 |

Rules:

- Text must reach 4.5:1 against its background and non-text UI (borders, focus rings) 3:1. Check new colors with axe before shipping.
- Red text uses `--danger`; `--live` fails contrast as text.
- WoW class colors (`frontend/src/lib/classColor.ts`) follow Blizzard's palette except Shaman, which is lightened to `#2E8CF0` because Blizzard's `#0070DE` is 3.6:1 on the card surface. `classColor.test.ts` fails if any class drops below 4.5:1.
- Unkilled bosses are 0.75 opacity; do not go lower.

## Layout and responsiveness

- `--gutter` is 56px, 16px below 720px. Radii: `--radius-card` 6px, `--radius-media` 4px, `--radius-chip` 3px.
- **900px:** the header nav collapses behind a Menu button (`aria-expanded`, Escape and link click close it), and raid progress goes to one column.
- Roster tables scroll horizontally in a labelled, keyboard-focusable region instead of clipping on phones.
- Tests run in jsdom, which applies CSS but ignores media queries. Write CSS so the desktop-first default is what jsdom sees: hide things behind `min-width` queries rather than hiding by default and showing in a `max-width` query, or role queries in tests will not find them.

## Links, controls and motion

- Inline links are 18px with a soft gold underline. Form fields use `--border-strong`.
- Role options on the application form are button-style choices around a visually hidden native radio, so keyboard and screen reader behavior is unchanged. This uses `:has()`.
- Disabled buttons keep full opacity with muted text and `--border-strong` (opacity alone made them too faint to read).
- `theme.css` has a global `prefers-reduced-motion: reduce` rule that collapses animation and transition durations and turns off smooth scroll. New motion does not need its own guard.
- The raid progress bar is a native `<progress>` styled with `::-webkit-progress-*` and `::-moz-progress-bar`. Checked in Chrome and Firefox.

## Logo and image assets

The logo is a painted raster (a gold atom with blue orbs inside a rune ring), so there is no SVG master and no `favicon.svg`; that requirement was dropped. The source is a 1254x1254 PNG kept outside the repo.

| Asset | Where | Notes |
|---|---|---|
| `src/assets/logo-48.webp`, `-96`, `-144` | Header, 48px at 1x, 2x, 3x via `srcSet` | Alt text is empty; the wordmark carries the name |
| `public/favicon.ico`, `favicon-32.png`, `favicon-192.png`, `apple-touch-icon.png` | Linked from `index.html` | |
| `public/logo-512.png` | Not referenced by the site | 512x512 for the Discord server icon and social profiles; also usable as a manifest icon later |
| `public/og-image.jpg` | Link previews | 1200x630 JPEG. Must be JPEG or PNG (WebP, AVIF and SVG are unreliable for crawlers) |
| `src/assets/hero-desktop.*`, `hero-mobile.*` | Home hero via `<picture>` (AVIF, WebP, JPEG) | 2400x900 and 900x600; dark on the left and right where text and the Next Raid card sit |
| `src/assets/news/*` | News card backdrops by category | |

Uploaded news images are stored lossless as WebP at up to 2560px with an 800px thumbnail. Cards, link previews and the lightbox preview use the thumbnail (`lib/uploadedImage.ts`); only the lightbox's expanded view loads the full image.

The hero artwork is AI-generated (Midjourney). The Legal page carries the attribution as voluntary disclosure; Midjourney's terms allow commercial use with no attribution requirement. Do not use Blizzard promotional art, class icons or boss portraits without checking usage first.

## Link previews

Discord, Bluesky and similar services read raw HTML and do not run JavaScript. `index.html` carries site-wide `og:*`, `twitter:card`, `description` and `canonical` tags, between `<!--og:start-->` and `<!--og:end-->`, with `{{SITE_BASE_URL}}` filled in at build time (default `https://fuziongaming.gg`, override with the `SITE_BASE_URL` environment variable). For `/news/:id` the API swaps that block for the post's own title, excerpt and first image (see the Hosting section of [project-structure.md](project-structure.md)). Discord caches embeds, so test changes with a `?v=2` URL.
