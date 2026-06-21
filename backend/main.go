package main

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
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

// Package-level rand source — avoids global mutex contention under concurrent load.
// Go 1.20+ auto-seeds the global rand, but a local source is faster at high QPS.
var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

const (
	UA          = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	PROFILE     = "606624d44113"
	LOCALE      = "en-US"
	ORG_ID      = "y6jn8c31"
	CUSTOMER_ID = "560dc9f3-1aa5-4a2f-b63c-9e18f8d0e175"
	PORT        = ":3002"
)

// --- Session cache (short-lived, used to chain /skuinfo → /proxy) ---

// simpleCookieJar stores cookies without domain scoping. The standard
// net/http/cookiejar rejects cookies whose Domain (e.g. .microsoft.com) doesn't
// match the request URL (our CF Worker domain). This jar stores all cookies and
// replays them on every request so they flow through the Worker to Microsoft.
type simpleCookieJar struct {
	mu      sync.Mutex
	cookies []*http.Cookie
}

func (j *simpleCookieJar) SetCookies(_ *url.URL, cookies []*http.Cookie) {
	j.mu.Lock()
	defer j.mu.Unlock()
	for _, c := range cookies {
		found := false
		for i, existing := range j.cookies {
			if existing.Name == c.Name {
				j.cookies[i] = c
				found = true
				break
			}
		}
		if !found {
			j.cookies = append(j.cookies, c)
		}
	}
}

func (j *simpleCookieJar) Cookies(_ *url.URL) []*http.Cookie {
	j.mu.Lock()
	defer j.mu.Unlock()
	out := make([]*http.Cookie, len(j.cookies))
	copy(out, j.cookies)
	return out
}

type SessionEntry struct {
	SessionID string
	CreatedAt time.Time
	Jar       *simpleCookieJar
}

var (
	sessionCache = make(map[string]SessionEntry)
	cacheMutex   sync.RWMutex
	SESSION_TTL  = 15 * time.Minute
	workerSecret = os.Getenv("CF_WORKER_SECRET")
	validSkuID   = regexp.MustCompile(`^[a-zA-Z0-9_\-]+$`)
)

// --- SKU info cache (7-day TTL — language lists are stable) ---

type skuCacheEntry struct {
	RawJSON   []byte
	ExpiresAt time.Time
}

var (
	skuCache    = make(map[string]skuCacheEntry)
	skuCacheMu  sync.RWMutex
	skuCacheTTL = 7 * 24 * time.Hour
)

// --- Download link cache (dynamic TTL from URL expiry) ---

type linkCacheEntry struct {
	RawJSON    []byte
	ExpiresAt  time.Time // soft expiry: stop serving without stale fallback
	FetchedAt  time.Time
}

var (
	linkCache   = make(map[string]linkCacheEntry)
	linkCacheMu sync.RWMutex
)

// --- Negative cache (60s — prevents thundering herd during rate-limit events) ---

type negCacheEntry struct {
	Message   string
	HTTPCode  int
	ExpiresAt time.Time
}

var (
	negCache    = make(map[string]negCacheEntry)
	negCacheMu  sync.RWMutex
	negCacheTTL = 60 * time.Second
)

// --- Singleflight (one in-flight Microsoft fetch per unique key) ---

var sfGroup singleflight.Group

// --- Redis L2 cache client ---

var rdb *redis.Client

// --- Metrics counters (atomic — no mutex needed) ---

var (
	// /skuinfo
	mSkuRequests  int64
	mSkuCacheHits int64
	mSkuNegHits   int64
	mSkuFetches   int64 // actual Microsoft calls

	// /proxy
	mLinkRequests  int64
	mLinkCacheHits int64
	mLinkNegHits   int64
	mLinkFetches   int64
	mLinkStale     int64 // stale-on-failure serves

	// /evallinks
	mEvalRequests  int64
	mEvalCacheHits int64
	mEvalStale     int64
)

// --- Evalcenter (Enterprise / Server eval ISOs) ---

type EvalProduct struct {
	Name    string
	EvalURL string
}

type EvalLink struct {
	Arch string `json:"arch"`
	Lang string `json:"lang"`
	URL  string `json:"url"`
}

type EvalLinksResponse struct {
	Product string     `json:"product"`
	Name    string     `json:"name"`
	Links   []EvalLink `json:"links"`
}

type evalCacheEntry struct {
	Links    []EvalLink
	CachedAt time.Time
}

var (
	evalProductMap = map[string]EvalProduct{
		"server-2025": {Name: "Windows Server 2025", EvalURL: "https://www.microsoft.com/en-us/evalcenter/download-windows-server-2025"},
		"server-2022": {Name: "Windows Server 2022", EvalURL: "https://www.microsoft.com/en-us/evalcenter/download-windows-server-2022"},
		"server-2019": {Name: "Windows Server 2019", EvalURL: "https://www.microsoft.com/en-us/evalcenter/download-windows-server-2019"},
		"server-2016": {Name: "Windows Server 2016", EvalURL: "https://www.microsoft.com/en-us/evalcenter/download-windows-server-2016"},
		"win11-ent":   {Name: "Windows 11 Enterprise", EvalURL: "https://www.microsoft.com/en-us/evalcenter/download-windows-11-enterprise"},
	}
	evalCache    = make(map[string]evalCacheEntry)
	evalCacheMu  sync.RWMutex
	evalCacheTTL = 24 * time.Hour
	fwlinkRe     = regexp.MustCompile(`https://go\.microsoft\.com/fwlink/[^"'\s<>]+`)
	isoLangRe    = regexp.MustCompile(`_([a-z]{2}-[a-z]{2})\.iso$`)
)

// --- Helpers ---

// jitter adds a random ±maxJ offset to base duration.
// Prevents synchronised mass-expiry spikes across cache entries.
func jitter(base, maxJ time.Duration) time.Duration {
	offset := time.Duration(rng.Int63n(int64(maxJ*2))) - maxJ
	return base + offset
}

