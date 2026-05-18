package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	UA          = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	PROFILE     = "606624d44113"
	LOCALE      = "en-US"
	ORG_ID      = "y6jn8c31"
	CUSTOMER_ID = "560dc9f3-1aa5-4a2f-b63c-9e18f8d0e175"
	PORT        = ":3002" // Different port to run alongside Node
)

type SessionEntry struct {
	SessionID string
	CreatedAt time.Time
}

var (
	sessionCache = make(map[string]SessionEntry)
	cacheMutex   sync.RWMutex
	SESSION_TTL  = 15 * time.Minute
	workerSecret = os.Getenv("CF_WORKER_SECRET")
	validSkuID   = regexp.MustCompile(`^[a-zA-Z0-9_\-]+$`)
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
	evalCache      = make(map[string]evalCacheEntry)
	evalCacheMu    sync.RWMutex
	evalCacheTTL   = 24 * time.Hour
	fwlinkRe  = regexp.MustCompile(`https://go\.microsoft\.com/fwlink/[^"'\s<>]+`)
	isoLangRe = regexp.MustCompile(`_([a-z]{2}-[a-z]{2})\.iso$`)
)

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

	// Step 1: Fetch the evalcenter page to extract fwlinks
	req, _ := http.NewRequest("GET", evalURL, nil)
	req.Header.Set("User-Agent", UA)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching evalcenter page: %w", err)
	}
	defer resp.Body.Close()
	bodyBytes, _ := io.ReadAll(resp.Body)

	// Step 2: Extract all fwlink URLs (unescape HTML entities like &amp; → &)
	rawMatches := fwlinkRe.FindAllString(string(bodyBytes), -1)

	// Deduplicate fwlinks
	seen := map[string]bool{}
	var fwlinks []string
	for _, raw := range rawMatches {
		fwlink := html.UnescapeString(raw)
		if !seen[fwlink] {
			seen[fwlink] = true
			fwlinks = append(fwlinks, fwlink)
		}
	}

	// Step 3: Follow all redirects in parallel
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

	// Sort: en-us first, then alphabetically by lang
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

func setWorkerSecret(req *http.Request) {
	if workerSecret != "" {
		req.Header.Set("X-Worker-Secret", workerSecret)
	}
}

// proxyURL rewrites a Microsoft URL to go through the CF Worker when CF_WORKER_URL is set.
// The Worker receives the original host via ?host= and forwards the request from Cloudflare's edge.
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

// Map product ID to referer URL
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

// Replicate Fido session tracking
func setupSession() (string, error) {
	sessionID := uuid.New().String()
	client := &http.Client{Timeout: 10 * time.Second}

	// Step 1: Register session
	q1 := url.Values{}
	q1.Set("org_id", ORG_ID)
	q1.Set("session_id", sessionID)
	req1, _ := http.NewRequest("GET", proxyURL("https://vlscppe.microsoft.com/tags?"+q1.Encode()), nil)
	req1.Header.Set("User-Agent", UA)
	setWorkerSecret(req1)
	client.Do(req1) // Ignore errors intentionally

	// Step 2: Fetch tracking JS
	q2 := url.Values{}
	q2.Set("instanceId", CUSTOMER_ID)
	q2.Set("PageId", "si")
	q2.Set("session_id", sessionID)
	req2, _ := http.NewRequest("GET", proxyURL("https://ov-df.microsoft.com/mdt.js?"+q2.Encode()), nil)
	req2.Header.Set("User-Agent", UA)
	setWorkerSecret(req2)
	resp2, err := client.Do(req2)
	if err != nil {
		return sessionID, nil // Proceed anyway
	}
	defer resp2.Body.Close()

	bodyBytes, _ := io.ReadAll(resp2.Body)
	mdtText := string(bodyBytes)

	// Regex to extract w and rticks
	reW := regexp.MustCompile(`[&?]w=([^&"'\s]+)`)
	reRt := regexp.MustCompile(`rticks[="]+\+?\s*(\d{10,})`)

	wMatch := reW.FindStringSubmatch(mdtText)
	rtMatch := reRt.FindStringSubmatch(mdtText)

	// Step 3: Send fingerprint response
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

	return sessionID, nil
}

// Map DownloadType code to string
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

