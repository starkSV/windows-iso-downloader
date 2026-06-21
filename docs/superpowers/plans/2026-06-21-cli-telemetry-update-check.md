# CLI Telemetry + Update Check Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add anonymous CLI usage telemetry, an update check on every run, and Redis durability for existing in-memory metrics — all on the `feat/cli-telemetry` branch.

**Architecture:** Three additions: (1) two new backend endpoints (`GET /cli/version`, `POST /telemetry`) that write counters to Redis; (2) Redis persistence for the existing in-memory metrics (seed on startup, flush every 5 min, final flush on graceful shutdown); (3) CLI goroutines that call those endpoints on every run. All CLI side-channel work uses `MSDL_NO_TELEMETRY=1` as a single opt-out.

**Tech Stack:** Go 1.21+, `backend/main.go` (monolith, no framework), `github.com/redis/go-redis/v9`, CLI uses `runtime.GOOS` for platform, `flag` package already in place.

## Global Constraints

- REDIS_URL must NEVER be committed to any file — it is injected as an env var in Coolify only.
- All changes go to feature branch `feat/cli-telemetry`. Do NOT push until winget PR #390908 is approved.
- Backend: `latestCLIVersion = "0.3.0"` is a constant in `backend/main.go` — bumped manually per release.
- Telemetry is best-effort throughout: if Redis is nil or HTTP fails, silently skip — never return an error to the user.
- Per-IP rate limit on `/telemetry`: 10 req/min, burst 10 (reuse existing `newIPRateLimiter` pattern).
- `MSDL_NO_TELEMETRY=1` env var skips both update check and telemetry in the CLI.
- CLI `Version` var defaults to `"dev"` in source; injected as `0.3.0` via `-X main.Version=0.3.0` in CI.
- No unit tests for backend handlers (existing codebase has none — test via curl). CLI: add tests only where deterministic pure functions are introduced.

---

### Task 1: Backend — `GET /cli/version` endpoint

**Files:**
- Modify: `backend/main.go`

**Interfaces:**
- Produces: `GET /cli/version` → `{"latest": "0.3.0"}` (200 OK, no auth)

- [ ] **Step 1: Create feature branch**

```bash
git checkout -b feat/cli-telemetry
```

- [ ] **Step 2: Add `latestCLIVersion` constant**

Find the existing `const (` block at the top of `backend/main.go` (lines ~32-39). Add `latestCLIVersion` as the last entry before the closing `)`:

```go
const (
    UA          = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
    PROFILE     = "606624d44113"
    LOCALE      = "en-US"
    ORG_ID      = "y6jn8c31"
    CUSTOMER_ID = "560dc9f3-1aa5-4a2f-b63c-9e18f8d0e175"
    PORT        = ":3002"

    latestCLIVersion = "0.3.0"
)
```

- [ ] **Step 3: Add `handleCLIVersion` function**

Add this function just before `func handleMetrics`:

```go
func handleCLIVersion(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{"latest": latestCLIVersion})
}
```

- [ ] **Step 4: Register route in `main()`**

In the `main()` function, inside the `mux.HandleFunc(...)` block, add after the `/health` route:

```go
mux.HandleFunc("/cli/version", handleCLIVersion)
```

- [ ] **Step 5: Build to verify compilation**

```bash
cd backend && go build -o /dev/null . && echo "OK"
```

Expected: `OK` with no errors.

- [ ] **Step 6: Smoke-test with the running backend**

```bash
curl -s http://localhost:3002/cli/version
```

Expected: `{"latest":"0.3.0"}`

- [ ] **Step 7: Commit**

```bash
git add backend/main.go
git commit -m "feat(backend): GET /cli/version returns latest CLI version"
```

---

### Task 2: Backend — `POST /telemetry` endpoint

**Files:**
- Modify: `backend/main.go`

**Interfaces:**
- Consumes: JSON body `{ action, product_id?, eval_slug?, platform, version, success }`
- Produces: `POST /telemetry` → `{}` (200 OK, no auth); 429 on rate limit; 400 on bad JSON/action