// parseLinkExpiry reads the expiration parameter (se or P1) from the first
// download URL in a Microsoft GetProductDownloadLinksBySku response and returns
// a cache expiry time of (expiration - now) - 30min, with ±5min jitter.
// Falls back to 22h on any parse failure.
func parseLinkExpiry(rawJSON []byte) time.Time {
	fallback := func() time.Time {
		return time.Now().Add(jitter(22*time.Hour, 5*time.Minute))
	}
	expStr := extractRawExpiry(rawJSON)
	if expStr == "" {
		return fallback()
	}
	t, err := time.Parse(time.RFC3339, expStr)
	if err != nil {
		return fallback()
	}
	ttl := time.Until(t) - 30*time.Minute
	if ttl < time.Hour {
		ttl = time.Hour // safety floor — never cache for less than 1h
	}
	return time.Now().Add(jitter(ttl, 5*time.Minute))
}

// extractRawExpiry returns the raw expiration timestamp from the first URL
// in a GetProductDownloadLinksBySku response (RFC3339 string).
// It supports `se` (RFC3339) and `P1` (Unix timestamp) query parameters.
// Returns "" if absent or unparseable.
func extractRawExpiry(rawJSON []byte) string {
	var data map[string]interface{}
	if err := json.Unmarshal(rawJSON, &data); err != nil {
		return ""
	}
	opts, ok := data["ProductDownloadOptions"].([]interface{})
	if !ok || len(opts) == 0 {
		return ""
	}
	first, ok := opts[0].(map[string]interface{})
	if !ok {
		return ""
	}
	uri, ok := first["Uri"].(string)
	if !ok {
		return ""
	}
	u, err := url.Parse(uri)
	if err != nil {
		return ""
	}

	// 1. Try "se" query parameter (used on Server/Eval ISOs)
	if se := u.Query().Get("se"); se != "" {
		if _, err := time.Parse(time.RFC3339, se); err == nil {
			return se
		}
	}

	// 2. Try "P1" query parameter (Unix timestamp, used on Consumer ISOs)
	if p1 := u.Query().Get("P1"); p1 != "" {
		if p1Int, err := strconv.ParseInt(p1, 10, 64); err == nil {
			return time.Unix(p1Int, 0).Format(time.RFC3339)
		}
	}

	return ""
}

// rateLimitError marks a Microsoft rate-limit response so callers can choose
// to serve stale data instead of propagating the error.
type rateLimitError struct{ message string }

func (e *rateLimitError) Error() string { return e.message }

// --- Redis L2 cache helpers ---

func redisKey(productID, skuID string) string {
	return "msdl:link:" + productID + ":" + skuID
}

func initRedis() {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		log.Println("redis: REDIS_URL not set — memory-only mode")
		return
	}
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Printf("redis: invalid REDIS_URL: %v — memory-only mode\n", err)
		return
	}
	client := redis.NewClient(opts)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		log.Printf("redis: ping failed: %v — memory-only mode\n", err)
		return
	}
	rdb = client
	log.Println("redis: connected")
}

func redisWriteLink(productID, skuID string, raw []byte, expiresAt time.Time) {
	if rdb == nil {
		return
	}
	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := rdb.Set(ctx, redisKey(productID, skuID), raw, ttl).Err(); err != nil {
		log.Printf("redis: write failed for %s:%s: %v\n", productID, skuID, err)
	}
}

func redisSeedLinkCache() {
	if rdb == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var cursor uint64
	var seeded int
	for {
		keys, nextCursor, err := rdb.Scan(ctx, cursor, "msdl:link:*", 100).Result()
		if err != nil {
			log.Printf("redis: seed scan error: %v\n", err)
			return
		}
		for _, key := range keys {
			raw, err := rdb.Get(ctx, key).Bytes()
			if err != nil {
				continue
			}
			ttl, err := rdb.TTL(ctx, key).Result()
			if err != nil || ttl <= 0 {
				continue
			}
			// Key format: "msdl:link:<productID>:<skuID>"
			parts := strings.SplitN(strings.TrimPrefix(key, "msdl:link:"), ":", 2)
			if len(parts) != 2 {
				continue
			}
			cacheKey := parts[0] + ":" + parts[1]
			expiresAt := time.Now().Add(ttl)
			linkCacheMu.Lock()
			linkCache[cacheKey] = linkCacheEntry{RawJSON: raw, ExpiresAt: expiresAt, FetchedAt: time.Now()}
			linkCacheMu.Unlock()
			seeded++
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	if seeded > 0 {
		log.Printf("redis: seeded %d link cache entries from Redis\n", seeded)
	}
}

// --- /contribute validation helpers ---

// validContributeProducts is the canonical product ID allow-list for POST /contribute.
var validContributeProducts = map[string]bool{
	"52": true, "2378": true, "2618": true,
	"3113": true, "3114": true, "3115": true,
	"3131": true, "3132": true, "3133": true,
	"3262": true, "3263": true, "3264": true,
	"3265": true, "3266": true, "3267": true,
	"3321": true, "3324": true,
}

// allowedCDNSuffixes is the CDN host allow-list for contributed download URLs.
// Worst-case abuse: attacker submits a valid Microsoft-signed link — harmless.
var allowedCDNSuffixes = []string{
	".download.prss.microsoft.com",
	".dl.delivery.mp.microsoft.com",
	".delivery.mp.microsoft.com",
}

func isAllowedCDNHost(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	for _, suffix := range allowedCDNSuffixes {
		if strings.HasSuffix(host, suffix) {
			return true
		}
	}
	return false
}

// --- Per-IP rate limiter (token bucket, used by /contribute) ---

type tokenBucket struct {
	tokens     float64
	maxTokens  float64
	refillRate float64 // tokens per second
	lastRefill time.Time
}

func (b *tokenBucket) allow() bool {
	now := time.Now()
	elapsed := now.Sub(b.lastRefill).Seconds()
	b.tokens += elapsed * b.refillRate
	if b.tokens > b.maxTokens {
		b.tokens = b.maxTokens
	}
	b.lastRefill = now
	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}

type ipRateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*tokenBucket
	rate    float64
	burst   float64
}

func newIPRateLimiter(ratePerMin, burst float64) *ipRateLimiter {
	rl := &ipRateLimiter{
		buckets: make(map[string]*tokenBucket),
		rate:    ratePerMin / 60.0,
		burst:   burst,
	}
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		for range ticker.C {
			rl.mu.Lock()
			for ip, b := range rl.buckets {
				if b.tokens >= b.maxTokens {
					delete(rl.buckets, ip)
				}
			}
			rl.mu.Unlock()
		}
	}()
	return rl
}

