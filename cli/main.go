package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

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

func contributeURL() string {
	if u := os.Getenv("MSDL_API_URL"); u != "" {
		return strings.TrimRight(u, "/") + "/contribute"
	}
	return "https://api.msdl.tech-latest.com/contribute"
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
  msdl --eval [slug]                            evaluation ISOs (server-2025, win11-ent, ...)
  msdl --list                                   list all products and exit

Flags:`)
		fs.PrintDefaults()
	}

	productID := fs.String("id", "", "consumer product ID (skips product picker)")
	langFlag := fs.String("lang", "", `language name, e.g. "English (United States)"`)
	evalMode := fs.Bool("eval", false, "fetch evaluation ISOs")
	listMode := fs.Bool("list", false, "list all products and exit")
	noContributeFlag := fs.Bool("no-contribute", false, "skip sharing the link with the msdl.tech cache")

	if err := fs.Parse(args); err != nil {
		return err
	}
	query := strings.Join(fs.Args(), " ")
	noContribute := *noContributeFlag || os.Getenv("MSDL_NO_CONTRIBUTE") == "1"

	if *listMode {
		fmt.Fprintln(os.Stderr, "Consumer products:")
		for _, p := range consumerProducts {
			fmt.Fprintf(os.Stderr, "  %-6s %s\n", p.ID, p.Name)
		}
		fmt.Fprintln(os.Stderr, "\nEvaluation products:")
		for _, p := range evalProducts {
			fmt.Fprintf(os.Stderr, "  %-20s %s\n", p.Slug, p.Name)
		}
		return nil
	}

	if *evalMode {
		return runEval(query)
	}
	return runConsumer(*productID, query, *langFlag, noContribute)
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
			return fmt.Errorf("language %q not available for %s", langName, product.Name)
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
	for _, link := range links {
		name := urlFilename(link.URI)
		fmt.Fprintf(os.Stderr, "  %s\n", name)
		fmt.Fprintf(os.Stderr, "  %s\n", strings.Repeat("─", len(name)))
		fmt.Println(link.URI) // stdout — stays clean for piping
		fmt.Fprintln(os.Stderr, "")
	}

	if !noContribute {
		fmt.Fprintln(os.Stderr, "  ✓ Shared with MSDL Web App (https://msdl.tech-latest.com/)")
		fmt.Fprintln(os.Stderr, "    Opt out: --no-contribute")
		fmt.Fprintln(os.Stderr, "")
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
	fmt.Println(links[n-1].URL)
	return nil
}
