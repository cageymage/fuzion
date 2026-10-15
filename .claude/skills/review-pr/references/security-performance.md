# Security and performance checks

Fuzion is a Go (chi, sqlx, Postgres) API with a React/TS (Vite, TanStack Query) frontend and Discord OAuth. Flag only what the diff adds or changes, and give the consequence in plain words.

## Security

Authentication and authorization
- Officer-only routes are wrapped in `auth.RequireOfficer`. A new write route (POST/PUT/PATCH/DELETE) with no auth wrapper is a `must`.
- Member-scoped data: does the query filter by the session's user ID, or can a user pass someone else's ID (IDOR)? Ownership checks belong in the service, not only the UI.
- Session and cookie handling stays in `internal/auth`. Cookies keep `HttpOnly`, `SameSite`, and `Secure` in production. OAuth `state` is verified on callback.
- Do not trust client-sent role flags such as `is_officer` in a request body.

Injection and validation
- SQL values always go through placeholders (`$1`, `:name`). `fmt.Sprintf`, `+`, or string-built values in SQL is a `must`. Dynamic column or order-by names must come from an allowlist.
- Request bodies: size limited, unknown or out-of-range values rejected in `service.go` (lengths, enums, UUID parsing, dates).
- Webhooks from outside (for example Raid-Helper) verify a secret or signature with a constant-time compare before acting.
- Outbound requests built from user input (SSRF): URL host should be fixed or allowlisted.

Data exposure
- Responses do not leak other users' emails, tokens, Discord IDs, or internal errors. Handlers return generic messages for 500s and log the detail.
- No secrets, tokens, or real IDs in code, tests, seed data, logs, or `.env` files being committed. New config comes from the central config load and `.env.example` gets a placeholder.

Uploads (images, files)
- Type checked by content sniffing, not just file extension or client MIME. Size limit enforced. Stored filenames are generated, never taken from the client (path traversal).

Frontend
- `dangerouslySetInnerHTML` or rendering markdown/HTML from users without sanitizing is a `must`.
- Links built from user input use `rel="noopener noreferrer"` and reject `javascript:` URLs.
- Tokens are not put in `localStorage` or URLs. Officer-only UI hidden in the client is not security; the API must enforce it too.

Dependencies
- New dependency: is it well known and maintained, and is it needed? Flag lockfile changes with no matching manifest change.

## Performance

Database
- N+1: a query inside a loop over rows. Suggest a join or a single `WHERE id = ANY($1)`.
- New `WHERE`, `JOIN`, or `ORDER BY` column on a table that can grow (signups, news, raids) with no index in the migration is a `should`.
- List endpoints without a `LIMIT` or pagination are a `should` (a `must` if the table is user-written and unbounded).
- `SELECT *` on wide rows when a few columns are needed; large transactions held open across network calls.
- Calls to Blizzard, Battle.net, Discord, or Raid-Helper inside a request handler with no timeout, or repeated per row, should be cached or moved to a sync job.

Go
- Loops over channels, rows, or API calls where one failure aborts the whole batch (a 403, 5xx, or one bad row stops every later item). Ask whether the item should be skipped and logged, and whether a cosmetic step (avatars, thumbnails) can block the important write.
- A scheduled job with no overall deadline, so a slow upstream makes one run overlap the next.
- `http.Client` without a timeout; goroutines with no way to stop; missing `defer rows.Close()`.
- Reading an entire large body or file into memory when streaming works.

Frontend
- TanStack Query: stable query keys, `invalidateQueries` after mutations rather than manual refetch loops, no fetch inside `useEffect` that duplicates a query.
- Effects or handlers that trigger state updates in a loop; large lists rendered with no pagination or virtualization.
- Heavy libraries imported at the top level of a route that could be lazy-loaded; large images with no size or `loading="lazy"`.
