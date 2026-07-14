// check-new-releases scrapes Microsoft's public Windows 11 download page for
// the current "flagship" product edition ID and compares it against the
// catalog already tracked in cli/catalog.go. Windows 8.1 and Windows 10 are
// intentionally not checked here -- 8.1 is fully frozen and Windows 10 is
// past end-of-life, so neither will ever produce a new consumer ISO release.
//
// If a new flagship ID is found, it probes a bounded range of adjacent IDs to
// discover the accompanying variant family (Home China, Pro China, ARM64,
// ...), since Microsoft allocates those in a cluster near the flagship ID
// but the exact offset isn't consistent release to release (observed gaps:
// +3 for the 25H2 family, +18 for 24H2's ARM64 family).
//
// This tool only *discovers* candidate IDs -- it does not touch cli/catalog.go,
// products.json, or validContributeProducts automatically. Naming a new entry
// correctly (build number, "Updated Oct"-style qualifiers, etc.) still needs a
// human checking Microsoft's own release-health pages.
//
// Usage: go run . [-catalog path/to/cli/catalog.go] [-probe-range N]
package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"sync"
	"time"
)

// --- Copied from cli/microsoft.go (not imported -- cli/ is package main and
// can't be imported from elsewhere). This is a standalone maintenance tool,
// not part of the shipped CLI/backend; keep in sync manually if the real
// session flow ever changes. ---

const (
	msUA       = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	msProfile  = "606624d44113"
	msLocale   = "en-US"
	msOrgID    = "y6jn8c31"
	msCustomer = "560dc9f3-1aa5-4a2f-b63c-9e18f8d0e175"
)

var (
	reW  = regexp.MustCompile(`[&?]w=([^&"'\s]+)`)
	reRt = regexp.MustCompile(`rticks[="]+\+?\s*(\d{10,})`)
)

type skuLang struct {
	ID                 string `json:"Id"`
	Language           string `json:"Language"`
	ProductDisplayName string `json:"ProductDisplayName"`
}

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

func newSessionID() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hex.EncodeToString(b[0:4]), hex.EncodeToString(b[4:6]),
		hex.EncodeToString(b[6:8]), hex.EncodeToString(b[8:10]),
		hex.EncodeToString(b[10:16]))
}

func newSession() (*http.Client, string) {
	sessionID := newSessionID()
	jar := &simpleCookieJar{}
	client := &http.Client{Timeout: 15 * time.Second, Jar: jar}

	q1 := url.Values{}
	q1.Set("org_id", msOrgID)
	q1.Set("session_id", sessionID)
	req1, _ := http.NewRequest("GET", "https://vlscppe.microsoft.com/tags?"+q1.Encode(), nil)
	req1.Header.Set("User-Agent", msUA)
	client.Do(req1)

	q2 := url.Values{}
	q2.Set("instanceId", msCustomer)
	q2.Set("PageId", "si")
	q2.Set("session_id", sessionID)
	req2, _ := http.NewRequest("GET", "https://ov-df.microsoft.com/mdt.js?"+q2.Encode(), nil)
	req2.Header.Set("User-Agent", msUA)
	resp2, err := client.Do(req2)
	if err != nil {
		return client, sessionID
	}
	body, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()

	wMatch := reW.FindStringSubmatch(string(body))
	rtMatch := reRt.FindStringSubmatch(string(body))
	if len(wMatch) > 1 && len(rtMatch) > 1 {
		q3 := url.Values{}
		q3.Set("session_id", sessionID)
		q3.Set("CustomerId", msCustomer)
		q3.Set("PageId", "si")
		q3.Set("w", wMatch[1])
		q3.Set("mdt", fmt.Sprintf("%d", time.Now().UnixMilli()))
		q3.Set("rticks", rtMatch[1])
		req3, _ := http.NewRequest("GET", "https://ov-df.microsoft.com/?"+q3.Encode(), nil)
		req3.Header.Set("User-Agent", msUA)
		client.Do(req3)
	}
	return client, sessionID
}

type msErrorEntry struct {
	Type  float64 `json:"Type"`
	Value string  `json:"Value"`
}

func firstError(errs []msErrorEntry) string {
	if len(errs) == 0 {
		return ""
	}
	if int(errs[0].Type) == 9 {
		if errs[0].Value != "" {
			return errs[0].Value
		}
		return "Your IP has been temporarily blocked by Microsoft (Code 715-123130)"
	}
	if errs[0].Value != "" {
		return errs[0].Value
	}
	return "Microsoft API error"
}

