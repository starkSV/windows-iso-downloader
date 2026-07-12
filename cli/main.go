package main

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

// Version is injected at build time via -ldflags "-X main.Version=0.3.0".
// Falls back to "dev" for local builds.
var Version = "dev"

// contributeURL is the MSDL backend endpoint that warms the shared link cache.
// Override with MSDL_API_URL env var, e.g. export MSDL_API_URL=http://localhost:3002
const contributeSecret = "msdl-contribute-2026"

// urlFilename extracts the bare filename from a URL (strips query string and path).
func urlFilename(rawURL string) string {
	s := rawURL
	if i := strings.LastIndex(s, "?"); i > 0 {
		s = s[:i]
	}
	if i := strings.LastIndex(s, "/"); i >= 0 {
		return s[i+1:]
	}
	return s
}

// apiBaseURL returns the backend base URL, with MSDL_API_URL override.
func apiBaseURL() string {
	if u := os.Getenv("MSDL_API_URL"); u != "" {
		return strings.TrimRight(u, "/")
	}
	return "https://api.msdl.tech-latest.com"
}

func contributeURL() string {
	return apiBaseURL() + "/contribute"
}

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

type cliTelemetryPayload struct {
	Action    string `json:"action"`
	ProductID string `json:"product_id,omitempty"`
	EvalSlug  string `json:"eval_slug,omitempty"`
	Platform  string `json:"platform"`
	Version   string `json:"version"`
	Success   bool   `json:"success"`
	Error     string `json:"error,omitempty"`
}