Redis keys written:
```
msdl:telemetry:actions    hash  { fetch, eval, list, interactive }
msdl:telemetry:platforms  hash  { windows, darwin, linux }
msdl:telemetry:versions   hash  { "0.2.0", "0.3.0", ... }
msdl:telemetry:products   hash  { "2618", "3262", ... }
msdl:telemetry:results    hash  { success, failed }
```

- [ ] **Step 1: Add `TelemetryPayload` struct and rate limiter**

Add just after the `var contributeRL` line (~line 472 in current file):

```go
var telemetryRL = newIPRateLimiter(10, 10) // 10 requests/min, burst 10

type telemetryPayload struct {
    Action    string `json:"action"`
    ProductID string `json:"product_id"`
    EvalSlug  string `json:"eval_slug"`
    Platform  string `json:"platform"`
    Version   string `json:"version"`
    Success   bool   `json:"success"`
}
```

- [ ] **Step 2: Add `handleTelemetry` function**

Add this function just before `handleCLIVersion`:

```go
func handleTelemetry(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }
    w.Header().Set("Content-Type", "application/json")

    ip := r.RemoteAddr
    if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
        ip = strings.TrimSpace(strings.SplitN(fwd, ",", 2)[0])
    }
    if !telemetryRL.allow(ip) {
        respondJSONError(w, http.StatusTooManyRequests, "rate limit exceeded")
        return
    }

    var p telemetryPayload
    if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
        respondJSONError(w, http.StatusBadRequest, "invalid JSON")
        return
    }

    validActions := map[string]bool{"fetch": true, "eval": true, "list": true, "interactive": true}
    if !validActions[p.Action] {
        respondJSONError(w, http.StatusBadRequest, "invalid action")
        return
    }

    if rdb != nil {
        ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
        defer cancel()
        rdb.HIncrBy(ctx, "msdl:telemetry:actions", p.Action, 1)
        if p.Platform != "" {
            rdb.HIncrBy(ctx, "msdl:telemetry:platforms", p.Platform, 1)
        }
        if p.Version != "" {
            rdb.HIncrBy(ctx, "msdl:telemetry:versions", p.Version, 1)
        }
        if p.ProductID != "" {
            rdb.HIncrBy(ctx, "msdl:telemetry:products", p.ProductID, 1)
        }
        result := "success"
        if !p.Success {
            result = "failed"
        }
        rdb.HIncrBy(ctx, "msdl:telemetry:results", result, 1)
    }

    w.WriteHeader(http.StatusOK)
    w.Write([]byte("{}"))
}
```

- [ ] **Step 3: Register route in `main()`**

```go
mux.HandleFunc("/telemetry", handleTelemetry)
```

- [ ] **Step 4: Build to verify compilation**

```bash
cd backend && go build -o /dev/null . && echo "OK"
```

- [ ] **Step 5: Smoke-test `/telemetry`**

```bash
curl -s -X POST http://localhost:3002/telemetry \
  -H "Content-Type: application/json" \
  -d '{"action":"fetch","product_id":"3262","platform":"windows","version":"0.3.0","success":true}'
```

Expected: `{}`

```bash
curl -s -X POST http://localhost:3002/telemetry \
  -H "Content-Type: application/json" \
  -d '{"action":"bad_action","platform":"linux","version":"0.3.0","success":false}'
```

Expected: `{"error":"invalid action"}` (400)

- [ ] **Step 6: Commit**

```bash
git add backend/main.go
git commit -m "feat(backend): POST /telemetry endpoint with Redis HINCRBY counters"
```

---

### Task 3: Backend — Redis metrics persistence + graceful shutdown

**Files:**
- Modify: `backend/main.go`

**Interfaces:**
- Consumes: existing `atomic.Int64` vars (`mSkuRequests`, `mLinkRequests`, etc.)
- Produces: Redis hashes `msdl:metrics:sku`, `msdl:metrics:link`, `msdl:metrics:eval`
- Produces: graceful shutdown that calls `flushMetricsToRedis()` before exit