func (rl *ipRateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	b, ok := rl.buckets[ip]
	if !ok {
		b = &tokenBucket{
			tokens:     rl.burst,
			maxTokens:  rl.burst,
			refillRate: rl.rate,
			lastRefill: time.Now(),
		}
		rl.buckets[ip] = b
	}
	return b.allow()
}

var contributeRL = newIPRateLimiter(5, 5) // 5 requests/min, burst of 5

// --- Evalcenter helpers ---

func detectArch(rawURL string) string {
	lower := strings.ToLower(rawURL)
	if strings.Contains(lower, "arm64") {
		return "ARM64"
	}
	if strings.Contains(lower, "x64") {
		return "x64"
	}
	if strings.Contains(lower, "x86") {
		return "x86"
	}
	return "ISO"
}

func detectLang(rawURL string) string {
	m := isoLangRe.FindStringSubmatch(strings.ToLower(rawURL))
	if len(m) > 1 {
		return m[1]
	}
	return ""
}

func fetchEvalLinks(evalURL string) ([]EvalLink, error) {
	client := &http.Client{Timeout: 15 * time.Second}

	req, _ := http.NewRequest("GET", evalURL, nil)
	req.Header.Set("User-Agent", UA)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching evalcenter page: %w", err)
	}
	defer resp.Body.Close()
	bodyBytes, _ := io.ReadAll(resp.Body)

	rawMatches := fwlinkRe.FindAllString(string(bodyBytes), -1)

	seen := map[string]bool{}
	var fwlinks []string
	for _, raw := range rawMatches {
		fwlink := html.UnescapeString(raw)
		if !seen[fwlink] {
			seen[fwlink] = true
			fwlinks = append(fwlinks, fwlink)
		}
	}

	type result struct {
		link EvalLink
		ok   bool
	}
	results := make([]result, len(fwlinks))
	var wg sync.WaitGroup
	for i, fwlink := range fwlinks {
		wg.Add(1)
		go func(idx int, fw string) {
			defer wg.Done()
			fwReq, _ := http.NewRequest("GET", fw, nil)
			fwReq.Header.Set("User-Agent", UA)
			fwResp, err := client.Do(fwReq)
			if err != nil {
				return
			}
			fwResp.Body.Close()
			finalURL := fwResp.Request.URL.String()
			if !strings.Contains(strings.ToLower(finalURL), ".iso") {
				return
			}
			results[idx] = result{
				link: EvalLink{Arch: detectArch(finalURL), Lang: detectLang(finalURL), URL: finalURL},
				ok:   true,
			}
		}(i, fwlink)
	}
	wg.Wait()

	var links []EvalLink
	for _, r := range results {
		if r.ok {
			links = append(links, r.link)
		}
	}

	sort.Slice(links, func(i, j int) bool {
		if links[i].Lang == "en-us" {
			return true
		}
		if links[j].Lang == "en-us" {
			return false
		}
		return links[i].Lang < links[j].Lang
	})

	return links, nil
}

// --- CF Worker / session helpers ---

func setWorkerSecret(req *http.Request) {
	if workerSecret != "" {
		req.Header.Set("X-Worker-Secret", workerSecret)
	}
}

func proxyURL(msURL string) string {
	workerBase := os.Getenv("CF_WORKER_URL")
	if workerBase == "" {
		return msURL
	}
	parsed, err := url.Parse(msURL)
	if err != nil {
		return msURL
	}
	workerParsed, err := url.Parse(workerBase)
	if err != nil {
		return msURL
	}
	q := parsed.Query()
	q.Set("host", parsed.Hostname())
	workerParsed.Path = parsed.Path
	workerParsed.RawQuery = q.Encode()
	return workerParsed.String()
}

func getReferer(productId string) string {
	id, err := strconv.Atoi(productId)
	if err != nil {
		return "https://www.microsoft.com/en-us/software-download/windows8ISO"
	}
	if id >= 2935 {
		return "https://www.microsoft.com/en-us/software-download/windows11"
	}
	if id >= 2618 {
		return "https://www.microsoft.com/en-us/software-download/windows10ISO"
	}
	return "https://www.microsoft.com/en-us/software-download/windows8ISO"
}

func setupSession() (string, *simpleCookieJar, error) {
	sessionID := uuid.New().String()
	jar := &simpleCookieJar{}
	client := &http.Client{Timeout: 10 * time.Second, Jar: jar}

	q1 := url.Values{}
	q1.Set("org_id", ORG_ID)
	q1.Set("session_id", sessionID)
	req1, _ := http.NewRequest("GET", proxyURL("https://vlscppe.microsoft.com/tags?"+q1.Encode()), nil)
	req1.Header.Set("User-Agent", UA)
	setWorkerSecret(req1)
	client.Do(req1)

	q2 := url.Values{}
	q2.Set("instanceId", CUSTOMER_ID)
	q2.Set("PageId", "si")
	q2.Set("session_id", sessionID)
	req2, _ := http.NewRequest("GET", proxyURL("https://ov-df.microsoft.com/mdt.js?"+q2.Encode()), nil)
	req2.Header.Set("User-Agent", UA)
	setWorkerSecret(req2)
	resp2, err := client.Do(req2)
	if err != nil {
		return sessionID, jar, nil
	}
	defer resp2.Body.Close()

	bodyBytes, _ := io.ReadAll(resp2.Body)
	mdtText := string(bodyBytes)

	reW := regexp.MustCompile(`[&?]w=([^&"'\s]+)`)
	reRt := regexp.MustCompile(`rticks[="]+\+?\s*(\d{10,})`)

	wMatch := reW.FindStringSubmatch(mdtText)
	rtMatch := reRt.FindStringSubmatch(mdtText)

	if len(wMatch) > 1 && len(rtMatch) > 1 {
		wVal := wMatch[1]
		rtVal := rtMatch[1]
		mdt := time.Now().UnixMilli()

		q3 := url.Values{}
		q3.Set("session_id", sessionID)
		q3.Set("CustomerId", CUSTOMER_ID)
		q3.Set("PageId", "si")
		q3.Set("w", wVal)
		q3.Set("mdt", fmt.Sprintf("%d", mdt))
		q3.Set("rticks", rtVal)
		fpURL := proxyURL("https://ov-df.microsoft.com/?" + q3.Encode())

		req3, _ := http.NewRequest("GET", fpURL, nil)
		req3.Header.Set("User-Agent", UA)
		setWorkerSecret(req3)
		client.Do(req3)
	}

	return sessionID, jar, nil
}

