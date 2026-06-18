package main

import "testing"

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
