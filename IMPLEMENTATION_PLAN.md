# MSDL Resilience & CLI Implementation Plan

> **Status:** Proposed — 2026-06-17
> **Context:** Microsoft's Azure Sentinel WAF is blocking data-center IPs (Hetzner, Cloudflare Workers) on the `GetProductDownloadLinksBySku` endpoint at the ASN level. Two senior engineers independently concluded that server-side calls from hosting IPs are not sustainable, and that moving the Microsoft call onto the end-user's machine (the Rufus/Fido model) is the only durable fix.

---

## 1. Problem Statement

MSDL fetches signed Windows ISO download URLs from Microsoft's internal download API. That endpoint is now protected by Azure Sentinel WAF, which blocks by **ASN reputation** — data-center ranges are cleanly distinguishable from residential/commercial ISPs, so Microsoft can throttle hosting traffic with zero impact on real browser users.

**Observed escalation:**
- 2026-06-15: ~2 Sentinel rejections/day, intermittent
- 2026-06-16: ~15 rejections clustered in the evening, consistently hitting Win11 25H2 (products 3262, 3321)
- Windows 10 (2618) and eval builds still succeed; newer/high-traffic products are targeted first

**Current blast radius:** The caching layer (95% hit rate) absorbs most of it. The failure mode is: cache expires → backend refresh is WAF-blocked → stale-on-failure serves the old URL → eventually the URL passes its real `se` expiry → **user gets a dead Microsoft CDN link.**

**Confirmed not fixable server-side:** IP rotation buys weeks; CF Workers already blocked; residential proxies cost money and add a volatile dependency unjustifiable for a free tool. This is Microsoft policy, not a glitch.

---

## 2. Architecture Decision

**Invert the network footprint.** Today: `backend (Hetzner IP) → Microsoft → users`. The hard call originates from a blocked ASN.

New model: **`users (residential IP) → Microsoft → backend cache → all users`**

Three coordinated changes:

1. **CLI calls Microsoft directly** from the user's own machine. Their residential/commercial IP is one Microsoft cannot block without breaking downloads for real customers. This is exactly why Rufus/Fido work.
2. **Crowdsourced cache warming.** When a CLI user fetches a fresh link, the CLI optionally contributes that URL back to the MSDL backend. Microsoft's signed URLs are *not* user-specific (the `se` param is just an expiry timestamp — MSDL already serves one cached URL to many users), so a link fetched by one user is valid for everyone. The backend's Microsoft API calls trend toward zero; Sentinel becomes irrelevant.
3. **Web graceful degradation.** When the cache is dead and the backend is blocked, the web stops silently serving dead links — it flags expiry honestly and offers both the official Microsoft fallback and the web-to-CLI handoff.

**What stays unchanged:** The Go backend keeps serving the React frontend. Eval builds (separate Eval Center endpoint, unaffected) remain a first-class, reliable offering. The two-layer caching system is retained and becomes the distribution layer for crowdsourced links.

```
                    ┌─────────────────────────────────────────┐
                    │   User visits msdl.tech-latest.com        │
                    └───────────────────┬───────────────────────┘
                                        │
                          Cached link valid & fresh?
                       ──────────┬──────────────┬──────────
                          (Yes)  │              │  (No / expired / WAF-blocked)
                                 ▼              ▼
                     Serve signed link    Graceful degradation:
                                          1. Show "likely expired" flag
                                          2. Official Microsoft fallback (guided)
                                          3. Web-to-CLI handoff command
                                                       │
                                                       ▼
                                   User runs:  msdl --id 3262 --lang "..."
                                                       │
                              ┌────────────────────────┼────────────────────────┐
                              ▼                         ▼                         ▼
                   CLI fetches from MS        Prints URL to user        POST /contribute
                   (their residential IP)     (immediate result)        (warms shared cache)
                                                                                  │
                                                                                  ▼
                                                                   Next web visitor: cache hit
```

---

## 3. Phasing Overview

