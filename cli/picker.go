package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// parseChoice reads a 1-based integer from r, validated in [1, max].
func parseChoice(r io.Reader, max int) (int, error) {
	scanner := bufio.NewScanner(r)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return 0, fmt.Errorf("reading input: %w", err)
		}
		return 0, fmt.Errorf("no input")
	}
	text := strings.TrimSpace(scanner.Text())
	if text == "" {
		return 0, fmt.Errorf("enter a number between 1 and %d", max)
	}
	n, err := strconv.Atoi(text)
	if err != nil || n < 1 || n > max {
		return 0, fmt.Errorf("enter a number between 1 and %d", max)
	}
	return n, nil
}

func pickProduct(products []Product) (Product, error) {
	if len(products) == 0 {
		return Product{}, fmt.Errorf("no products available")
	}
	fmt.Fprintln(os.Stderr, "\nSelect a product:")
	for i, p := range products {
		fmt.Fprintf(os.Stderr, "  %2d. %s\n", i+1, p.Name)
	}
	fmt.Fprint(os.Stderr, "\nChoice: ")
	n, err := parseChoice(os.Stdin, len(products))
	if err != nil {
		return Product{}, err
	}
	return products[n-1], nil
}

func pickLanguage(langs []Language) (Language, error) {
	if len(langs) == 0 {
		return Language{}, fmt.Errorf("no languages available")
	}
	fmt.Fprintln(os.Stderr, "\nSelect a language:")
	for i, l := range langs {
		fmt.Fprintf(os.Stderr, "  %2d. %s\n", i+1, l.Language)
	}
	fmt.Fprint(os.Stderr, "\nChoice: ")
	n, err := parseChoice(os.Stdin, len(langs))
	if err != nil {
		return Language{}, err
	}
	return langs[n-1], nil
}

func pickEvalProduct(products []EvalProduct) (EvalProduct, error) {
	if len(products) == 0 {
		return EvalProduct{}, fmt.Errorf("no eval products available")
	}
	fmt.Fprintln(os.Stderr, "\nSelect an evaluation product:")
	for i, p := range products {
		fmt.Fprintf(os.Stderr, "  %2d. %s\n", i+1, p.Name)
	}
	fmt.Fprint(os.Stderr, "\nChoice: ")
	n, err := parseChoice(os.Stdin, len(products))
	if err != nil {
		return EvalProduct{}, err
	}
	return products[n-1], nil
}