Redis hash schemas:
```
msdl:metrics:sku  → { requests, cache_hits, ms_fetches, neg_hits }
msdl:metrics:link → { requests, cache_hits, ms_fetches, neg_hits, stale }
msdl:metrics:eval → { requests, cache_hits, stale }
```

- [ ] **Step 1: Add `"os/signal"` and `"syscall"` to imports**

The existing import block in `backend/main.go` starts on line 3. Add the two new packages:

```go
import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "html"
    "io"
    "log"
    "math/rand"
    "net/http"
    "net/url"
    "os"
    "os/signal"
    "regexp"
    "sort"
    "strconv"
    "strings"
    "sync"
    "sync/atomic"
    "syscall"
    "time"

    "github.com/google/uuid"
    "github.com/redis/go-redis/v9"
    "golang.org/x/sync/singleflight"
)
```

- [ ] **Step 2: Add `seedMetricsFromRedis` and `flushMetricsToRedis` functions**

Add both functions just after the `redisSeedLinkCache` function (~line 370):

```go
// seedMetricsFromRedis loads persisted metric counters from Redis into the
// in-memory atomic vars. Called once on startup after Redis connects.
func seedMetricsFromRedis() {
    if rdb == nil {
        return
    }
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    loadHash := func(key string) map[string]string {
        vals, err := rdb.HGetAll(ctx, key).Result()
        if err != nil {
            return nil
        }
        return vals
    }
    parseI64 := func(m map[string]string, field string) int64 {
        if m == nil {
            return 0
        }
        v, _ := strconv.ParseInt(m[field], 10, 64)
        return v
    }

    sku := loadHash("msdl:metrics:sku")
    atomic.StoreInt64(&mSkuRequests, parseI64(sku, "requests"))
    atomic.StoreInt64(&mSkuCacheHits, parseI64(sku, "cache_hits"))
    atomic.StoreInt64(&mSkuFetches, parseI64(sku, "ms_fetches"))
    atomic.StoreInt64(&mSkuNegHits, parseI64(sku, "neg_hits"))

    link := loadHash("msdl:metrics:link")
    atomic.StoreInt64(&mLinkRequests, parseI64(link, "requests"))
    atomic.StoreInt64(&mLinkCacheHits, parseI64(link, "cache_hits"))
    atomic.StoreInt64(&mLinkFetches, parseI64(link, "ms_fetches"))
    atomic.StoreInt64(&mLinkNegHits, parseI64(link, "neg_hits"))
    atomic.StoreInt64(&mLinkStale, parseI64(link, "stale"))

    eval := loadHash("msdl:metrics:eval")
    atomic.StoreInt64(&mEvalRequests, parseI64(eval, "requests"))
    atomic.StoreInt64(&mEvalCacheHits, parseI64(eval, "cache_hits"))
    atomic.StoreInt64(&mEvalStale, parseI64(eval, "stale"))

    log.Println("redis: seeded in-memory metrics from Redis")
}

// flushMetricsToRedis writes current in-memory metric counters to Redis.
// Called periodically (every 5 min) and on graceful shutdown.
func flushMetricsToRedis() {
    if rdb == nil {
        return
    }
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    rdb.HSet(ctx, "msdl:metrics:sku",
        "requests", atomic.LoadInt64(&mSkuRequests),
        "cache_hits", atomic.LoadInt64(&mSkuCacheHits),
        "ms_fetches", atomic.LoadInt64(&mSkuFetches),
        "neg_hits", atomic.LoadInt64(&mSkuNegHits),
    )
    rdb.HSet(ctx, "msdl:metrics:link",
        "requests", atomic.LoadInt64(&mLinkRequests),
        "cache_hits", atomic.LoadInt64(&mLinkCacheHits),
        "ms_fetches", atomic.LoadInt64(&mLinkFetches),
        "neg_hits", atomic.LoadInt64(&mLinkNegHits),
        "stale", atomic.LoadInt64(&mLinkStale),
    )
    rdb.HSet(ctx, "msdl:metrics:eval",
        "requests", atomic.LoadInt64(&mEvalRequests),
        "cache_hits", atomic.LoadInt64(&mEvalCacheHits),
        "stale", atomic.LoadInt64(&mEvalStale),
    )
    log.Println("redis: flushed in-memory metrics to Redis")
}
```

