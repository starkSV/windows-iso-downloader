package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	msUA       = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	msProfile  = "606624d44113"
	msLocale   = "en-US"
	msOrgID    = "y6jn8c31"
	msCustomer = "560dc9f3-1aa5-4a2f-b63c-9e18f8d0e175"
)

var (
	fwlinkRe  = regexp.MustCompile(`https://go\.microsoft\.com/fwlink/[^"'\s<>]+`)
	isoLangRe = regexp.MustCompile(`_([a-z]{2}-[a-z]{2})\.iso$`)
	reW       = regexp.MustCompile(`[&?]w=([^&"'\s]+)`)
	reRt      = regexp.MustCompile(`rticks[="]+\+?\s*(\d{10,})`)
)

type Language struct {
	ID       string `json:"Id"`
	Language string `json:"Language"`
}

type DownloadLink struct {
	URI          string
	Architecture string
}

type EvalLink struct {
	Arch string
	Lang string
	URL  string
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

func referer(productID string) string {
	id, err := strconv.Atoi(productID)
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

func mapDownloadType(n int) string {
	switch n {
	case 0:
		return "x86"
	case 1:
		return "x64"
	case 2:
		return "ARM64"
	default:
		return fmt.Sprintf("type_%d", n)
	}
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

func msGet(client *http.Client, reqURL, productID string) ([]byte, error) {
	req, _ := http.NewRequest("GET", reqURL, nil)
	req.Header.Set("User-Agent", msUA)
	req.Header.Set("Referer", referer(productID))
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
	return body, nil
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

func parseSkuInfo(raw []byte) ([]Language, error) {
	var data struct {
		Skus   []Language     `json:"Skus"`
		Errors []msErrorEntry `json:"Errors"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
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

func parseDownloadLinks(raw []byte) ([]DownloadLink, error) {
	var data struct {
		ProductDownloadOptions []struct {
			Uri          string      `json:"Uri"`
			Architecture interface{} `json:"Architecture"`
			DownloadType *float64    `json:"DownloadType"`
		} `json:"ProductDownloadOptions"`
		Errors []msErrorEntry `json:"Errors"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, fmt.Errorf("invalid download response: %w", err)
	}
	if msg := firstError(data.Errors); msg != "" {
		return nil, fmt.Errorf("%s", msg)
	}
	if len(data.ProductDownloadOptions) == 0 {
		return nil, fmt.Errorf("no download links found for this SKU")
	}
	var links []DownloadLink
	for _, o := range data.ProductDownloadOptions {
		arch := ""
		if s, ok := o.Architecture.(string); ok && s != "" {
			arch = s
		} else if o.DownloadType != nil {
			arch = mapDownloadType(int(*o.DownloadType))
		}
		links = append(links, DownloadLink{URI: o.Uri, Architecture: arch})
	}
	return links, nil
}

func fetchLanguages(client *http.Client, sessionID, productID string) ([]Language, error) {
	q := url.Values{}
	q.Set("profile", msProfile)
	q.Set("productEditionId", productID)
	q.Set("SKU", "undefined")
	q.Set("friendlyFileName", "undefined")
	q.Set("Locale", msLocale)
	q.Set("sessionID", sessionID)
	raw, err := msGet(client,
		"https://www.microsoft.com/software-download-connector/api/getskuinformationbyproductedition?"+q.Encode(),
		productID)
	if err != nil {
		return nil, err
	}
	return parseSkuInfo(raw)
}

// fetchDownloadLinks returns parsed links and the raw Microsoft JSON (for cache contribution).
func fetchDownloadLinks(client *http.Client, sessionID, productID, skuID string) ([]DownloadLink, []byte, error) {
	wq := url.Values{}
	wq.Set("profile", msProfile)
	wq.Set("productEditionId", productID)
	wq.Set("SKU", "undefined")
	wq.Set("friendlyFileName", "undefined")
	wq.Set("Locale", msLocale)
	wq.Set("sessionID", sessionID)
	msGet(client,
		"https://www.microsoft.com/software-download-connector/api/getskuinformationbyproductedition?"+wq.Encode(),
		productID)

	q := url.Values{}
	q.Set("profile", msProfile)
	q.Set("productEditionId", "undefined")
	q.Set("SKU", skuID)
	q.Set("friendlyFileName", "undefined")
	q.Set("Locale", msLocale)
	q.Set("sessionID", sessionID)
	raw, err := msGet(client,
		"https://www.microsoft.com/software-download-connector/api/GetProductDownloadLinksBySku?"+q.Encode(),
		productID)
	if err != nil {
		return nil, nil, err
	}
	links, err := parseDownloadLinks(raw)
	if err != nil {
		return nil, nil, err
	}
	return links, raw, nil
}

func extractFwlinks(pageHTML string) []string {
	matches := fwlinkRe.FindAllString(pageHTML, -1)
	seen := map[string]bool{}
	var out []string
	for _, m := range matches {
		link := html.UnescapeString(m)
		if !seen[link] {
			seen[link] = true
			out = append(out, link)
		}
	}
	return out
}

func detectArch(rawURL string) string {
	l := strings.ToLower(rawURL)
	switch {
	case strings.Contains(l, "arm64"):
		return "ARM64"
	case strings.Contains(l, "x64"):
		return "x64"
	case strings.Contains(l, "x86"):
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
	client := &http.Client{Timeout: 20 * time.Second}
	req, _ := http.NewRequest("GET", evalURL, nil)
	req.Header.Set("User-Agent", msUA)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching eval page: %w", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	fwlinks := extractFwlinks(string(body))
	var (
		links []EvalLink
		mu    sync.Mutex
		wg    sync.WaitGroup
	)
	for _, fw := range fwlinks {
		wg.Add(1)
		go func(fw string) {
			defer wg.Done()
			r, _ := http.NewRequest("GET", fw, nil)
			r.Header.Set("User-Agent", msUA)
			resp, err := client.Do(r)
			if err != nil {
				return
			}
			resp.Body.Close()
			final := resp.Request.URL.String()
			if !strings.Contains(strings.ToLower(final), ".iso") {
				return
			}
			mu.Lock()
			links = append(links, EvalLink{Arch: detectArch(final), Lang: detectLang(final), URL: final})
			mu.Unlock()
		}(fw)
	}
	wg.Wait()
	if len(links) == 0 {
		return nil, fmt.Errorf("no eval ISO links found")
	}
	return links, nil
}