func mapDownloadType(typeNum int) string {
	switch typeNum {
	case 0:
		return "x86"
	case 1:
		return "x64"
	case 2:
		return "ARM64"
	default:
		return fmt.Sprintf("type_%d", typeNum)
	}
}

// --- Microsoft fetch functions (called via singleflight) ---

// fetchSkuInfoFromMS performs the full Microsoft session + SKU info fetch.
// Returns (rawJSON, httpErrorCode, error). httpErrorCode is non-zero only on
// rate-limit/API errors that should be stored in the negative cache.
func fetchSkuInfoFromMS(productID string) ([]byte, int, error) {
	sessionID, jar, _ := setupSession()

	cacheMutex.Lock()
	sessionCache[productID] = SessionEntry{SessionID: sessionID, CreatedAt: time.Now(), Jar: jar}
	cacheMutex.Unlock()

	skuQ := url.Values{}
	skuQ.Set("profile", PROFILE)
	skuQ.Set("productEditionId", productID)
	skuQ.Set("SKU", "undefined")
	skuQ.Set("friendlyFileName", "undefined")
	skuQ.Set("Locale", LOCALE)
	skuQ.Set("sessionID", sessionID)
	reqURL := proxyURL("https://www.microsoft.com/software-download-connector/api/getskuinformationbyproductedition?" + skuQ.Encode())

	client := &http.Client{Timeout: 15 * time.Second, Jar: jar}
	var finalData map[string]interface{}
	var finalRaw []byte

	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(2 * time.Second)
		}
		req, _ := http.NewRequest("GET", reqURL, nil)
		req.Header.Set("User-Agent", UA)
		req.Header.Set("Referer", getReferer(productID))
		req.Header.Set("Accept", "application/json")
		setWorkerSecret(req)

		resp, err := client.Do(req)
		if err != nil {
			if attempt == 2 {
				return nil, 0, fmt.Errorf("request failed: %w", err)
			}
			continue
		}

		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == http.StatusTooManyRequests {
			return nil, http.StatusTooManyRequests, &rateLimitError{"Microsoft is rate-limiting requests. Please try again shortly."}
		}

		if resp.StatusCode != http.StatusOK {
			if attempt == 2 {
				return nil, http.StatusBadGateway, fmt.Errorf("Microsoft API returned HTTP %d: %s", resp.StatusCode, string(bodyBytes))
			}
			continue
		}

		if len(bodyBytes) > 0 && bodyBytes[0] == '"' {
			var unquoted string
			json.Unmarshal(bodyBytes, &unquoted)
			bodyBytes = []byte(unquoted)
		}

		var data map[string]interface{}
		json.Unmarshal(bodyBytes, &data)

		if errs, ok := data["Errors"].([]interface{}); ok && len(errs) > 0 {
			if attempt == 2 {
				msg := "Microsoft API error"
				code := http.StatusBadGateway
				if errMap, ok := errs[0].(map[string]interface{}); ok {
					if val, exists := errMap["Value"]; exists {
						msg = val.(string)
					}
					// 715-123130 rate-limit code
					if typeNum, ok := errMap["Type"].(float64); ok && int(typeNum) == 9 {
						code = http.StatusTooManyRequests
						return nil, code, &rateLimitError{msg}
					}
				}
				return nil, code, fmt.Errorf("%s", msg)
			}
			continue
		}

		finalData = data
		finalRaw = bodyBytes
		break
	}

	skus, ok := finalData["Skus"].([]interface{})
	if !ok || len(skus) == 0 {
		return nil, http.StatusNotFound, fmt.Errorf("no languages found for this product ID")
	}

	log.Printf("MS fetch /skuinfo: product_id=%s -> %d languages\n", productID, len(skus))
	return finalRaw, 0, nil
}