- [ ] **Step 3: Add `startMetricsFlusher` goroutine function**

Add after `flushMetricsToRedis`:

```go
// startMetricsFlusher flushes in-memory metrics to Redis every 5 minutes.
func startMetricsFlusher() {
    ticker := time.NewTicker(5 * time.Minute)
    for range ticker.C {
        flushMetricsToRedis()
    }
}
```

- [ ] **Step 4: Call `seedMetricsFromRedis` in `initRedis`**

Find the `initRedis` function. After `rdb = client` and `log.Println("redis: connected")`, add:

```go
    rdb = client
    log.Println("redis: connected")
    seedMetricsFromRedis()
```

- [ ] **Step 5: Rewrite `main()` to use `http.Server` with graceful shutdown**

Replace the existing `main()` function body:

```go
func main() {
    initRedis()
    go redisSeedLinkCache()
    go cleanupSessions()
    go cleanupCaches()
    go warmEvalCache()
    go startMetricsFlusher()

    mux := http.NewServeMux()
    mux.HandleFunc("/skuinfo", handleSkuInfo)
    mux.HandleFunc("/proxy", handleProxy)
    mux.HandleFunc("/contribute", handleContribute)
    mux.HandleFunc("/evallinks", handleEvalLinks)
    mux.HandleFunc("/metrics", handleMetrics)
    mux.HandleFunc("/telemetry", handleTelemetry)
    mux.HandleFunc("/cli/version", handleCLIVersion)
    mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(`{"status":"ok"}`))
    })
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path != "/" {
            http.NotFound(w, r)
            return
        }
        w.Write([]byte("MSDL API v3 is running"))
    })

    handler := enableCORS(mux)
    srv := &http.Server{Addr: PORT, Handler: handler}

    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
    go func() {
        <-sigCh
        log.Println("shutdown: flushing metrics to Redis...")
        flushMetricsToRedis()
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()
        srv.Shutdown(ctx)
    }()

    log.Printf("Go Backend running on http://localhost%s\n", PORT)
    if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
        log.Fatal(err)
    }
}
```

- [ ] **Step 6: Build to verify compilation**

```bash
cd backend && go build -o /dev/null . && echo "OK"
```

- [ ] **Step 7: Verify Redis flush manually**

Start the backend locally with `REDIS_URL` set, run a few requests, then check Redis:

```bash
redis-cli -u "$REDIS_URL" HGETALL msdl:metrics:sku
```

(Values will be 0 on a fresh Redis; after a request they increment.)

- [ ] **Step 8: Commit**

```bash
git add backend/main.go
git commit -m "feat(backend): Redis persistence for metrics (seed + 5min flush + graceful shutdown)"
```

---

### Task 4: Backend — Telemetry section in `/metrics` response

**Files:**
- Modify: `backend/main.go`

**Interfaces:**
- Consumes: `msdl:telemetry:*` Redis hashes (written by `handleTelemetry`)
- Produces: adds `"telemetry"` key to existing `handleMetrics` JSON response

- [ ] **Step 1: Add `loadTelemetryFromRedis` helper**

Add just before `handleMetrics`:

```go
// loadTelemetryFromRedis reads all telemetry counters from Redis.
// Returns nil maps when Redis is unavailable.
func loadTelemetryFromRedis() map[string]map[string]string {
    if rdb == nil {
        return nil
    }
    ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
    defer cancel()
    keys := []string{
        "msdl:telemetry:actions",
        "msdl:telemetry:platforms",
        "msdl:telemetry:versions",
        "msdl:telemetry:products",
        "msdl:telemetry:results",
    }
    out := make(map[string]map[string]string, len(keys))
    for _, k := range keys {
        vals, err := rdb.HGetAll(ctx, k).Result()
        if err == nil {
            // Strip the "msdl:telemetry:" prefix for the JSON key
            name := strings.TrimPrefix(k, "msdl:telemetry:")
            out[name] = vals
        }
    }
    return out
}
```

