# Fuzion frontend

React 18 + TypeScript + Vite client for the Fuzion guild site. The visual
theme replicates the `Main.dc.html` design mockup (dark/gold "tavern"
palette, Cinzel + Crimson Pro, WoW item-rarity colors for category chips).

## Commands

```
npm install
npm run dev        # Vite dev server on :5173, /api proxied to :8080
npm test           # Vitest (MSW-intercepted HTTP, jsdom)
npm run typecheck  # tsc --noEmit
npm run build      # typecheck + production build to dist/
```

`npm run dev` expects the Go backend on `http://localhost:8080` — from the
repo root, `npm run dev` starts both together. Without the backend the page
shell renders with per-section empty/error states; MSW is wired for Vitest
only, never for the browser.

## Theme tokens

All colors, fonts, radii and the page gutter live as CSS custom properties
in `src/app/theme.css`, with `.card` and `.eyebrow` as the two shared
utility classes the design repeats.