// fetchDownloadLinksFromMS performs the Microsoft download link fetch.
// Returns (rawJSON, error). rateLimitError is returned on 429/715-123130.
func fetchDownloadLinksFromMS(productID, skuID string) ([]byte, error) {
	var sessionID string
	var jar *simpleCookieJar
	cacheMutex.RLock()
	cached, exists := sessionCache[productID]
	cacheMutex.RUnlock()

	if exists && time.Since(cached.CreatedAt) < SESSION_TTL {
		sessionID = cached.SessionID
		jar = cached.Jar
		log.Printf("MS fetch /proxy: reusing session %s for product_id=%s\n", sessionID[:8], productID)
	} else {
		log.Printf("MS fetch /proxy: new session for product_id=%s\n", productID)
		sessionID, jar, _ = setupSession()

		// Warm the session with a SKU info call so cookies from setup carry through
		warmupQ := url.Values{}
		warmupQ.Set("profile", PROFILE)
		warmupQ.Set("productEditionId", productID)
		warmupQ.Set("SKU", "undefined")
		warmupQ.Set("friendlyFileName", "undefined")
		warmupQ.Set("Locale", LOCALE)
		warmupQ.Set("sessionID", sessionID)
		warmupURL := proxyURL("https://www.microsoft.com/software-download-connector/api/getskuinformationbyproductedition?" + warmupQ.Encode())

		warmupClient := &http.Client{Timeout: 10 * time.Second, Jar: jar}
		req, _ := http.NewRequest("GET", warmupURL, nil)
		req.Header.Set("User-Agent", UA)
		req.Header.Set("Referer", getReferer(productID))
		req.Header.Set("Accept", "application/json")
		setWorkerSecret(req)
		warmupClient.Do(req)

		cacheMutex.Lock()
		sessionCache[productID] = SessionEntry{SessionID: sessionID, CreatedAt: time.Now(), Jar: jar}
		cacheMutex.Unlock()
	}

	// Guard against sessions cached before the cookie jar was introduced
	if jar == nil {
		jar = &simpleCookieJar{}
	}


	proxyQ := url.Values{}
	proxyQ.Set("profile", PROFILE)
	proxyQ.Set("productEditionId", "undefined")
	proxyQ.Set("SKU", skuID)
	proxyQ.Set("friendlyFileName", "undefined")
	proxyQ.Set("Locale", LOCALE)
	proxyQ.Set("sessionID", sessionID)
	reqURL := proxyURL("https://www.microsoft.com/software-download-connector/api/GetProductDownloadLinksBySku?" + proxyQ.Encode())

	downloadClient := &http.Client{Timeout: 15 * time.Second, Jar: jar}
	req, _ := http.NewRequest("GET", reqURL, nil)
	req.Header.Set("User-Agent", UA)
	req.Header.Set("Referer", getReferer(productID))
	req.Header.Set("Accept", "application/json")
	setWorkerSecret(req)

	resp, err := downloadClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, &rateLimitError{"Microsoft is rate-limiting requests. Please try again shortly."}
	}

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Microsoft API returned HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}

	if len(bodyBytes) > 0 && bodyBytes[0] == '"' {
		var unquoted string
		json.Unmarshal(bodyBytes, &unquoted)
		bodyBytes = []byte(unquoted)
	}

	var data map[string]interface{}
	json.Unmarshal(bodyBytes, &data)

	if errs, ok := data["Errors"].([]interface{}); ok && len(errs) > 0 {
		if errMap, ok := errs[0].(map[string]interface{}); ok {
			if typeNum, exists := errMap["Type"].(float64); exists && int(typeNum) == 9 {
				msg := "Your IP has been temporarily blocked by Microsoft. Please try again later. (Code 715-123130)"
				if val, ok := errMap["Value"].(string); ok {
					msg = val
				}
				return nil, &rateLimitError{msg}
			}
			if val, exists := errMap["Value"].(string); exists {
				return nil, fmt.Errorf("%s", val)
			}
		}
		return nil, fmt.Errorf("Microsoft API error")
	}

	opts, ok := data["ProductDownloadOptions"].([]interface{})
	if !ok || len(opts) == 0 {
		return nil, fmt.Errorf("no download links found for this SKU")
	}

	for _, optRaw := range opts {
		if optMap, ok := optRaw.(map[string]interface{}); ok {
			if arch, hasArch := optMap["Architecture"]; !hasArch || arch == nil {
				if dType, hasDType := optMap["DownloadType"].(float64); hasDType {
					optMap["Architecture"] = mapDownloadType(int(dType))
				}
			}
		}
	}

	log.Printf("MS fetch /proxy: product_id=%s sku_id=%s -> %d links\n", productID, skuID, len(opts))

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.Encode(data)
	return buf.Bytes(), nil
}

// --- Background helpers ---

func cleanupSessions() {
	for {
		time.Sleep(1 * time.Minute)
		cacheMutex.Lock()
		now := time.Now()
		for key, entry := range sessionCache {
			if now.Sub(entry.CreatedAt) > SESSION_TTL {
				delete(sessionCache, key)
			}
		}
		cacheMutex.Unlock()
	}
}

// --- CORS middleware ---

func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "https://msdl.tech-latest.com" || strings.HasPrefix(origin, "http://localhost:") || strings.HasPrefix(origin, "http://100.") || strings.HasPrefix(origin, "http://192.168.") {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Expose-Headers", "X-MSDL-Link-Status, X-MSDL-Link-Expires")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func respondJSONError(w http.ResponseWriter, status int, message string) {
	log.Printf("ERROR [%d]: %s\n", status, message)
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// --- /skuinfo endpoint ---

func handleSkuInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	productID := r.URL.Query().Get("product_id")
	if productID == "" {
		respondJSONError(w, http.StatusBadRequest, "product_id query parameter is required")
		return
	}
	if _, err := strconv.Atoi(productID); err != nil {
		respondJSONError(w, http.StatusBadRequest, "product_id must be a numeric value")
		return
	}

	atomic.AddInt64(&mSkuRequests, 1)
	negKey := "sku:" + productID

	// 1. Check negative cache (short-circuit during rate-limit events)
	negCacheMu.RLock()
	neg, hasNeg := negCache[negKey]
	negCacheMu.RUnlock()
	if hasNeg && time.Now().Before(neg.ExpiresAt) {
		atomic.AddInt64(&mSkuNegHits, 1)
		log.Printf("/skuinfo: product_id=%s -> neg cache hit\n", productID)
		respondJSONError(w, neg.HTTPCode, neg.Message)
		return
	}

	// 2. Check SKU cache (7-day TTL)
	skuCacheMu.RLock()
	entry, hasSku := skuCache[productID]
	skuCacheMu.RUnlock()
	if hasSku && time.Now().Before(entry.ExpiresAt) {
		atomic.AddInt64(&mSkuCacheHits, 1)
		log.Printf("/skuinfo: product_id=%s -> cache hit\n", productID)
		w.Write(entry.RawJSON)
		return
	}

	// 3. Singleflight: collapse concurrent cache misses into one Microsoft fetch
	sfKey := "sku:" + productID
	type sfResult struct {
		raw  []byte
		code int
	}
	v, err, _ := sfGroup.Do(sfKey, func() (interface{}, error) {
		raw, code, err := fetchSkuInfoFromMS(productID)
		return sfResult{raw: raw, code: code}, err
	})

	if err != nil {
		res := v.(sfResult)
		code := res.code
		if code == 0 {
			code = http.StatusBadGateway
		}
		// 4. Store in negative cache only for rate-limit / API errors
		if _, isRL := err.(*rateLimitError); isRL || code == http.StatusTooManyRequests {
			negCacheMu.Lock()
			negCache[negKey] = negCacheEntry{
				Message:   err.Error(),
				HTTPCode:  http.StatusTooManyRequests,
				ExpiresAt: time.Now().Add(negCacheTTL),
			}
			negCacheMu.Unlock()
		}
		respondJSONError(w, code, err.Error())
		return
	}

	res := v.(sfResult)

	// 5. Store in SKU cache with 7-day TTL + jitter
	skuCacheMu.Lock()
	skuCache[productID] = skuCacheEntry{
		RawJSON:   res.raw,
		ExpiresAt: time.Now().Add(jitter(skuCacheTTL, 5*time.Minute)),
	}
	skuCacheMu.Unlock()

	atomic.AddInt64(&mSkuFetches, 1)
	log.Printf("/skuinfo: product_id=%s -> fetched and cached\n", productID)
	w.Write(res.raw)
}