| Phase | Deliverable | Why this order |
|-------|-------------|----------------|
| **1** | Web graceful degradation (expired flag + official fallback) | Immediate — stops users hitting silent dead links today. Pure frontend + small backend signal. No dependency on the CLI. |
| **2** | CLI rework: direct-to-Microsoft | The core durable fix. Replaces the backend-routed CLI plan. |
| **3** | Crowdsourced cache contribution (`/contribute` + CLI flag) | Depends on Phase 2 CLI existing. Turns every CLI run into cache warming. |
| **4** | Web-to-CLI handoff UX | Depends on Phases 2–3. Wires the website's expired state to the CLI install/command. |
| **5** | Distribution & winget | Ships the CLI to users (covered in detail in the existing TDD plan). |

> The granular, TDD-level, step-by-step build instructions for the CLI binary itself (module scaffold, catalog, picker, flags, release workflow, winget manifests) already live in
> [`docs/superpowers/plans/2026-06-15-msdl-cli.md`](docs/superpowers/plans/2026-06-15-msdl-cli.md).
> **That plan must be amended for Phase 2** (it currently routes through `api.msdl.tech-latest.com`; it must instead port the Microsoft session flow into the binary). This document is the strategic master; that document is the implementation detail.

---

## Phase 1 — Web Graceful Degradation (Immediate)

**Goal:** No user is ever silently handed a dead Microsoft CDN link. When the link is at-risk or unavailable, say so and give a guaranteed path forward.

### 1.1 Backend: expose link freshness in the proxy response

**Files:** `backend/main.go` (`handleProxy`, ~line 817; `linkCacheEntry` line 105)

The frontend already parses `se` for the countdown, but it can't distinguish "fresh from Microsoft" from "stale-on-failure served past a failed refresh." Surface that explicitly.

- Add a response header on `/proxy` responses indicating provenance and confidence:
  - `X-MSDL-Link-Status: fresh` — fetched live from Microsoft this request
  - `X-MSDL-Link-Status: cached` — served from a valid (not-yet-soft-expired) cache entry
  - `X-MSDL-Link-Status: stale` — served via stale-on-failure after a WAF block; **may be expired**
- The `stale` branch is the existing `serving stale` path (the `fetch failed (...), serving stale` log line). Set the header there.
- Also emit `X-MSDL-Link-Expires: <RFC3339>` (the `se`-derived expiry) so the frontend has authoritative metadata for the countdown without re-parsing the URL itself. *(Engg-2 suggestion.)*
- Ensure `enableCORS` (line 708) exposes the headers: add `Access-Control-Expose-Headers: X-MSDL-Link-Status, X-MSDL-Link-Expires`.

### 1.2 Frontend: "likely expired" flag

**Files:** `frontend/src/pages/ProductDetailPage.tsx`

- Read `X-MSDL-Link-Status` from the `/proxy` fetch response.
- When `stale`, render an amber inline warning above the download button:
  > ⚠️ This link is cached and may have expired. Microsoft is currently limiting automated requests. Use the options below to get a guaranteed fresh download.
- When `fresh`/`cached`, keep the existing expiry countdown unchanged.

### 1.3 Frontend: official Microsoft fallback (guided)

**Files:** `frontend/src/pages/ProductDetailPage.tsx`, new `frontend/src/components/OfficialFallback.tsx`

- Below the download card, a collapsible "Get it directly from Microsoft" section that:
  - Links to the official page (`https://www.microsoft.com/software-download/windows11`, or the matching page per product family).
  - Lists the exact dropdown selections the user must make (edition + the language matching their chosen SKU).
- **Data-driven, not hardcoded** *(Engg-2 suggestion):* derive the official-page URL and edition label from the product catalog (`frontend/public/data/products.json`) keyed by product family, and use the SKU's own language string for the language step — so adding a product doesn't require touching the fallback component.
- This keeps MSDL useful as an **information directory** even when automation is fully blocked.

### 1.4 Acceptance

- Trigger a stale serve (point the backend at a WAF-blocked product or mock the stale path); confirm the amber warning + official fallback both render.
- Confirm `fresh`/`cached` responses show no warning and the countdown still works.
- `git commit -m "feat: graceful degradation for WAF-blocked links (expiry flag + official fallback)"`

---

## Phase 2 — CLI Direct-to-Microsoft (Core Durable Fix)

**Goal:** A standalone `msdl` binary that performs the **entire** Microsoft flow locally — session setup, SKU lookup, link fetch — so the request originates from the user's residential IP.