- [ ] **Step 2: Update `handleMetrics` to include telemetry**

Find the `json.NewEncoder(w).Encode(map[string]interface{}{` call in `handleMetrics`. Replace the map literal to add a `"telemetry"` field:

```go
    telemetry := loadTelemetryFromRedis()

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "sku": map[string]interface{}{
            "requests":   skuReqs,
            "cache_hits": skuHits,
            "ms_fetches": skuFetches,
            "neg_hits":   skuNeg,
            "hit_rate":   hitRate(skuHits, skuReqs),
            "cache_size": skuSize,
        },
        "link": map[string]interface{}{
            "requests":   linkReqs,
            "cache_hits": linkHits,
            "ms_fetches": linkFetches,
            "neg_hits":   linkNeg,
            "stale":      linkStale,
            "hit_rate":   hitRate(linkHits, linkReqs),
            "cache_size": linkSize,
        },
        "eval": map[string]interface{}{
            "requests":   evalReqs,
            "cache_hits": evalHits,
            "stale":      evalStale,
            "hit_rate":   hitRate(evalHits, evalReqs),
            "cache_size": evalSize,
        },
        "neg_cache_size":   negSize,
        "total_ms_fetches": skuFetches + linkFetches,
        "telemetry":        telemetry,
    })
```

- [ ] **Step 3: Build to verify compilation**

```bash
cd backend && go build -o /dev/null . && echo "OK"
```

- [ ] **Step 4: Smoke-test `/metrics`**

```bash
curl -s "http://localhost:3002/metrics?secret=$METRICS_SECRET" | python -m json.tool
```

Expected: existing fields still present plus `"telemetry": { "actions": {}, ... }` (empty until telemetry events arrive).

- [ ] **Step 5: Commit**

```bash
git add backend/main.go
git commit -m "feat(backend): include telemetry counters in /metrics response"
```

---

### Task 5: CLI — `Version` var + GitHub Actions ldflags + `--help` update

**Files:**
- Modify: `cli/main.go`
- Modify: `.github/workflows/cli-release.yml`

**Interfaces:**
- Produces: `Version` package-level variable accessible from all CLI functions
- Produces: CI builds inject `-X main.Version=<tag>` so the binary reports its real version

- [ ] **Step 1: Add `Version` variable to `cli/main.go`**

At the top of `cli/main.go`, after the `import` block, add:

```go
// Version is injected at build time via -ldflags "-X main.Version=0.3.0".
// Falls back to "dev" for local builds.
var Version = "dev"
```

- [ ] **Step 2: Update `fs.Usage` in `run()` to include env vars section**

Replace the existing `fs.Usage` function in `run()`:

```go
fs.Usage = func() {
    fmt.Fprintln(os.Stderr, `msdl — Windows ISO downloader

Usage:
  msdl [search terms]                           interactive: filter + pick product, pick language
  msdl --id 3262                                skip product picker
  msdl --id 3262 --lang "English (United States)"  no prompts, print URL directly
  msdl --eval [slug]                            evaluation ISOs (server-2025, win11-ent, ...)
  msdl --list                                   list all products and exit

Flags:`)
    fs.PrintDefaults()
    fmt.Fprintln(os.Stderr, `