// --- /proxy endpoint ---

func handleProxy(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	productID := r.URL.Query().Get("product_id")
	skuID := r.URL.Query().Get("sku_id")

	if productID == "" || skuID == "" {
		respondJSONError(w, http.StatusBadRequest, "product_id and sku_id query parameters are required")
		return
	}
	if _, err := strconv.Atoi(productID); err != nil {
		respondJSONError(w, http.StatusBadRequest, "product_id must be a numeric value")
		return
	}
	if !validSkuID.MatchString(skuID) {
		respondJSONError(w, http.StatusBadRequest, "sku_id contains invalid characters")
		return
	}

	atomic.AddInt64(&mLinkRequests, 1)
	cacheKey := productID + ":" + skuID
	negKey := "link:" + cacheKey
	forceRefresh := r.URL.Query().Get("force") == "true"

	// 1. Check link cache (dynamic TTL, stale-on-failure) — skipped when force=true
	linkCacheMu.RLock()
	cached, hasCached := linkCache[cacheKey]
	linkCacheMu.RUnlock()

	if !forceRefresh && hasCached && time.Now().Before(cached.ExpiresAt) {
		atomic.AddInt64(&mLinkCacheHits, 1)
		log.Printf("/proxy: product_id=%s sku_id=%s -> cache hit\n", productID, skuID)
		w.Header().Set("X-MSDL-Link-Status", "cached")
		if exp := extractRawExpiry(cached.RawJSON); exp != "" {
			w.Header().Set("X-MSDL-Link-Expires", exp)
		}
		w.Write(cached.RawJSON)
		return
	}

	// 1b. Redis L2 cache — check after in-memory miss, before calling Microsoft
	if !forceRefresh && !hasCached && rdb != nil {
		rctx, rcancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		redisRaw, redisErr := rdb.Get(rctx, redisKey(productID, skuID)).Bytes()
		rcancel()
		if redisErr == nil && len(redisRaw) > 0 {
			exp := parseLinkExpiry(redisRaw)
			if time.Now().Before(exp) {
				linkCacheMu.Lock()
				linkCache[cacheKey] = linkCacheEntry{RawJSON: redisRaw, ExpiresAt: exp, FetchedAt: time.Now()}
				linkCacheMu.Unlock()
				atomic.AddInt64(&mLinkCacheHits, 1)
				log.Printf("/proxy: product_id=%s sku_id=%s -> redis L2 hit\n", productID, skuID)
				w.Header().Set("X-MSDL-Link-Status", "cached")
				if rawExp := extractRawExpiry(redisRaw); rawExp != "" {
					w.Header().Set("X-MSDL-Link-Expires", rawExp)
				}
				w.Write(redisRaw)
				return
			}
		}
	}

	// 2. Check negative cache — but only if no stale data and not a force refresh
	if !forceRefresh && !hasCached {
		negCacheMu.RLock()
		neg, hasNeg := negCache[negKey]
		negCacheMu.RUnlock()
		if hasNeg && time.Now().Before(neg.ExpiresAt) {
			atomic.AddInt64(&mLinkNegHits, 1)
			log.Printf("/proxy: product_id=%s sku_id=%s -> neg cache hit\n", productID, skuID)
			respondJSONError(w, neg.HTTPCode, neg.Message)
			return
		}
	}

	// 3. Singleflight: one fetch per (product, sku) key
	sfKey := "link:" + cacheKey
	if forceRefresh {
		sfKey = "link:force:" + cacheKey
	}
	v, err, _ := sfGroup.Do(sfKey, func() (interface{}, error) {
		return fetchDownloadLinksFromMS(productID, skuID)
	})

	if err != nil {
		// 6. Stale-on-failure: serve expired cache entry rather than erroring out
		// Also fire a background refresh so the next request gets fresh data.
		if hasCached {
			atomic.AddInt64(&mLinkStale, 1)
			log.Printf("/proxy: product_id=%s sku_id=%s -> fetch failed (%v), serving stale\n", productID, skuID, err)
			go func() {
				// Use singleflight so concurrent stale serves don't all hit Microsoft
				sfBgKey := "link:" + cacheKey
				sfGroup.Do(sfBgKey, func() (interface{}, error) {
					raw, bgErr := fetchDownloadLinksFromMS(productID, skuID)
					if bgErr != nil {
						log.Printf("/proxy: background refresh failed for %s:%s: %v\n", productID, skuID, bgErr)
						return nil, bgErr
					}
					exp := parseLinkExpiry(raw)
					linkCacheMu.Lock()
					linkCache[cacheKey] = linkCacheEntry{RawJSON: raw, ExpiresAt: exp, FetchedAt: time.Now()}
					linkCacheMu.Unlock()
					redisWriteLink(productID, skuID, raw, exp)
					log.Printf("/proxy: background refresh succeeded for %s:%s, cached until %s\n", productID, skuID, exp.Format(time.RFC3339))
					return nil, nil
				})
			}()
			w.Header().Set("X-MSDL-Link-Status", "stale")
			if exp := extractRawExpiry(cached.RawJSON); exp != "" {
				w.Header().Set("X-MSDL-Link-Expires", exp)
			}
			w.Write(cached.RawJSON)
			return
		}

		code := http.StatusBadGateway
		// 4. Negative cache for rate-limit errors (only when no stale available)
		if _, isRL := err.(*rateLimitError); isRL {
			code = http.StatusTooManyRequests
			negCacheMu.Lock()
			negCache[negKey] = negCacheEntry{
				Message:   err.Error(),
				HTTPCode:  code,
				ExpiresAt: time.Now().Add(negCacheTTL),
			}
			negCacheMu.Unlock()
		}
		respondJSONError(w, code, err.Error())
		return
	}

	raw := v.([]byte)

	// 5. Dynamic TTL: parse `se` from signed URL, subtract 30min buffer, add jitter
	expiresAt := parseLinkExpiry(raw)

	linkCacheMu.Lock()
	linkCache[cacheKey] = linkCacheEntry{
		RawJSON:   raw,
		ExpiresAt: expiresAt,
		FetchedAt: time.Now(),
	}
	linkCacheMu.Unlock()
	redisWriteLink(productID, skuID, raw, expiresAt)

	atomic.AddInt64(&mLinkFetches, 1)
	if forceRefresh {
		log.Printf("/proxy: product_id=%s sku_id=%s -> force refresh, cached until %s\n", productID, skuID, expiresAt.Format(time.RFC3339))
	} else {
		log.Printf("/proxy: product_id=%s sku_id=%s -> fetched and cached until %s\n", productID, skuID, expiresAt.Format(time.RFC3339))
	}
	w.Header().Set("X-MSDL-Link-Status", "fresh")
	if exp := extractRawExpiry(raw); exp != "" {
		w.Header().Set("X-MSDL-Link-Expires", exp)
	}
	w.Write(raw)
}

