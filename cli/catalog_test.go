package main

import (
	"strings"
	"testing"
)

func TestFindProductByID_found(t *testing.T) {
	p, ok := findProductByID("3262")
	if !ok {
		t.Fatal("expected product 3262 to exist")
	}
	if p.ID != "3262" {
		t.Errorf("got ID %s, want 3262", p.ID)
	}
}

func TestFindProductByID_notFound(t *testing.T) {
	_, ok := findProductByID("9999")
	if ok {
		t.Fatal("expected product 9999 to not exist")
	}
}

func TestSearchProducts_match(t *testing.T) {
	results := searchProducts("windows 11 25h2")
	if len(results) == 0 {
		t.Fatal("expected at least one result for 'windows 11 25h2'")
	}
}

func TestSearchProducts_noMatch(t *testing.T) {
	results := searchProducts("zzznomatch")
	if len(results) != 0 {
		t.Errorf("expected no results, got %d", len(results))
	}
}

func TestSearchProducts_caseInsensitive(t *testing.T) {
	results := searchProducts("WINDOWS 11")
	if len(results) == 0 {
		t.Fatal("expected results for uppercase query")
	}
}

func TestSearchProducts_noDigitFragmentCollision(t *testing.T) {
	// "10" must not match inside the "26100.1742" build number of Windows 11
	// 24H2/25H2 products -- regression test for the reported "windows 10"
	// bug that returned Windows 11 results.
	results := searchProducts("windows 10")
	for _, p := range results {
		if strings.Contains(p.Name, "11") {
			t.Errorf("query %q matched %q, a Windows 11 product", "windows 10", p.Name)
		}
	}
	if len(results) == 0 {
		t.Fatal("expected windows 10 to match the actual Windows 10 products")
	}
}

func TestSearchProducts_prefixSubstringStillWorks(t *testing.T) {
	// "arm" should still fuzzy-match "ARM64" -- the word-boundary fix must
	// only reject matches that start mid-digit-run, not all substrings.
	results := searchProducts("arm")
	if len(results) == 0 {
		t.Fatal("expected arm to match ARM64 products")
	}
	for _, p := range results {
		if !strings.Contains(strings.ToLower(p.Name), "arm") {
			t.Errorf("unexpected match %q for query %q", p.Name, "arm")
		}
	}
}

func TestFindEvalProduct_found(t *testing.T) {
	p, ok := findEvalProduct("server-2025")
	if !ok {
		t.Fatal("expected server-2025 to exist")
	}
	if p.Slug != "server-2025" {
		t.Errorf("got slug %s, want server-2025", p.Slug)
	}
}

func TestFindEvalProduct_notFound(t *testing.T) {
	_, ok := findEvalProduct("zzz-nope")
	if ok {
		t.Fatal("expected zzz-nope to not exist")
	}
}
