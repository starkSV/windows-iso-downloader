package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

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

	if err := fs.Parse(args); err != nil {
		return err
	}
	query := strings.Join(fs.Args(), " ")

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
	return runConsumer(*productID, query, *langFlag)
}

func runConsumer(productID, query, langName string) error {
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
	links, err := fetchDownloadLinks(client, sessionID, product.ID, lang.ID)
	if err != nil {
		return fmt.Errorf("fetching download links: %w", err)
	}

	for _, link := range links {
		fmt.Println(link.URI)
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

	for _, link := range links {
		if link.Arch != "" {
			fmt.Printf("%s\t%s\n", link.Arch, link.URL)
		} else {
			fmt.Println(link.URL)
		}
	}
	return nil
}
