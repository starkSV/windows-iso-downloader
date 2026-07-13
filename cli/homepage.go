package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// homepageFeaturedIDs are the consumer products shown on the bare `msdl`
// landing screen, picked from real usage telemetry rather than the full
// catalog. The full catalog is still one step away via "list".
var homepageFeaturedIDs = []string{"3262", "2618", "3321", "3113", "3265"}

// homepageEvalSlugs are the eval products shown on the landing screen.
var homepageEvalSlugs = []string{"server-2025", "win11-ent"}

type homepageKind int

const (
	homepageKindProduct homepageKind = iota
	homepageKindEval
	homepageKindList
	homepageKindSearch
)

type homepageChoice struct {
	kind      homepageKind
	productID string
	evalSlug  string
	query     string
}

const homepageRule = "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

// showHomepage prints the landing screen for a bare `msdl` invocation and
// reads the user's choice: a number, "list" for the full catalog, or free
// text to search by name.
func showHomepage(latestVersion string) (homepageChoice, error) {
	fmt.Fprintln(os.Stderr, homepageRule)
	fmt.Fprintln(os.Stderr, "msdl — Windows ISO Downloader")
	fmt.Fprintln(os.Stderr, "Direct links from Microsoft's CDN. No browser, no Media Creation Tool.")
	fmt.Fprintln(os.Stderr, homepageRule)
	fmt.Fprintln(os.Stderr)

	fmt.Fprintln(os.Stderr, "Popular:")
	featured := make([]Product, 0, len(homepageFeaturedIDs))
	for _, id := range homepageFeaturedIDs {
		if p, ok := findProductByID(id); ok {
			featured = append(featured, p)
		}
	}
	for i, p := range featured {
		fmt.Fprintf(os.Stderr, "%d. %s\n", i+1, p.Name)
	}
	fmt.Fprintln(os.Stderr)

	fmt.Fprintln(os.Stderr, "Evaluation / Enterprise:")
	evalShortlist := make([]EvalProduct, 0, len(homepageEvalSlugs))
	for _, slug := range homepageEvalSlugs {
		if ep, ok := findEvalProduct(slug); ok {
			evalShortlist = append(evalShortlist, ep)
		}
	}
	for i, ep := range evalShortlist {
		fmt.Fprintf(os.Stderr, "%d. %s\n", len(featured)+i+1, ep.Name)
	}
	fmt.Fprintln(os.Stderr)

	fmt.Fprintln(os.Stderr, `Commands: --list (full catalog) · --help`)
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, homepageRule)
	if latestVersion != "" && latestVersion != Version && Version != "dev" {
		fmt.Fprintf(os.Stderr, "Version %s  (Update Available  %s)  - a TechLatest Open-source Contribution\n", Version, latestVersion)
	} else {
		fmt.Fprintf(os.Stderr, "Version %s  - a TechLatest Open-source Contribution\n", Version)
	}
	fmt.Fprintln(os.Stderr)

	max := len(featured) + len(evalShortlist)
	fmt.Fprintf(os.Stderr, `Choice [1-%d], search by name, or "list": `, max)

	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return homepageChoice{}, fmt.Errorf("reading input: %w", err)
		}
		return homepageChoice{}, fmt.Errorf("no input")
	}
	return parseHomepageInput(scanner.Text(), featured, evalShortlist)
}

// parseHomepageInput interprets the raw input line from the homepage prompt.
func parseHomepageInput(input string, featured []Product, evalShortlist []EvalProduct) (homepageChoice, error) {
	text := strings.TrimSpace(input)
	if text == "" {
		return homepageChoice{}, fmt.Errorf(`enter a choice, search term, or "list"`)
	}
	if strings.EqualFold(text, "list") {
		return homepageChoice{kind: homepageKindList}, nil
	}
	if n, err := strconv.Atoi(text); err == nil {
		max := len(featured) + len(evalShortlist)
		if n < 1 || n > max {
			return homepageChoice{}, fmt.Errorf(`enter a number between 1 and %d, a search term, or "list"`, max)
		}
		if n <= len(featured) {
			return homepageChoice{kind: homepageKindProduct, productID: featured[n-1].ID}, nil
		}
		return homepageChoice{kind: homepageKindEval, evalSlug: evalShortlist[n-len(featured)-1].Slug}, nil
	}
	return homepageChoice{kind: homepageKindSearch, query: text}, nil
}