### 2.1 Amend the existing CLI plan

The detailed plan at [`docs/superpowers/plans/2026-06-15-msdl-cli.md`](docs/superpowers/plans/2026-06-15-msdl-cli.md) currently has `cli/api.go` calling `api.msdl.tech-latest.com`. **Replace that approach:** the CLI must own the Microsoft logic. Keep from that plan: module scaffold (Task 1), product catalog (Task 2), interactive picker (Task 4), main flow/flags (Task 5), release workflow + winget (Tasks 6–7). **Rewrite the API layer (Task 3).**

### 2.2 Port the Microsoft session flow into the CLI

**Source (backend):** these functions are the reference implementation to port —
- `setupSession() (string, *simpleCookieJar, error)` — `backend/main.go:394` (vlscppe → ov-df fingerprinting chain + cookie accumulation)
- `simpleCookieJar` — custom `http.CookieJar` with domain scoping disabled
- `fetchSkuInfoFromMS(productID string) ([]byte, int, error)` — `backend/main.go:470`
- `fetchDownloadLinksFromMS(productID, skuID string) ([]byte, error)` — `backend/main.go:566`

**Target (CLI):** create `cli/microsoft.go` with the ported equivalents. Strip server concerns (no shared session map — the CLI creates a fresh session per invocation, which is fine and more browser-like). Reuse the catalog and picker from the existing plan.

> **Shared package vs. copy** *(Engg-2 raised `pkg/microsoft/`):* the CLI is a separate Go module (`cli/go.mod`), so sharing a package across the backend and CLI modules requires a Go workspace or `replace` directives — added friction for ~250 lines of stable code. **Decision: copy/port for v1** (the logic changes rarely; divergence risk is low). Revisit a shared module only if the fingerprinting flow starts changing frequently and the two copies drift.

**New `cli/microsoft.go` responsibilities:**
- `newSession() (*http.Client, string)` — build the `simpleCookieJar`-backed client, run the fingerprinting chain, return the client + session ID.
- `fetchLanguages(client, sessionID, productID) ([]Language, error)` — GET SKU info, parse `Skus`.
- `fetchDownloadLinks(client, sessionID, productID, skuID) ([]DownloadLink, error)` — link fetch (with SKU-info warmup), parse `ProductDownloadOptions`.
- `fetchEvalLinks(evalURL) ([]EvalLink, error)` — scrape the Eval Center page directly and resolve fwlinks.

**Eval builds run directly too.** For consistency and future-proofing, the CLI scrapes the Microsoft Eval Center itself rather than proxying evals through the backend. The Eval Center isn't WAF-blocked today, but "today" is exactly the assumption that just broke for the consumer endpoint — keeping the CLI fully backend-independent means it never inherits a future Eval Center block.

### 2.3 TLS fingerprint hardening (note, not blocking)

