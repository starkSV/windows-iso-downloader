# CLI Telemetry, Update Check & Help — Design Spec

**Date:** 2026-06-21  
**Status:** Approved  
**Scope:** CLI (cli/) + Backend (backend/main.go)

---

## Overview

Three related additions to the msdl CLI and backend:

1. **Telemetry** — anonymous usage counters sent on every CLI run, stored durably in Redis
2. **Update check** — CLI checks our backend for the latest version on every run, notifies if behind
3. **Help** — `msdl --help` / `-h` prints usage with all flags and examples

No opt-in required. `MSDL_NO_TELEMETRY=1` skips both telemetry and update check silently.  
Disclosed in README and `/cli` page with one line each.

---

## Backend Changes

### 1. New constant: `latestCLIVersion`

```go
const latestCLIVersion = "0.3.0"
```

Bumped manually in `backend/main.go` whenever a new CLI release is cut.

### 2. `GET /cli/version` — public, no auth

Returns the latest CLI version. The CLI calls this on every run.

```json
{ "latest": "0.3.0" }
```

No rate limiting needed — response is trivial and cacheable by the client.

### 3. `POST /telemetry` — public, no auth

Accepts one event per CLI run. Body:

```json
{
  "action":     "fetch" | "eval" | "list" | "interactive",
  "product_id": "2618",
  "eval_slug":  "server-2025",
  "platform":   "windows" | "darwin" | "linux",
  "version":    "0.3.0",
  "success":    true
}
```

- `product_id` present only for `fetch` action
- `eval_slug` present only for `eval` action
- All other fields always present
- Returns `200 OK` with `{}` — CLI ignores the response
- Per-IP rate limit: ~10 req/min (same token bucket pattern as `/contribute`)
- Increments Redis counters via `HINCRBY` directly (no in-memory layer)
- If Redis unavailable: silently drop (telemetry is best-effort)

### 4. Redis telemetry keys

```
msdl:telemetry:actions    hash  { fetch, eval, list, interactive }
msdl:telemetry:platforms  hash  { windows, darwin, linux }
msdl:telemetry:versions   hash  { "0.2.0", "0.3.0", ... }
msdl:telemetry:products   hash  { "2618", "3262", "3113", ... }
msdl:telemetry:results    hash  { success, failed }
```

### 5. Existing metrics — Redis persistence

Current in-memory `atomic.Int64` counters are preserved for fast per-request increments. Added durability:

- **On startup**: seed in-memory counters from Redis (`HGETALL msdl:metrics:*`)
- **Every 5 minutes**: flush in-memory counters to Redis (`HSET msdl:metrics:*`)
- **On graceful shutdown**: final flush before exit

Redis keys for existing metrics:

```
msdl:metrics:sku   hash  { requests, hits, fetches, neg_hits }
msdl:metrics:link  hash  { requests, hits, fetches, neg_hits, stale }
msdl:metrics:eval  hash  { requests, hits, stale }
```

### 6. `/metrics` response — updated

Adds telemetry section to existing cache stats:

```json
{
  "sku":  { ... },
  "link": { ... },
  "eval": { ... },
  "telemetry": {
    "actions":   { "fetch": 142, "eval": 31, "list": 8, "interactive": 67 },
    "platforms": { "windows": 180, "darwin": 42, "linux": 18 },
    "versions":  { "0.3.0": 145, "0.2.0": 95 },
    "products":  { "3262": 134, "2618": 89, "3113": 67 },
    "results":   { "success": 390, "failed": 48 }
  }
}
```

---

## CLI Changes

### 1. Version constant

Injected at build time via ldflags:

```
-ldflags "-X main.Version=0.3.0"
```

Added to the existing GitHub Actions release workflow (`cli-release.yml`). Fallback: `const Version = "dev"` in source.

### 2. Update check + telemetry — concurrent goroutines

On every run, immediately after flag parsing, two goroutines are launched:

```
main()
  ├── go updateCheck()   — GET /cli/version, 500ms timeout, print notice if behind
  └── go sendTelemetry() — POST /telemetry, fire-and-forget, result ignored
```

Both are non-blocking. The update check result is printed before any other output if it arrives within 500ms. If the timeout fires, it's silently skipped for that run.

**Update notice format** (printed before picker or output):
```
  A new version of msdl is available: v0.3.0
  Download: https://github.com/starkSV/windows-iso-downloader/releases/latest
```

Both goroutines are skipped entirely when `MSDL_NO_TELEMETRY=1` is set.

### 3. Telemetry payload construction

| Scenario | action | product_id | eval_slug | success |
|---|---|---|---|---|
| `msdl` (interactive) | `interactive` | set after pick | — | true/false |
| `msdl --id X --lang Y` | `fetch` | X | — | true/false |
| `msdl --eval slug` | `eval` | — | slug | true/false |
| `msdl --list` | `list` | — | — | true |

`platform` is set from `runtime.GOOS` at runtime (not build time — same result, simpler).  
`version` is set from the `Version` constant.

Telemetry is sent **after** the main action completes so `success` reflects the actual outcome.

### 4. `--help` / `-h`

Prints a formatted usage block and exits 0. Shown automatically by Go's `flag` package or via a custom print if the existing CLI uses manual arg parsing.

```
msdl — Windows ISO downloader

Usage:
  msdl                          Interactive mode — pick product and language
  msdl --id <id> --lang <lang>  Fetch link directly (skip picker)
  msdl --eval <slug>            Evaluation / Server ISO
  msdl --list                   List all available products

Flags:
  --id <id>           Product ID (e.g. 3262 for Windows 11 25H2)
  --lang <language>   Language name (e.g. "English")
  --eval <slug>       Eval ISO slug (server-2025, server-2022, win11-ent, ...)
  --list              List all products and exit
  --no-contribute     Skip contributing link back to msdl web cache
  -h, --help          Show this help

Environment:
  MSDL_NO_TELEMETRY=1   Disable anonymous usage reporting and update checks
  MSDL_NO_CONTRIBUTE=1  Disable cache contribution

More info: https://msdl.tech-latest.com/cli
```

---

## Documentation Updates

- **README** — one line under CLI section: "msdl sends anonymous usage counts (action, platform, version) to help us understand which products are popular. Set `MSDL_NO_TELEMETRY=1` to opt out."
- **`/cli` page** — same one-liner in the contribute section, next to the `--no-contribute` note.

---

## What's Not In Scope

- No per-user tracking, no IP storage, no session IDs
- No telemetry dashboard UI (raw numbers via `/metrics` is sufficient)
- No forced update or auto-update — notification only
- No telemetry for the web frontend
