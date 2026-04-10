package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
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
)

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
	req1, _ := http.NewRequest("GET", fmt.Sprintf("https://vlscppe.microsoft.com/tags?org_id=%s&session_id=%s", ORG_ID, sessionID), nil)
	req1.Header.Set("User-Agent", UA)
	client.Do(req1) // Ignore errors intentionally

	// Step 2: Fetch tracking JS
	req2, _ := http.NewRequest("GET", fmt.Sprintf("https://ov-df.microsoft.com/mdt.js?instanceId=%s&PageId=si&session_id=%s", CUSTOMER_ID, sessionID), nil)
	req2.Header.Set("User-Agent", UA)
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

		fpURL := fmt.Sprintf("https://ov-df.microsoft.com/?session_id=%s&CustomerId=%s&PageId=si&w=%s&mdt=%d&rticks=%s",
			sessionID, CUSTOMER_ID, wVal, mdt, rtVal)

		req3, _ := http.NewRequest("GET", fpURL, nil)
		req3.Header.Set("User-Agent", UA)
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
		if origin == "https://msdl.tech-latest.com" || origin == "http://localhost:5173" || origin == "http://localhost:3000" {
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

	sessionID, _ := setupSession()

	cacheMutex.Lock()
	sessionCache[productID] = SessionEntry{SessionID: sessionID, CreatedAt: time.Now()}
	cacheMutex.Unlock()

	reqURL := fmt.Sprintf("https://www.microsoft.com/software-download-connector/api/getskuinformationbyproductedition?profile=%s&productEditionId=%s&SKU=undefined&friendlyFileName=undefined&Locale=%s&sessionID=%s",
		PROFILE, productID, LOCALE, sessionID)

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
				respondJSONError(w, http.StatusBadGateway, fmt.Sprintf("Microsoft API returned HTTP %d", resp.StatusCode))
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
		warmupURL := fmt.Sprintf("https://www.microsoft.com/software-download-connector/api/getskuinformationbyproductedition?profile=%s&productEditionId=%s&SKU=undefined&friendlyFileName=undefined&Locale=%s&sessionID=%s",
			PROFILE, productID, LOCALE, sessionID)

		client := &http.Client{Timeout: 10 * time.Second}
		req, _ := http.NewRequest("GET", warmupURL, nil)
		req.Header.Set("User-Agent", UA)
		req.Header.Set("Referer", getReferer(productID))
		req.Header.Set("Accept", "application/json")
		client.Do(req)

		cacheMutex.Lock()
		sessionCache[productID] = SessionEntry{SessionID: sessionID, CreatedAt: time.Now()}
		cacheMutex.Unlock()
	}

	reqURL := fmt.Sprintf("https://www.microsoft.com/software-download-connector/api/GetProductDownloadLinksBySku?profile=%s&productEditionId=undefined&SKU=%s&friendlyFileName=undefined&Locale=%s&sessionID=%s",
		PROFILE, skuID, LOCALE, sessionID)

	client := &http.Client{Timeout: 15 * time.Second}
	req, _ := http.NewRequest("GET", reqURL, nil)
	req.Header.Set("User-Agent", UA)
	req.Header.Set("Referer", getReferer(productID))
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		respondJSONError(w, http.StatusBadGateway, "Request failed: "+err.Error())
		return
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		respondJSONError(w, http.StatusBadGateway, fmt.Sprintf("Microsoft API returned HTTP %d", resp.StatusCode))
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
	enc.SetEscapeHTML(false) // Don't encode & to \u0026 in URLs!
	enc.Encode(data)
	w.Write(buf.Bytes())
}

func main() {
	go cleanupSessions()

	mux := http.NewServeMux()

	// API Routing
	mux.HandleFunc("/skuinfo", handleSkuInfo)
	mux.HandleFunc("/proxy", handleProxy)
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
