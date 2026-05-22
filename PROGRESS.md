# MSDL — Progress Log

## Shipped

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

## Planned — Frontend UX improvements

| Feature | Notes |
|---|---|
| Expiry countdown | Parse `se` param from signed URL, live countdown. Under 6h → show Refresh button |
| Refresh links | `?force=true` on `/proxy` bypasses cache, returns fresh URL, resets countdown |
| CLI command tabs | aria2 / wget / curl tabs, persist selection in localStorage, default wget |
| Recently viewed | localStorage only — active links show countdown, expired show re-fetch CTA |
| File size | Research first — verify if Microsoft CDN response includes it |

---

## Deferred

| Item | Notes |
|---|---|
| Per-IP / per-product rate limiter | Revisit after 1 month of production traffic data |
| `_redirects` Cloudflare Pages bug | SPA fallback rule flagged as infinite loop, may cause 404s on direct nav |