Environment:
  MSDL_NO_TELEMETRY=1    Disable anonymous usage reporting and update checks
  MSDL_NO_CONTRIBUTE=1   Disable cache contribution
  MSDL_API_URL=<url>     Override backend URL (default: https://api.msdl.tech-latest.com)

More info: https://msdl.tech-latest.com/cli`)
}
```

- [ ] **Step 3: Handle `flag.ErrHelp` cleanly in `run()`**

Find `if err := fs.Parse(args); err != nil {` and update it:

```go
if err := fs.Parse(args); err != nil {
    if err == flag.ErrHelp {
        return nil // fs.Usage already printed
    }
    return err
}
```

- [ ] **Step 4: Update GitHub Actions workflow to inject version ldflags**

In `.github/workflows/cli-release.yml`, find the `Build binaries` step. Replace the `go build` commands to inject version:

```yaml
      - name: Build binaries
        working-directory: cli
        run: |
          VERSION="${{ github.ref_name }}"
          VERSION="${VERSION#cli/v}"
          mkdir -p ../dist
          GOOS=linux   GOARCH=amd64 go build -ldflags="-s -w -X main.Version=${VERSION}" -o ../dist/msdl-linux-amd64   .
          GOOS=darwin  GOARCH=amd64 go build -ldflags="-s -w -X main.Version=${VERSION}" -o ../dist/msdl-darwin-amd64  .
          GOOS=darwin  GOARCH=arm64 go build -ldflags="-s -w -X main.Version=${VERSION}" -o ../dist/msdl-darwin-arm64  .
          GOOS=windows GOARCH=amd64 go build -ldflags="-s -w -X main.Version=${VERSION}" -o ../dist/msdl-windows-amd64.exe .
```

- [ ] **Step 5: Build CLI locally to verify**

```bash
cd cli && go build -o /dev/null . && echo "OK"
```

- [ ] **Step 6: Verify --help flag works**

```bash
cd cli && go run . --help
```

Expected: prints usage block + env var section, exits 0 (no "error:" prefix).

- [ ] **Step 7: Commit**

```bash
git add cli/main.go .github/workflows/cli-release.yml
git commit -m "feat(cli): Version var injected via ldflags, updated --help text"
```

---

### Task 6: CLI — Update check + telemetry goroutines

**Files:**
- Modify: `cli/main.go`

**Interfaces:**
- Consumes: `Version` (from Task 5), `apiBaseURL()` helper
- Consumes: `GET /cli/version` backend endpoint (from Task 1)
- Consumes: `POST /telemetry` backend endpoint (from Task 2)
- Produces: update notice printed to stderr before main output when a newer version is available
- Produces: telemetry payload sent fire-and-forget after every CLI run

- [ ] **Step 1: Add `runtime` to imports in `cli/main.go`**

The existing import block has `"sync"` and other packages. Add `"runtime"`:

```go
import (
    "bytes"
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "net/http"
    "os"
    "runtime"
    "strings"
    "sync"
    "time"
)
```

- [ ] **Step 2: Add `apiBaseURL` helper**

Add after the `contributeSecret` const near the top of the file (before `urlFilename`):

```go
// apiBaseURL returns the backend base URL, with MSDL_API_URL override.
func apiBaseURL() string {
    if u := os.Getenv("MSDL_API_URL"); u != "" {
        return strings.TrimRight(u, "/")
    }
    return "https://api.msdl.tech-latest.com"
}
```

Note: `contributeURL()` already does similar logic but for `/contribute` specifically. `apiBaseURL` is the shared base — update `contributeURL` to use it:

```go
func contributeURL() string {
    return apiBaseURL() + "/contribute"
}
```

- [ ] **Step 3: Add `printUpdateNotice` function**

Add after `apiBaseURL`:

```go
// printUpdateNotice checks /cli/version and prints a notice if a newer version
// is available. Blocks up to 500ms; silently no-ops on timeout or any error.
func printUpdateNotice() {
    ch := make(chan string, 1)
    go func() {
        ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
        defer cancel()
        req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiBaseURL()+"/cli/version", nil)
        if err != nil {
            return
        }
        resp, err := http.DefaultClient.Do(req)
        if err != nil {
            return
        }
        defer resp.Body.Close()
        var body struct {
            Latest string `json:"latest"`
        }
        if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
            return
        }
        ch <- body.Latest
    }()
    select {
    case latest := <-ch:
        if latest != "" && latest != Version && Version != "dev" {
            fmt.Fprintf(os.Stderr, "\n  A new version of msdl is available: v%s\n", latest)
            fmt.Fprintf(os.Stderr, "  Download: https://github.com/starkSV/windows-iso-downloader/releases/latest\n\n")
        }
    case <-time.After(500 * time.Millisecond):
    }
}
```

- [ ] **Step 4: Add `cliTelemetryPayload` struct and `sendTelemetry` function**

Add after `printUpdateNotice`:

```go
type cliTelemetryPayload struct {
    Action    string `json:"action"`
    ProductID string `json:"product_id,omitempty"`
    EvalSlug  string `json:"eval_slug,omitempty"`
    Platform  string `json:"platform"`
    Version   string `json:"version"`
    Success   bool   `json:"success"`
}