// --- /contribute endpoint ---

func handleContribute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	// Check CONTRIBUTE_SECRET header
	if secret := os.Getenv("CONTRIBUTE_SECRET"); secret != "" {
		if r.Header.Get("X-Contribute-Secret") != secret {
			respondJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
	}

	// Per-IP rate limit (~5/min)
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		ip = strings.TrimSpace(strings.SplitN(fwd, ",", 2)[0])
	}
	if !contributeRL.allow(ip) {
		respondJSONError(w, http.StatusTooManyRequests, "rate limit exceeded")
		return
	}

	// Parse body
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1 MB cap
	if err != nil {
		respondJSONError(w, http.StatusBadRequest, "failed to read body")
		return
	}
	var req struct {
		ProductID string          `json:"product_id"`
		SkuID     string          `json:"sku_id"`
		RawJSON   json.RawMessage `json:"raw_json"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		respondJSONError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	productID := req.ProductID
	skuID := req.SkuID

	// Validation 1: product in catalog
	if !validContributeProducts[productID] {
		log.Printf("contribute: product_id=%s sku_id=%s -> rejected (unknown product)\n", productID, skuID)
		respondJSONError(w, http.StatusUnprocessableEntity, "unknown product_id")
		return
	}

	// Validation 2: sku_id format
	if !validSkuID.MatchString(skuID) {
		log.Printf("contribute: product_id=%s sku_id=%s -> rejected (invalid sku_id format)\n", productID, skuID)
		respondJSONError(w, http.StatusUnprocessableEntity, "invalid sku_id")
		return
	}

	raw := []byte(req.RawJSON)
	if len(raw) == 0 {
		respondJSONError(w, http.StatusBadRequest, "raw_json is required")
		return
	}

	// Validation 3: expiry must be > 1 hour out.
	// Use extractRawExpiry (direct se timestamp) — NOT parseLinkExpiry, which has
	// a 1h safety floor that would accept already-expired URLs.
	seStr := extractRawExpiry(raw)
	if seStr == "" {
		log.Printf("contribute: product_id=%s sku_id=%s -> rejected (no valid se param)\n", productID, skuID)
		respondJSONError(w, http.StatusUnprocessableEntity, "no valid signed expiry found in download URL")
		return
	}
	seTime, _ := time.Parse(time.RFC3339, seStr)
	if time.Until(seTime) < time.Hour {
		log.Printf("contribute: product_id=%s sku_id=%s -> rejected (expiry < 1h: %s)\n", productID, skuID, seStr)
		respondJSONError(w, http.StatusUnprocessableEntity, "link expires in less than 1 hour")
		return
	}
	expiresAt := parseLinkExpiry(raw) // cache TTL (se minus 30m buffer + jitter)

	// Validation 4: all download URLs must be from allowed Microsoft CDN hosts
	var data map[string]interface{}
	if err := json.Unmarshal(raw, &data); err != nil {
		respondJSONError(w, http.StatusUnprocessableEntity, "raw_json is not valid JSON")
		return
	}
	opts, _ := data["ProductDownloadOptions"].([]interface{})
	if len(opts) == 0 {
		log.Printf("contribute: product_id=%s sku_id=%s -> rejected (no ProductDownloadOptions)\n", productID, skuID)
		respondJSONError(w, http.StatusUnprocessableEntity, "raw_json has no ProductDownloadOptions")
		return
	}
	for _, opt := range opts {
		m, ok := opt.(map[string]interface{})
		if !ok {
			continue
		}
		uri, _ := m["Uri"].(string)
		if uri == "" || !isAllowedCDNHost(uri) {
			log.Printf("contribute: product_id=%s sku_id=%s -> rejected (disallowed host: %s)\n", productID, skuID, uri)
			respondJSONError(w, http.StatusUnprocessableEntity, "download URL host not in allow-list")
			return
		}
	}

	// Write to in-memory cache + Redis
	cacheKey := productID + ":" + skuID
	linkCacheMu.Lock()
	linkCache[cacheKey] = linkCacheEntry{RawJSON: raw, ExpiresAt: expiresAt, FetchedAt: time.Now()}
	linkCacheMu.Unlock()
	redisWriteLink(productID, skuID, raw, expiresAt)

	log.Printf("contribute: product_id=%s sku_id=%s -> accepted, cached until %s\n", productID, skuID, expiresAt.Format(time.RFC3339))
	w.WriteHeader(http.StatusNoContent)
}

// --- /evallinks endpoint ---

func handleEvalLinks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	product := r.URL.Query().Get("product")
	if product == "" {
		respondJSONError(w, http.StatusBadRequest, "product query parameter is required")
		return
	}

	evalProduct, ok := evalProductMap[product]
	if !ok {
		respondJSONError(w, http.StatusNotFound, "Unknown eval product: "+product)
		return
	}

	atomic.AddInt64(&mEvalRequests, 1)
	evalCacheMu.RLock()
	cached, exists := evalCache[product]
	evalCacheMu.RUnlock()

	if exists && time.Since(cached.CachedAt) < evalCacheTTL {
		atomic.AddInt64(&mEvalCacheHits, 1)
		log.Printf("/evallinks: product=%s -> cache hit (%d links)\n", product, len(cached.Links))
		json.NewEncoder(w).Encode(EvalLinksResponse{Product: product, Name: evalProduct.Name, Links: cached.Links})
		return
	}

	links, err := fetchEvalLinks(evalProduct.EvalURL)
	if err != nil {
		// Stale-on-failure for eval links too
		if exists {
			atomic.AddInt64(&mEvalStale, 1)
			log.Printf("/evallinks: product=%s -> fetch failed, serving stale\n", product)
			json.NewEncoder(w).Encode(EvalLinksResponse{Product: product, Name: evalProduct.Name, Links: cached.Links})
			return
		}
		respondJSONError(w, http.StatusBadGateway, "Failed to fetch eval links: "+err.Error())
		return
	}
	if len(links) == 0 {
		if exists {
			atomic.AddInt64(&mEvalStale, 1)
			log.Printf("/evallinks: product=%s -> 0 links returned, serving stale\n", product)
			json.NewEncoder(w).Encode(EvalLinksResponse{Product: product, Name: evalProduct.Name, Links: cached.Links})
			return
		}
		respondJSONError(w, http.StatusNotFound, "No download links found for this eval product")
		return
	}

	evalCacheMu.Lock()
	evalCache[product] = evalCacheEntry{Links: links, CachedAt: time.Now()}
	evalCacheMu.Unlock()

	log.Printf("/evallinks: product=%s -> %d links cached\n", product, len(links))
	json.NewEncoder(w).Encode(EvalLinksResponse{Product: product, Name: evalProduct.Name, Links: links})
}

// --- Cache eviction (prevents unbounded map growth) ---

func cleanupCaches() {
	ticker := time.NewTicker(30 * time.Minute)
	for range ticker.C {
		now := time.Now()
		var skuDel, linkDel, negDel int

		skuCacheMu.Lock()
		for k, v := range skuCache {
			if now.After(v.ExpiresAt) {
				delete(skuCache, k)
				skuDel++
			}
		}
		skuCacheMu.Unlock()

		// Keep link entries for 4h past soft expiry — stale-on-failure window
		linkCacheMu.Lock()
		for k, v := range linkCache {
			if now.After(v.ExpiresAt.Add(4 * time.Hour)) {
				delete(linkCache, k)
				linkDel++
			}
		}
		linkCacheMu.Unlock()

		negCacheMu.Lock()
		for k, v := range negCache {
			if now.After(v.ExpiresAt) {
				delete(negCache, k)
				negDel++
			}
		}
		negCacheMu.Unlock()

		if skuDel+linkDel+negDel > 0 {
			log.Printf("cache cleanup: evicted sku=%d link=%d neg=%d\n", skuDel, linkDel, negDel)
		}
	}
}

// --- /metrics endpoint ---

func handleMetrics(w http.ResponseWriter, r *http.Request) {
	secret := os.Getenv("METRICS_SECRET")
	if secret == "" {
		respondJSONError(w, http.StatusForbidden, "metrics endpoint not configured")
		return
	}

	provided := r.URL.Query().Get("secret")
	if provided == "" {
		provided = strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	}
	if provided != secret {
		respondJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	skuReqs := atomic.LoadInt64(&mSkuRequests)
	skuHits := atomic.LoadInt64(&mSkuCacheHits)
	skuFetches := atomic.LoadInt64(&mSkuFetches)
	skuNeg := atomic.LoadInt64(&mSkuNegHits)

	linkReqs := atomic.LoadInt64(&mLinkRequests)
	linkHits := atomic.LoadInt64(&mLinkCacheHits)
	linkFetches := atomic.LoadInt64(&mLinkFetches)
	linkNeg := atomic.LoadInt64(&mLinkNegHits)
	linkStale := atomic.LoadInt64(&mLinkStale)

	evalReqs := atomic.LoadInt64(&mEvalRequests)
	evalHits := atomic.LoadInt64(&mEvalCacheHits)
	evalStale := atomic.LoadInt64(&mEvalStale)

	hitRate := func(hits, total int64) string {
		if total == 0 {
			return "N/A"
		}
		return fmt.Sprintf("%.1f%%", float64(hits)/float64(total)*100)
	}

	skuCacheMu.RLock()
	skuSize := len(skuCache)
	skuCacheMu.RUnlock()
	linkCacheMu.RLock()
	linkSize := len(linkCache)
	linkCacheMu.RUnlock()
	negCacheMu.RLock()
	negSize := len(negCache)
	negCacheMu.RUnlock()
	evalCacheMu.RLock()
	evalSize := len(evalCache)
	evalCacheMu.RUnlock()

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
	})
}

func warmEvalCache() {
	for slug, product := range evalProductMap {
		go func(s string, p EvalProduct) {
			links, err := fetchEvalLinks(p.EvalURL)
			if err != nil || len(links) == 0 {
				log.Printf("eval warm: %s failed: %v\n", s, err)
				return
			}
			evalCacheMu.Lock()
			evalCache[s] = evalCacheEntry{Links: links, CachedAt: time.Now()}
			evalCacheMu.Unlock()
			log.Printf("eval warm: %s -> %d links cached\n", s, len(links))
		}(slug, product)
	}
}

func main() {
	initRedis()
	go redisSeedLinkCache()
	go cleanupSessions()
	go cleanupCaches()
	go warmEvalCache()

	mux := http.NewServeMux()

	mux.HandleFunc("/skuinfo", handleSkuInfo)
	mux.HandleFunc("/proxy", handleProxy)
	mux.HandleFunc("/contribute", handleContribute)
	mux.HandleFunc("/evallinks", handleEvalLinks)
	mux.HandleFunc("/metrics", handleMetrics)
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

	log.Printf("Go Backend running on http://localhost%s\n", PORT)
	err := http.ListenAndServe(PORT, handler)
	if err != nil {
		log.Fatal(err)
	}
}