Go's default `crypto/tls` ClientHello produces a JA3 fingerprint unlike any real browser. On user residential IPs this is a secondary signal (IP diversity is the primary protection), so ship with stdlib first. **If Sentinel starts fingerprinting the CLI,** swap `net/http`'s transport for [`utls`](https://github.com/refraction-networking/utls) with a Chrome ClientHello profile. Track as a follow-up, not a launch blocker.

### 2.4 Acceptance

- On a residential connection: `msdl --id 3262 --lang "English (United States)"` prints a valid signed URL with no backend involved.
- `msdl --id 3321` (a product the backend currently gets WAF-blocked on) succeeds from a home IP — this is the proof the architecture works.
- Eval builds: `msdl --eval server-2025` scrapes the Eval Center directly and prints the resolved `.iso` URLs — no backend involved.

---

## Phase 3 — Crowdsourced Cache Contribution

**Goal:** Every CLI link fetch optionally warms the shared backend cache, so web visitors get hits without the backend ever calling Microsoft.

### 3.0 Redis/Valkey cache persistence (prerequisite to 3.1)

**Why:** Without persistence, a backend restart wipes the link cache. If the backend is also WAF-blocked it cannot rebuild from Microsoft — crowdsourced contributions would be the only source of fresh links, so losing them on restart defeats Phase 3 entirely.

**Stack:** Redis/Valkey on Coolify (same Hetzner instance). TTL-native, Go has a first-class client (`github.com/redis/go-redis/v9`), zero added ops cost. Preferred over PostgreSQL (overkill for key-value TTL cache) and CF KV (requires routing through the Worker from Go, awkward).

**Architecture:** Two-layer cache — Redis as L2 persistent store; the existing in-memory `linkCache` map stays as L1 for hot reads.

**Files:** `backend/main.go`, `backend/go.mod`

- Add `REDIS_URL` env var (e.g. `redis://localhost:6379`). If unset, skip Redis silently — backend runs in memory-only mode for local dev.
- Key format: `msdl:link:<productID>:<skuID>` (namespaced).
- On startup: seed in-memory cache from Redis (so a restart recovers immediately).
- On link cache write (fresh fetch or contribution): `SET key value EX <ttl_seconds>` using `time.Until(entry.ExpiresAt)`.
- On in-memory cache miss in `handleProxy`: check Redis before calling Microsoft.
- If Redis is unavailable at runtime: log a warning and fall through to in-memory-only (keeps the existing failure behaviour, no hard dependency).

**Deploy order:** Provision a Valkey service on Coolify and set `REDIS_URL` before Phase 3 goes to production. Add `github.com/redis/go-redis/v9` to `backend/go.mod`.

---

### 3.1 Backend: `/contribute` endpoint

**Files:** `backend/main.go` (register in `main()` ~line 1147; cache write mirrors `handleProxy` line 922)

- `POST /contribute` accepting JSON: `{ "product_id": "3262", "sku_id": "19675", "raw_json": <the exact GetProductDownloadLinksBySku response bytes> }`.
- **Validation (reject untrusted contributions):**
  1. `product_id` must be in the known catalog (reuse the allow-list logic the proxy uses).
  2. Parse `raw_json` with the existing `parseLinkExpiry(rawJSON []byte) time.Time` (line 208). Reject if expiry is in the past or < 1 hour out.
  3. Extract the URL(s) from `ProductDownloadOptions`; reject unless the host matches the Microsoft CDN pattern (e.g. `*.download.prss.microsoft.com` / `software-static.download.prss.microsoft.com`).
  4. **No `HEAD` validation by default** *(both engineers):* parsing + host-allow-list + expiry are the real defense, and an extra request to Microsoft's CDN on every contribution adds traffic Microsoft may be sensitive to. Rely on parsing first; only add a `HEAD` check later if junk slips through.
- On success, write to the shared cache exactly as `handleProxy` does:
  ```go
  cacheKey := productID + ":" + skuID
  expiresAt := parseLinkExpiry(raw)
  linkCacheMu.Lock()
  linkCache[cacheKey] = linkCacheEntry{RawJSON: raw, ExpiresAt: expiresAt, FetchedAt: time.Now()}
  linkCacheMu.Unlock()
  ```
- Respond `204 No Content` on accept, `400`/`422` on validation failure.
- **Logging** *(Engg-2):* log accepts as `contribute: product_id=%s sku_id=%s -> accepted, cached until %s` AND rejections clearly as `contribute: product_id=%s sku_id=%s -> rejected (<reason>)` so contribution patterns/abuse are visible in ops logs from day one.
- **Abuse protection:** since this writes to a cache served to all users —
  - **Per-IP rate limit** on `/contribute`: ~5/min *(Engg-2's concrete number)*. Build it as small middleware (a per-IP token bucket in a `map[string]…` with mutex) — there's no existing rate-limit middleware to reuse.
  - **`CONTRIBUTE_SECRET`** env var, baked into released CLI builds, checked as a request header. Cheap obfuscation that blocks casual abuse; not real security.
  - The host allow-list + expiry validation remain the real defense: the worst a bad actor can do is submit a *valid Microsoft link*, which is harmless.

### 3.2 CLI: contribution flag

**Files:** `cli/main.go`, `cli/microsoft.go`

- After a successful link fetch, POST the raw response to `https://api.msdl.tech-latest.com/contribute` in the background. **Truly non-blocking** *(Engg-2):* fire it in a goroutine with a `context.WithTimeout(ctx, 5*time.Second)`, send the `CONTRIBUTE_SECRET` as a header, ignore all errors. The user already has their URL printed before this runs — and `main` must wait on it only briefly (or the process may exit first; spawn it before printing the result and `wg.Wait()` with the same short timeout, or simply accept best-effort).
- **Opt-out, transparent by default:** print to stderr `✓ Shared with msdl.tech cache to help other users (disable with --no-contribute)`. Add `--no-contribute` flag. Honor a `MSDL_NO_CONTRIBUTE=1` env var too.

### 3.3 Acceptance

- Run the CLI for a product whose backend cache is empty/expired; confirm `/contribute` logs an accept and a subsequent web request to `/proxy` for that product is a **cache hit** with `X-MSDL-Link-Status: cached`.
- Submit a junk/non-Microsoft URL to `/contribute`; confirm rejection.
- Submit a near-expired link; confirm rejection.

---

## Phase 4 — Web-to-CLI Handoff

**Goal:** When the web cache is dead and the backend is WAF-blocked, the site offers the CLI as the guaranteed path — and that CLI run heals the cache for the next visitor (Phase 3).

### 4.1 Frontend handoff UI

**Files:** `frontend/src/pages/ProductDetailPage.tsx`, `frontend/src/components/CliHandoff.tsx`

- In the `stale` state (from Phase 1.1), in addition to the official fallback, show:
  > **Get a fresh link instantly with the MSDL CLI** — runs from your computer, always works.
  - Install one-liner per platform (winget for Windows, curl|sh for Linux/macOS — final commands from the release in Phase 5).
  - The exact command pre-filled for this product: `msdl --id 3262 --lang "English (United States)"`
  - One-line explanation: "Runs the download lookup from your own connection, so Microsoft's rate limits don't apply. It also shares the fresh link back to help other visitors."
- Copy-to-clipboard button (reuse the existing CLI command tab component pattern).

### 4.2 Acceptance

- Force a stale state; confirm the handoff block shows the correct install command + pre-filled product command, and copy works.

---

## Phase 5 — Distribution

Covered in full by [`docs/superpowers/plans/2026-06-15-msdl-cli.md`](docs/superpowers/plans/2026-06-15-msdl-cli.md) Tasks 6–7:
- GitHub Actions release workflow (4-platform binaries, `cli/v*` tag trigger)
- Winget publishing (`starkSV.msdl`) with `wingetcreate` auto-PR on future releases
- `curl | sh` install script for Linux/macOS

No changes needed here beyond ensuring the `--no-contribute` flag and contribute endpoint URL are baked into release builds.

---

## 4. Cleanup / Catalog Note (do ASAP — independent of the above)

- **Product 48 (Windows 8.1 Single Language)** returns `ERROR [502]: no download links found for this SKU` on every attempt — this is a Microsoft-side removal, not a WAF block. **Both reviewers flagged this as do-ASAP.** Remove it from `frontend/public/data/products.json` now (and from `cli/catalog.go` when the CLI is built). This is a one-line config change → straight to main.

---

## 5. Risk & Open Questions

- **Contribution trust model:** the CDN-host allow-list + expiry validation make malicious contributions essentially harmless (worst case: a valid Microsoft link). Revisit if abuse appears.
- **Eval Center longevity:** the CLI scrapes Eval Center directly (decided). If Microsoft changes that page's markup, `fwlinkRe`/`isoLangRe` in `cli/microsoft.go` (and the backend) need updating — low risk, easy fix.
- **TLS fingerprinting (`utls`):** only if Sentinel begins fingerprinting the CLI on residential IPs. Not a launch blocker.
- **Backend's own Microsoft calls:** once crowdsourced contributions are flowing, consider whether the backend should still attempt live fetches at all, or become contribution-only + stale-serving. Likely keep live fetch as a fallback for cold products with no contributor yet.

---

## 6. Sequencing Recommendation

1. **Phase 1 now** — independent, immediate user benefit, low risk.
2. **Catalog cleanup (§4)** — trivial, do alongside Phase 1.
3. **Phase 2** — amend & execute the CLI TDD plan with the direct-to-Microsoft rewrite.
4. **Phase 3** — once the CLI fetches links, add contribution.
5. **Phase 4 + 5** — wire the web handoff and ship distribution together.