// Background cleanup for stale sessions
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

// Middleware for CORS
func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		// Allowed origins
		if origin == "https://msdl.tech-latest.com" || strings.HasPrefix(origin, "http://localhost:") {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
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

	sessionID, _ := setupSession()

	cacheMutex.Lock()
	sessionCache[productID] = SessionEntry{SessionID: sessionID, CreatedAt: time.Now()}
	cacheMutex.Unlock()

	skuQ := url.Values{}
	skuQ.Set("profile", PROFILE)
	skuQ.Set("productEditionId", productID)
	skuQ.Set("SKU", "undefined")
	skuQ.Set("friendlyFileName", "undefined")
	skuQ.Set("Locale", LOCALE)
	skuQ.Set("sessionID", sessionID)
	reqURL := proxyURL("https://www.microsoft.com/software-download-connector/api/getskuinformationbyproductedition?" + skuQ.Encode())

	client := &http.Client{Timeout: 15 * time.Second}
	var finalData map[string]interface{}

	// Retry logic
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
				respondJSONError(w, http.StatusBadGateway, "Request failed: "+err.Error())
				return
			}
			continue
		}

		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			if attempt == 2 {
				respondJSONError(w, http.StatusBadGateway, fmt.Sprintf("Microsoft API returned HTTP %d: %s", resp.StatusCode, string(bodyBytes)))
				return
			}
			continue
		}

		// Double encoded decode logic
		if len(bodyBytes) > 0 && bodyBytes[0] == '"' {
			var unquoted string
			json.Unmarshal(bodyBytes, &unquoted)
			bodyBytes = []byte(unquoted)
		}

		var data map[string]interface{}
		json.Unmarshal(bodyBytes, &data)

		// Check for specific MS errors
		if errs, ok := data["Errors"].([]interface{}); ok && len(errs) > 0 {
			if attempt == 2 {
				msg := "Microsoft API error"
				if errMap, ok := errs[0].(map[string]interface{}); ok {
					if val, exists := errMap["Value"]; exists {
						msg = val.(string)
					}
				}
				respondJSONError(w, http.StatusBadGateway, msg)
				return
			}
			continue
		}

		finalData = data
		break
	}

	skus, ok := finalData["Skus"].([]interface{})
	if !ok || len(skus) == 0 {
		respondJSONError(w, http.StatusNotFound, "No languages found for this product ID.")
		return
	}

	log.Printf("/skuinfo: product_id=%s -> %d languages (session=%s)\n", productID, len(skus), sessionID[:8])
	json.NewEncoder(w).Encode(finalData)
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

	var sessionID string
	cacheMutex.RLock()
	cached, exists := sessionCache[productID]
	cacheMutex.RUnlock()

	if exists && time.Since(cached.CreatedAt) < SESSION_TTL {
		sessionID = cached.SessionID
		log.Printf("/proxy: Reusing session %s for product_id=%s\n", sessionID[:8], productID)
	} else {
		log.Printf("/proxy: No valid session for product_id=%s, creating new one...\n", productID)
		sessionID, _ = setupSession()

		// Warmup with SKU info request
		warmupQ := url.Values{}
		warmupQ.Set("profile", PROFILE)
		warmupQ.Set("productEditionId", productID)
		warmupQ.Set("SKU", "undefined")
		warmupQ.Set("friendlyFileName", "undefined")
		warmupQ.Set("Locale", LOCALE)
		warmupQ.Set("sessionID", sessionID)
		warmupURL := proxyURL("https://www.microsoft.com/software-download-connector/api/getskuinformationbyproductedition?" + warmupQ.Encode())

		client := &http.Client{Timeout: 10 * time.Second}
		req, _ := http.NewRequest("GET", warmupURL, nil)
		req.Header.Set("User-Agent", UA)
		req.Header.Set("Referer", getReferer(productID))
		req.Header.Set("Accept", "application/json")
		setWorkerSecret(req)
		client.Do(req)

		cacheMutex.Lock()
		sessionCache[productID] = SessionEntry{SessionID: sessionID, CreatedAt: time.Now()}
		cacheMutex.Unlock()
	}

	proxyQ := url.Values{}
	proxyQ.Set("profile", PROFILE)
	proxyQ.Set("productEditionId", "undefined")
	proxyQ.Set("SKU", skuID)
	proxyQ.Set("friendlyFileName", "undefined")
	proxyQ.Set("Locale", LOCALE)
	proxyQ.Set("sessionID", sessionID)
	reqURL := proxyURL("https://www.microsoft.com/software-download-connector/api/GetProductDownloadLinksBySku?" + proxyQ.Encode())

	client := &http.Client{Timeout: 15 * time.Second}
	req, _ := http.NewRequest("GET", reqURL, nil)
	req.Header.Set("User-Agent", UA)
	req.Header.Set("Referer", getReferer(productID))
	req.Header.Set("Accept", "application/json")
	setWorkerSecret(req)

	resp, err := client.Do(req)
	if err != nil {
		respondJSONError(w, http.StatusBadGateway, "Request failed: "+err.Error())
		return
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		respondJSONError(w, http.StatusBadGateway, fmt.Sprintf("Microsoft API returned HTTP %d: %s", resp.StatusCode, string(bodyBytes)))
		return
	}

	// Double encoded decode
	if len(bodyBytes) > 0 && bodyBytes[0] == '"' {
		var unquoted string
		json.Unmarshal(bodyBytes, &unquoted)
		bodyBytes = []byte(unquoted)
	}

	var data map[string]interface{}
	json.Unmarshal(bodyBytes, &data)

	// Check for Microsoft errors
	if errs, ok := data["Errors"].([]interface{}); ok && len(errs) > 0 {
		if errMap, ok := errs[0].(map[string]interface{}); ok {
			if typeNum, exists := errMap["Type"].(float64); exists && typeNum == 9 {
				respondJSONError(w, http.StatusTooManyRequests, "Your IP has been temporarily blocked by Microsoft. Please try again later. (Code 715-123130)")
				return
			}
			if val, exists := errMap["Value"].(string); exists {
				respondJSONError(w, http.StatusBadGateway, val)
				return
			}
		}
		respondJSONError(w, http.StatusBadGateway, "Microsoft API error")
		return
	}

	optsRaw, exists := data["ProductDownloadOptions"]
	if !exists {
		respondJSONError(w, http.StatusNotFound, "No download links found for this SKU.")
		return
	}

	opts, ok := optsRaw.([]interface{})
	if !ok || len(opts) == 0 {
		respondJSONError(w, http.StatusNotFound, "No download links found for this SKU.")
		return
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

	log.Printf("/proxy: product_id=%s, sku_id=%s -> %d links\n", productID, skuID, len(opts))

	// Write pure JSON
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false) // Don't encode & to & in URLs!
	enc.Encode(data)
	w.Write(buf.Bytes())
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

	// Check cache
	evalCacheMu.RLock()
	cached, exists := evalCache[product]
	evalCacheMu.RUnlock()

	if exists && time.Since(cached.CachedAt) < evalCacheTTL {
		log.Printf("/evallinks: product=%s -> cache hit (%d links)\n", product, len(cached.Links))
		json.NewEncoder(w).Encode(EvalLinksResponse{Product: product, Name: evalProduct.Name, Links: cached.Links})
		return
	}

	links, err := fetchEvalLinks(evalProduct.EvalURL)
	if err != nil {
		respondJSONError(w, http.StatusBadGateway, "Failed to fetch eval links: "+err.Error())
		return
	}
	if len(links) == 0 {
		respondJSONError(w, http.StatusNotFound, "No download links found for this eval product")
		return
	}

	evalCacheMu.Lock()
	evalCache[product] = evalCacheEntry{Links: links, CachedAt: time.Now()}
	evalCacheMu.Unlock()

	log.Printf("/evallinks: product=%s -> %d links\n", product, len(links))
	json.NewEncoder(w).Encode(EvalLinksResponse{Product: product, Name: evalProduct.Name, Links: links})
}

// warmEvalCache pre-fetches all eval product links in the background at startup.
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
	go cleanupSessions()
	go warmEvalCache()

	mux := http.NewServeMux()

	// API Routing
	mux.HandleFunc("/skuinfo", handleSkuInfo)
	mux.HandleFunc("/proxy", handleProxy)
	mux.HandleFunc("/evallinks", handleEvalLinks)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Plain root response
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