// sendTelemetry posts a single telemetry event. Fire-and-forget: all errors
// are silently ignored so telemetry never affects the user experience.
func sendTelemetry(p cliTelemetryPayload) {
    body, err := json.Marshal(p)
    if err != nil {
        return
    }
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()
    req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiBaseURL()+"/telemetry", bytes.NewReader(body))
    if err != nil {
        return
    }
    req.Header.Set("Content-Type", "application/json")
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return
    }
    resp.Body.Close()
}
```

- [ ] **Step 5: Thread update check and telemetry into `run()`**

Replace the entire `run()` function with the complete version below. This incorporates the Task 5 changes (Usage block, ErrHelp handling) plus the new telemetry logic:

```go
func run(args []string) error {
    fs := flag.NewFlagSet("msdl", flag.ContinueOnError)
    fs.SetOutput(os.Stderr)
    fs.Usage = func() {
        fmt.Fprintln(os.Stderr, `msdl — Windows ISO downloader

Usage:
  msdl [search terms]                           interactive: filter + pick product, pick language
  msdl --id 3262                                skip product picker
  msdl --id 3262 --lang "English (United States)"  no prompts, print URL directly
  msdl --eval [slug]                            evaluation ISOs (server-2025, win11-ent, ...)
  msdl --list                                   list all products and exit

Flags:`)
        fs.PrintDefaults()
        fmt.Fprintln(os.Stderr, `
Environment:
  MSDL_NO_TELEMETRY=1    Disable anonymous usage reporting and update checks
  MSDL_NO_CONTRIBUTE=1   Disable cache contribution
  MSDL_API_URL=<url>     Override backend URL (default: https://api.msdl.tech-latest.com)

More info: https://msdl.tech-latest.com/cli`)
    }

    productID := fs.String("id", "", "consumer product ID (skips product picker)")
    langFlag := fs.String("lang", "", `language name, e.g. "English (United States)"`)
    evalMode := fs.Bool("eval", false, "fetch evaluation ISOs")
    listMode := fs.Bool("list", false, "list all products and exit")
    noContributeFlag := fs.Bool("no-contribute", false, "skip sharing the link with the msdl.tech cache")

    if err := fs.Parse(args); err != nil {
        if err == flag.ErrHelp {
            return nil
        }
        return err
    }
    query := strings.Join(fs.Args(), " ")
    noContribute := *noContributeFlag || os.Getenv("MSDL_NO_CONTRIBUTE") == "1"
    noTelemetry := os.Getenv("MSDL_NO_TELEMETRY") == "1"

    // Update check — blocks up to 500ms, then continues regardless
    if !noTelemetry {
        printUpdateNotice()
    }

    if *listMode {
        fmt.Fprintln(os.Stderr, "Consumer products:")
        for _, p := range consumerProducts {
            fmt.Fprintf(os.Stderr, "  %-6s %s\n", p.ID, p.Name)
        }
        fmt.Fprintln(os.Stderr, "\nEvaluation products:")
        for _, p := range evalProducts {
            fmt.Fprintf(os.Stderr, "  %-20s %s\n", p.Slug, p.Name)
        }
        if !noTelemetry {
            go sendTelemetry(cliTelemetryPayload{
                Action:   "list",
                Platform: runtime.GOOS,
                Version:  Version,
                Success:  true,
            })
        }
        return nil
    }

    // Determine telemetry fields before running (product_id / eval_slug known at this point)
    telAction := "interactive"
    telProductID := ""
    telEvalSlug := ""
    switch {
    case *evalMode:
        telAction = "eval"
        telEvalSlug = query
    case *productID != "":
        telAction = "fetch"
        telProductID = *productID
    }

    var err error
    if *evalMode {
        err = runEval(query)
    } else {
        err = runConsumer(*productID, query, *langFlag, noContribute)
    }

    if !noTelemetry {
        go sendTelemetry(cliTelemetryPayload{
            Action:    telAction,
            ProductID: telProductID,
            EvalSlug:  telEvalSlug,
            Platform:  runtime.GOOS,
            Version:   Version,
            Success:   err == nil,
        })
    }

    return err
}
```

- [ ] **Step 6: Build to verify compilation**

```bash
cd cli && go build -o /dev/null . && echo "OK"
```

- [ ] **Step 7: Test update check with local backend**

Start the backend, then:

```bash
cd cli && MSDL_API_URL=http://localhost:3002 Version=0.2.0 go run . --list
```

Expected: update notice printed ("A new version of msdl is available: v0.3.0") before the product list. (Note: `Version` env var does NOT override the Go variable — use a built binary with `-ldflags "-X main.Version=0.2.0"` to test this properly.)

Correct test:

```bash
cd cli && go build -ldflags "-X main.Version=0.2.0" -o msdl-test . && \
  MSDL_API_URL=http://localhost:3002 ./msdl-test --list
