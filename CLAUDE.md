# MSDL — Claude Context

## Project

Open-source Windows ISO downloader. React + Go. Live at msdl.tech-latest.com.
Frontend on Cloudflare Pages, backend on Hetzner via Coolify.
Outbound Microsoft API calls route through a Cloudflare Worker (`cloudflare-worker/worker.js`)
controlled by `CF_WORKER_URL` + `CF_WORKER_SECRET` env vars on the backend.

## Conventions

- Small config/one-liner fixes → commit directly to main
- Any meaningful feature or multi-file change → feature branch → PR → merge
- GitHub issues are for bugs and community contributions only — planned features tracked here and in PROGRESS.md
- Commit messages follow conventional commits: `feat:`, `fix:`, `docs:`, `chore:`

## Backlog (implement when ready, in priority order)

### #12 — Two-layer in-memory caching ✅ DONE (feat/caching-layer, merged)

All 7 items shipped: singleflight, SKU cache (7d TTL), link cache (dynamic TTL from `se` param),
negative cache (60s), dynamic TTL, stale-on-failure, jitter. Verified in production.

### Beyond #12 — Observability & resilience

- [x] **Cache hit/miss logging** — logged on every request (fetched vs cached, cached until timestamp).
- [x] **`/metrics` endpoint** (auth-protected) — exposes cache hit rate, miss count, stale serves. Auth via `METRICS_SECRET` env var. (feat/metrics, merged)
- [x] **README architecture section** — caching layer, `/metrics`, env vars documented.
- [ ] **Per-IP / per-product rate limiter** — deferred; revisit after 1 month of production traffic data.

### Frontend UX improvements (next feature branch)

- [ ] **Expiry countdown** — parse `se` param from signed URL, show live countdown. Under 6h remaining → show "Refresh links" button.
- [ ] **Refresh links** — `?force=true` on `/proxy` bypasses cache, fetches fresh from Microsoft, resets countdown to ~24h.
- [ ] **CLI command tabs** — aria2 / wget / curl tabs instead of showing all at once. Persist last selected tab in localStorage. Default: wget.
- [ ] **Recently viewed** — localStorage only. Store product_id, name, version, se param. Show on homepage. Active links show countdown, expired links show "Expired" with re-fetch CTA.
- [ ] **File size** — verify if Microsoft CDN response includes file size. If yes, show alongside each download link (e.g. `x64 · 5.4 GB`). Research first before implementing.

### Known bugs / open issues

- [ ] **`_redirects` Cloudflare Pages** — SPA fallback rule (`/* /index.html 200`) is flagged
      as an infinite loop and ignored by Cloudflare Pages, potentially causing 404s on
      direct navigation to non-home routes. Needs investigation and fix.
