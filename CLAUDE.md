# MSDL — Claude Context

## Project

Open-source Windows ISO downloader. React + Go. Live at msdl.tech-latest.com.
Frontend on Cloudflare Pages, backend on Hetzner via Coolify.
Outbound Microsoft API calls route through a Cloudflare Worker (`cloudflare-worker/worker.js`)
controlled by `CF_WORKER_URL` + `CF_WORKER_SECRET` env vars on the backend.

## Conventions

- Small config/one-liner fixes → commit directly to main
- Any meaningful feature or multi-file change → feature branch → PR → merge
- Always reference the GitHub issue number in the commit message (Closes #X)
- Commit messages follow conventional commits: `feat:`, `fix:`, `docs:`, `chore:`

## Backlog (implement when ready, in priority order)

### #12 — Two-layer in-memory caching (backend/main.go only)

Goal: reduce Microsoft API traffic from thousands/day to ~50–100/day,
eliminating the 715-123130 rate-limit block under real traffic.

- [ ] **1. Singleflight** — `golang.org/x/sync/singleflight` wrapping all Microsoft fetches.
      Prevents cache stampede: 500 concurrent cache misses → 1 Microsoft hit.
- [ ] **2. SKU info cache** — 7-day TTL, keyed by `product_id`.
      Language lists are stable; no need to hit Microsoft on every page load.
- [ ] **3. Download link cache** — ~22h TTL, keyed by `product_id:sku_id`.
      Signed URLs are not IP-bound — safe to serve the same link to all users.
- [ ] **4. Negative response caching** — cache 429 / 715-123130 failures for 30–60s.
      Prevents thundering herd from retries worsening an existing block.
- [ ] **5. Dynamic TTL** — parse expiry from Microsoft's signed URL instead of fixed 22h.
      Derive as: `(link_expiry - now) - 30min`. Safer against clock skew.
      Store `cached_at` + `expires_at` metadata with each entry.
- [ ] **6. Stale-on-failure** — if refresh fails (rate-limited/transient), serve stale link
      temporarily and retry refresh in background. Prevents global outages.
- [ ] **7. Jitter** — add ±few minutes random offset to TTLs.
      Prevents synchronized mass-expiry spikes.

### Beyond #12 — Observability & resilience

- [ ] **Per-IP / per-product rate limiter** on the backend as extra protection against abuse.
- [ ] **Cache hit/miss logging** — extend existing request logs to show cache hits vs Microsoft calls.
- [ ] **`/metrics` endpoint** (auth-protected) — expose cache hit rate, miss count, stale serves.
- [ ] **README architecture section** — document the caching layer once implemented
      (CF Worker section already done, extend for caching).

### Known bugs / open issues

- [ ] **`_redirects` Cloudflare Pages** — SPA fallback rule (`/* /index.html 200`) is flagged
      as an infinite loop and ignored by Cloudflare Pages, potentially causing 404s on
      direct navigation to non-home routes. Needs investigation and fix.