```

Expected first lines of stderr:

```
  A new version of msdl is available: v0.3.0
  Download: https://github.com/starkSV/windows-iso-downloader/releases/latest
```

- [ ] **Step 8: Test telemetry fires**

After the above run, check Redis:

```bash
redis-cli -u "$REDIS_URL" HGETALL msdl:telemetry:actions
```

Expected: `list` field with value `1`.

```bash
redis-cli -u "$REDIS_URL" HGETALL msdl:telemetry:platforms
redis-cli -u "$REDIS_URL" HGETALL msdl:telemetry:versions
```

Expected: platform and version populated.

- [ ] **Step 9: Test opt-out**

```bash
cd cli && MSDL_NO_TELEMETRY=1 MSDL_API_URL=http://localhost:3002 ./msdl-test --list
```

Expected: no update notice, no new telemetry entries in Redis.

- [ ] **Step 10: Clean up test binary**

```bash
rm cli/msdl-test
```

- [ ] **Step 11: Run existing CLI tests**

```bash
cd cli && go test ./... -v
```

Expected: all existing tests pass (no new tests required — goroutines are integration-tested manually above).

- [ ] **Step 12: Commit**

```bash
git add cli/main.go
git commit -m "feat(cli): update check and anonymous telemetry on every run"
```

---

### Task 7: Documentation updates

**Files:**
- Modify: `README.md`

**Interfaces:** None — doc-only.

- [ ] **Step 1: Add telemetry disclosure to README**

Find the `### Crowdsourced cache` section in `README.md`. Add a new `### Usage telemetry` section immediately after it:

```markdown
### Usage telemetry

By default, each run sends an anonymous event to help us understand which products are popular and which platforms are used. No personal data is sent — only action type (`fetch`, `eval`, `list`, `interactive`), platform (`windows`, `darwin`, `linux`), CLI version, and whether the run succeeded. To opt out:

```bash
MSDL_NO_TELEMETRY=1 msdl --id 3262 --lang "English"
```
```

- [ ] **Step 2: Commit**

```bash
git add README.md
git commit -m "docs: add CLI telemetry disclosure to README"
```

---

## Post-Implementation Checklist

After all tasks are committed to `feat/cli-telemetry`:

- [ ] Run `cd backend && go build -o /dev/null . && echo OK` — must pass
- [ ] Run `cd cli && go test ./... && echo OK` — must pass
- [ ] Verify `/cli/version` endpoint returns correct version
- [ ] Verify `/telemetry` endpoint writes to Redis
- [ ] Verify `/metrics` includes `telemetry` section
- [ ] Verify update notice shows for stale version, suppressed for current version
- [ ] Verify `MSDL_NO_TELEMETRY=1` skips both update check and telemetry
- [ ] **DO NOT push until winget PR #390908 is merged**