// truncateError caps an error string for telemetry (avoids leaking huge bodies,
// keeps Redis field values small). Keeps both head and tail: the head usually
// names what failed (e.g. "language X not available for Y"), while wrapped
// errors put the root cause last (e.g. "fetching X: <root cause>") — a long
// middle section (URLs, option lists) is the least useful part to keep.
func truncateError(err error) string {
	if err == nil {
		return ""
	}
	s := err.Error()
	const max = 150
	if len(s) <= max {
		return s
	}
	const headLen = 60
	const sep = " ... "
	tailLen := max - headLen - len(sep)
	return s[:headLen] + sep + s[len(s)-tailLen:]
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

func postContribute(productID, skuID string, rawJSON []byte) {
	payload := struct {
		ProductID string          `json:"product_id"`
		SkuID     string          `json:"sku_id"`
		RawJSON   json.RawMessage `json:"raw_json"`
	}{ProductID: productID, SkuID: skuID, RawJSON: rawJSON}
	body, _ := json.Marshal(payload)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, contributeURL(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Contribute-Secret", contributeSecret)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	resp.Body.Close()
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("msdl", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `msdl — Windows ISO downloader

Usage:
  msdl [search terms]                           interactive: filter + pick product, pick language
  msdl --id 3262                                skip product picker
  msdl --id 3262 --lang "English (United States)"  no prompts, print URL directly
                                                 (language names vary by product — omit --lang
                                                 or check the error's "available" list to see them)
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
			return nil // fs.Usage already printed
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

	telAction := "interactive"
	telProductID := ""
	telEvalSlug := ""

	var err error
	switch {
	case *evalMode:
		telAction = "eval"
		telEvalSlug = query
		err = runEval(query)
	case *productID != "":
		telAction = "fetch"
		telProductID = *productID
		err = runConsumer(*productID, query, *langFlag, noContribute)
	case query == "" && isTerminal():
		// Pure interactive: combined consumer + eval picker
		isEval, product, ep, pickErr := pickCombined()
		if pickErr != nil {
			return pickErr
		}
		if isEval {
			telAction = "eval"
			telEvalSlug = ep.Slug
			err = runEval(ep.Slug)
		} else {
			telAction = "fetch"
			telProductID = product.ID
			err = runConsumer(product.ID, "", *langFlag, noContribute)
		}
	default:
		err = runConsumer(*productID, query, *langFlag, noContribute)
	}

	if !noTelemetry {
		sendTelemetry(cliTelemetryPayload{
			Action:    telAction,
			ProductID: telProductID,
			EvalSlug:  telEvalSlug,
			Platform:  runtime.GOOS,
			Version:   Version,
			Success:   err == nil,
			Error:     truncateError(err),
		})
	}

	return err
}

func runConsumer(productID, query, langName string, noContribute bool) error {
	var product Product

	if productID != "" {
		p, ok := findProductByID(productID)
		if !ok {
			return fmt.Errorf("unknown product ID %q — run msdl --list to see all products", productID)
		}
		product = p
	} else {
		candidates := consumerProducts
		if query != "" {
			candidates = searchProducts(query)
			if len(candidates) == 0 {
				return fmt.Errorf("no products match %q — run msdl --list to see all products", query)
			}
		}
		if len(candidates) == 1 {
			product = candidates[0]
			fmt.Fprintf(os.Stderr, "Selected: %s\n", product.Name)
		} else {
			var err error
			product, err = pickProduct(candidates)
			if err != nil {
				return err
			}
		}
	}

	fmt.Fprintf(os.Stderr, "Setting up Microsoft session...\n")
	client, sessionID := newSession()

	langs, err := fetchLanguages(client, sessionID, product.ID)
	if err != nil {
		return fmt.Errorf("fetching languages for %s: %w", product.Name, err)
	}

	var lang Language
	if langName != "" {
		for _, l := range langs {
			if strings.EqualFold(l.Language, langName) {
				lang = l
				break
			}
		}
		if lang.ID == "" {
			names := make([]string, len(langs))
			for i, l := range langs {
				names[i] = l.Language
			}
			return fmt.Errorf("language %q not available for %s — available: %s", langName, product.Name, strings.Join(names, ", "))
		}
	} else {
		lang, err = pickLanguage(langs)
		if err != nil {
			return err
		}
	}

	fmt.Fprintf(os.Stderr, "Fetching download link...\n")
	links, rawJSON, err := fetchDownloadLinks(client, sessionID, product.ID, lang.ID)
	if err != nil {
		return fmt.Errorf("fetching download links: %w", err)
	}

	// Fire contribute in background before printing so the goroutine starts immediately.
	var wg sync.WaitGroup
	if !noContribute {
		wg.Add(1)
		go func() {
			defer wg.Done()
			postContribute(product.ID, lang.ID, rawJSON)
		}()
	}

	fmt.Fprintln(os.Stderr, "")

	var selectedURI string
	if isTerminal() && len(links) > 1 {
		link, err := pickArchitecture(links)
		if err != nil {
			return err
		}
		name := urlFilename(link.URI)
		fmt.Fprintf(os.Stderr, "\n  %s\n", name)
		fmt.Fprintf(os.Stderr, "  %s\n\n", strings.Repeat("─", len(name)))
		fmt.Println(link.URI)
		selectedURI = link.URI
	} else {
		for _, link := range links {
			name := urlFilename(link.URI)
			fmt.Fprintf(os.Stderr, "  %s\n", name)
			fmt.Fprintf(os.Stderr, "  %s\n", strings.Repeat("─", len(name)))
			fmt.Println(link.URI) // stdout — stays clean for piping
			fmt.Fprintln(os.Stderr, "")
		}
		if len(links) == 1 {
			selectedURI = links[0].URI
		}
	}

	if !noContribute {
		fmt.Fprintln(os.Stderr, "  ✓ Shared with MSDL Web App (https://msdl.tech-latest.com/)")
		fmt.Fprintln(os.Stderr, "    Opt out: --no-contribute")
		fmt.Fprintln(os.Stderr, "")
	}

	if isTerminal() && selectedURI != "" {
		postFetchMenu(selectedURI)
	}

	if !noContribute {
		// Wait up to 5s for the contribution to complete before process exit.
		done := make(chan struct{})
		go func() { wg.Wait(); close(done) }()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
		}
	}
	return nil
}

func runEval(slug string) error {
	var ep EvalProduct

	if slug != "" {
		p, ok := findEvalProduct(slug)
		if !ok {
			return fmt.Errorf("unknown eval product %q — valid slugs: server-2025, server-2022, server-2019, server-2016, win11-ent", slug)
		}
		ep = p
	} else {
		var err error
		ep, err = pickEvalProduct(evalProducts)
		if err != nil {
			return err
		}
	}

	fmt.Fprintf(os.Stderr, "Fetching eval links for %s...\n", ep.Name)
	links, err := fetchEvalLinks(ep.EvalURL)
	if err != nil {
		return fmt.Errorf("fetching eval links for %s: %w", ep.Name, err)
	}
	if len(links) == 0 {
		return fmt.Errorf("no eval links returned for %s", ep.Slug)
	}

	if len(links) == 1 {
		fmt.Println(links[0].URL)
		if isTerminal() {
			postFetchMenu(links[0].URL)
		}
		return nil
	}

	fmt.Fprintln(os.Stderr, "\nSelect a download:")
	for i, link := range links {
		label := link.Lang
		if link.Arch != "" && link.Lang != "" {
			label = fmt.Sprintf("%-6s %s", link.Lang, link.Arch)
		} else if link.Arch != "" {
			label = link.Arch
		}
		fmt.Fprintf(os.Stderr, "  %2d. %s\n", i+1, label)
	}
	fmt.Fprint(os.Stderr, "\nChoice: ")
	n, err := parseChoice(os.Stdin, len(links))
	if err != nil {
		return err
	}
	selected := links[n-1].URL
	fmt.Println(selected)
	if isTerminal() {
		postFetchMenu(selected)
	}
	return nil
}
