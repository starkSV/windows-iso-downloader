# MSDL — Progress Log

## Shipped

### CLI + resilience architecture (formerly `IMPLEMENTATION_PLAN.md`, Phases 1–5) — merged

Microsoft's Azure Sentinel WAF started blocking the backend's data-center IP (Hetzner) on
the link endpoint. Rather than fight it server-side, inverted the network footprint: the
CLI now talks to Microsoft directly from the user's own residential IP, and feeds fresh
links back into the shared backend cache.

- **`msdl` CLI (`cli/`)** — standalone Go binary, ports the full Microsoft session flow
  (`cli/microsoft.go`: vlscppe permit → ov-df fingerprint → SKU lookup → signed link fetch)
  so requests originate from the user's machine, not a blockable ASN. Also scrapes the
  Eval Center directly for eval builds — no backend dependency either way.
- **Crowdsourced cache contribution** — after a successful CLI fetch, the raw Microsoft
  response is POSTed to the backend's `POST /contribute` (validated against a product
  allow-list, CDN-host allow-list, and expiry check; rate-limited per IP; gated by
  `CONTRIBUTE_SECRET`). Opt out with `--no-contribute` or `MSDL_NO_CONTRIBUTE=1`.
- **Redis/Valkey L2 cache** — `REDIS_URL` env var; in-memory cache is seeded from Redis on
  startup and written through on every fresh fetch, so a backend restart no longer loses
  crowdsourced links. Falls back to memory-only mode silently if unset/unreachable.
  Added `github.com/redis/go-redis/v9`.
- **Web graceful degradation** — `/proxy` responses carry `X-MSDL-Link-Status`
  (`fresh` / `cached` / `stale`) and `X-MSDL-Link-Expires` headers. A stale (WAF-blocked
  refresh) response now shows an explicit warning plus an `OfficialFallback` component
  (data-driven from `products.json`, links to Microsoft's own download page) instead of
  silently handing out a possibly-dead link.
- **CLI handoff UI** — `CliHandoff` component on the product page offers the install
  one-liner + pre-filled `msdl` command as a guaranteed-fresh alternative when the cached
  link is stale.
- **CLI telemetry** — `GET /cli/version` (update check) and `POST /telemetry` (anonymous
  action/platform/version counts, no personal data) endpoints; opt out with
  `--no-telemetry` / `MSDL_NO_TELEMETRY=1`. Counts surfaced in `/metrics`.
- **Catalog cleanup** — removed product `48` (Windows 8.1 Single Language), which
  Microsoft had already pulled server-side (`ERROR [502]: no download links found`).
- **Distribution** — GitHub Actions release workflow (4-platform binaries on `cli/vX.Y.Z`
  tags), winget manifest (`starkSV.msdl`), Homebrew tap (`msdl-cli` formula), AUR package
  (`msdl-bin`), and a `curl | bash` installer for Linux/macOS without Homebrew.

---

### eval ISOs — PR #13 `feat/evalcenter-isos`
**Windows Server 2016–2025 + Windows 11 Enterprise evaluation editions**

- `GET /evallinks?product=<slug>` endpoint — resolves Microsoft Eval Center fwlink redirects, caches for 24h, warms on startup
- Frontend eval section: `/eval` listing page + `/eval/:slug` detail page with download links
- Eval products: `server-2025`, `server-2022`, `server-2019`, `server-2016`, `win11-ent`
- ComparisonTable + FAQAccordion updated to mention eval ISOs
- HowItWorks step 1 updated
- SEO audit across all pages (meta descriptions, robots noindex on legal pages, JSON-LD on product detail)
- README: `/evallinks` API docs, eval product table, contributing guide for eval editions

---

### Two-layer caching — PR #14 + #15 `feat/caching-layer` (closes #12)
**Reduced Microsoft API calls from thousands/day to ~50–100/day**

- **Singleflight** (`golang.org/x/sync/singleflight`) — collapses concurrent cache misses into 1 Microsoft fetch
- **SKU cache** — 7-day TTL, keyed by `product_id`
- **Link cache** — dynamic TTL parsed from `se` param in signed Microsoft CDN URL, minus 30min buffer
- **Eval cache** — 24h TTL, keyed by slug, warmed at startup
- **Negative cache** — 60s TTL, prevents thundering-herd retries during rate-limit blocks
- **Stale-on-failure** — serves expired entry if refresh fails; background refresh via singleflight
- **Jitter** — ±5min random offset on TTLs to prevent synchronized mass-expiry
- **Cache eviction** — background goroutine every 30min, logs only when entries deleted
- **Cache hit/miss logging** — every request logs fetched vs cached + expiry timestamp
- Dockerfile bumped to `golang:1.25-alpine` to match `go.mod`

---

### Metrics endpoint — `feat/metrics`
**Real-time cache observability**

- `GET /metrics?secret=<secret>` — returns cache hit rates, miss counts, MS fetch totals, cache sizes
- Auth via `METRICS_SECRET` env var (`?secret=` param or `Authorization: Bearer` header)
- Atomic counters (`sync/atomic`) — lock-free, resets on restart
- Covers SKU, link, eval, and negative caches
- Background stale refresh routed through singleflight (prevents race condition under concurrent stale hits)

---

### README architecture docs — direct to `main`

- How It Works flow diagram updated with cache check step
- Caching layer section — table of all 4 caches, singleflight, stale-on-failure, jitter, eviction
- Backend Tech Stack table updated (Go 1.25+, real cache description)
- `/metrics` added to API Reference with example response
- `METRICS_SECRET` added to env vars block

---

## Production metrics snapshot (after ~1 day of traffic)

```json
{
  "sku":  { "requests": 37, "cache_hits": 29, "ms_fetches": 8,  "hit_rate": "78.4%" },
  "link": { "requests": 28, "cache_hits": 14, "ms_fetches": 14, "hit_rate": "50.0%" },
  "eval": { "requests": 2,  "cache_hits": 1,  "stale": 0,       "hit_rate": "50.0%" },
  "total_ms_fetches": 22
}
```
67 user requests → 22 Microsoft calls. **67% of calls eliminated** on a still-warming cache.
CF Worker ensures Hetzner IP is never exposed to Microsoft — rate-limit block (715-123130) is effectively impossible.

---

### Frontend UX improvements — `feat/ux-improvements` (v1.2.0)

- **Expiry countdown** — shows on consumer links as static "24h" (Microsoft doesn't include `se` in consumer CDN URLs); ticks live when `se` param present
- **Refresh links** — `?force=true` on `/proxy`, button appears under 6h remaining, fires fresh Microsoft fetch
- **CLI command tabs** — wget / curl / aria2, persists in localStorage, replaces old Aria2Tip on all pages
- **Recently viewed** — localStorage, homepage row, both consumer and eval pages tracked, shows expired state
- ~~**File size**~~ — not feasible, Microsoft CDN API does not return file size

---

## Deferred

| Item | Notes |
|---|---|
| Per-IP / per-product rate limiter | Revisit after 1 month of production traffic data |
| `_redirects` Cloudflare Pages bug | SPA fallback rule flagged as infinite loop, may cause 404s on direct nav |