func fetchLanguages(client *http.Client, sessionID, productID string) ([]skuLang, error) {
	q := url.Values{}
	q.Set("profile", msProfile)
	q.Set("productEditionId", productID)
	q.Set("SKU", "undefined")
	q.Set("friendlyFileName", "undefined")
	q.Set("Locale", msLocale)
	q.Set("sessionID", sessionID)

	req, _ := http.NewRequest("GET", "https://www.microsoft.com/software-download-connector/api/getskuinformationbyproductedition?"+q.Encode(), nil)
	req.Header.Set("User-Agent", msUA)
	req.Header.Set("Referer", "https://www.microsoft.com/en-us/software-download/windows11")
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Microsoft returned HTTP %d", resp.StatusCode)
	}
	if len(body) > 0 && body[0] == '"' {
		var unquoted string
		json.Unmarshal(body, &unquoted)
		body = []byte(unquoted)
	}

	var data struct {
		Skus   []skuLang      `json:"Skus"`
		Errors []msErrorEntry `json:"Errors"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("invalid SKU response: %w", err)
	}
	if msg := firstError(data.Errors); msg != "" {
		return nil, fmt.Errorf("%s", msg)
	}
	if len(data.Skus) == 0 {
		return nil, fmt.Errorf("no languages found for this product")
	}
	return data.Skus, nil
}

// --- New logic specific to this tool ---

var catalogEntryRe = regexp.MustCompile(`\{"(\d+)",\s*"([^"]*)"\}`)

func knownCatalogIDs(catalogPath string) (map[string]string, error) {
	data, err := os.ReadFile(catalogPath)
	if err != nil {
		return nil, err
	}
	matches := catalogEntryRe.FindAllStringSubmatch(string(data), -1)
	out := make(map[string]string, len(matches))
	for _, m := range matches {
		out[m[1]] = m[2]
	}
	return out, nil
}

var flagshipRe = regexp.MustCompile(`<option value="(\d+)">Windows`)

func scrapeFlagshipID(pageURL string) (string, error) {
	req, _ := http.NewRequest("GET", pageURL, nil)
	req.Header.Set("User-Agent", msUA)
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching download page: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	m := flagshipRe.FindStringSubmatch(string(body))
	if len(m) < 2 {
		return "", fmt.Errorf("could not find a product edition ID on the page -- Microsoft may have changed the page layout")
	}
	return m[1], nil
}

func main() {
	catalogPath := flag.String("catalog", "../../cli/catalog.go", "path to cli/catalog.go")
	probeRange := flag.Int("probe-range", 20, "how many adjacent IDs to probe after a new flagship is found")
	flag.Parse()

	const windows11Page = "https://www.microsoft.com/en-us/software-download/windows11"
	fmt.Printf("Checking %s for the current flagship product edition ID...\n", windows11Page)

	flagshipID, err := scrapeFlagshipID(windows11Page)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	fmt.Printf("Flagship ID found: %s\n", flagshipID)

	known, err := knownCatalogIDs(*catalogPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error reading catalog:", err)
		os.Exit(1)
	}
	fmt.Printf("Loaded %d known product IDs from %s\n\n", len(known), *catalogPath)

	if name, ok := known[flagshipID]; ok {
		fmt.Printf("Up to date -- flagship ID %s is already in the catalog as %q\n", flagshipID, name)
		return
	}

	fmt.Printf("NEW RELEASE DETECTED -- product edition ID %s is not in the current catalog.\n\n", flagshipID)
	fmt.Println("Probing adjacent IDs to discover the variant family (this makes real")
	fmt.Println("requests to Microsoft -- allow it to finish rather than re-running):")
	fmt.Println()

	client, sessionID := newSession()
	idNum, err := strconv.Atoi(flagshipID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: flagship ID is not numeric:", err)
		os.Exit(1)
	}

	fmt.Printf("%-8s %-8s %s\n", "ID", "STATUS", "NOTES")
	for offset := 0; offset <= *probeRange; offset++ {
		candidate := strconv.Itoa(idNum + offset)
		langs, err := fetchLanguages(client, sessionID, candidate)
		time.Sleep(300 * time.Millisecond) // be a good citizen -- don't hammer Microsoft
		if err != nil {
			continue // not a valid product ID; skip silently, only report hits
		}
		realName := ""
		if len(langs) > 0 {
			realName = langs[0].ProductDisplayName
		}
		note := fmt.Sprintf("%q -- %d language(s)", realName, len(langs))
		switch {
		case len(langs) == 1:
			note += " -- likely a China-only single-edition variant"
		case len(langs) > 30:
			note += " -- likely a worldwide multi-edition variant"
		}
		if existingName, ok := known[candidate]; ok {
			note += fmt.Sprintf(" (already in catalog as %q)", existingName)
		}
		fmt.Printf("%-8s %-8s %s\n", candidate, "FOUND", note)
	}

	fmt.Println()
	fmt.Println("Note: the ARM64 family's offset from the flagship isn't consistent release")
	fmt.Println("to release (seen +3 for 25H2, +18 for 24H2) -- if no ARM64 variant showed up")
	fmt.Println("above, try a wider -probe-range.")
	fmt.Println()
	fmt.Println("Next steps: confirm the build number from Microsoft's release-health pages,")
	fmt.Println("then add entries to cli/catalog.go, frontend/public/data/products.json,")
	fmt.Println("and the validContributeProducts map in backend/main.go.")
}
