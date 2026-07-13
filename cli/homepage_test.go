package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func testFeatured() []Product {
	return []Product{
		{ID: "3262", Name: "Windows 11 25H2 (26200.6584)"},
		{ID: "2618", Name: "Windows 10 22H2 v1 (19045.2965)"},
	}
}

func testEvalShortlist() []EvalProduct {
	return []EvalProduct{
		{Slug: "server-2025", Name: "Windows Server 2025"},
		{Slug: "win11-ent", Name: "Windows 11 Enterprise"},
	}
}

func TestParseHomepageInput_product(t *testing.T) {
	choice, err := parseHomepageInput("1", testFeatured(), testEvalShortlist())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if choice.kind != homepageKindProduct || choice.productID != "3262" {
		t.Errorf("got %+v, want product 3262", choice)
	}
}

func TestParseHomepageInput_eval(t *testing.T) {
	choice, err := parseHomepageInput("3", testFeatured(), testEvalShortlist())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if choice.kind != homepageKindEval || choice.evalSlug != "server-2025" {
		t.Errorf("got %+v, want eval server-2025", choice)
	}
}

func TestParseHomepageInput_evalSecond(t *testing.T) {
	choice, err := parseHomepageInput("4", testFeatured(), testEvalShortlist())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if choice.kind != homepageKindEval || choice.evalSlug != "win11-ent" {
		t.Errorf("got %+v, want eval win11-ent", choice)
	}
}

func TestParseHomepageInput_outOfRange(t *testing.T) {
	_, err := parseHomepageInput("5", testFeatured(), testEvalShortlist())
	if err == nil {
		t.Fatal("expected error for out-of-range choice")
	}
}

func TestParseHomepageInput_listCaseInsensitive(t *testing.T) {
	for _, s := range []string{"list", "List", "LIST"} {
		choice, err := parseHomepageInput(s, testFeatured(), testEvalShortlist())
		if err != nil {
			t.Fatalf("unexpected error for %q: %v", s, err)
		}
		if choice.kind != homepageKindList {
			t.Errorf("input %q: got kind %v, want list", s, choice.kind)
		}
	}
}

func TestParseHomepageInput_searchFallback(t *testing.T) {
	choice, err := parseHomepageInput("windows 10", testFeatured(), testEvalShortlist())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if choice.kind != homepageKindSearch || choice.query != "windows 10" {
		t.Errorf("got %+v, want search query \"windows 10\"", choice)
	}
}

func TestParseHomepageInput_emptyInput(t *testing.T) {
	_, err := parseHomepageInput("   ", testFeatured(), testEvalShortlist())
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}

// captureHomepage runs showHomepage with stdin/stderr redirected to pipes so
// the rendered output and the parsed choice can both be inspected without a
// real terminal.
func captureHomepage(t *testing.T, stdin string, latestVersion string) (homepageChoice, string) {
	t.Helper()
	oldStdin, oldStderr := os.Stdin, os.Stderr
	defer func() { os.Stdin, os.Stderr = oldStdin, oldStderr }()

	inR, inW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdin = inR
	inW.WriteString(stdin)
	inW.Close()

	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = outW

	choice, hpErr := showHomepage(latestVersion)
	outW.Close()

	var buf bytes.Buffer
	io.Copy(&buf, outR)
	if hpErr != nil {
		t.Fatalf("unexpected error: %v", hpErr)
	}
	return choice, buf.String()
}

func TestShowHomepage_rendersFeaturedAndEval(t *testing.T) {
	choice, output := captureHomepage(t, "1\n", "")
	if choice.kind != homepageKindProduct || choice.productID != "3262" {
		t.Errorf("got %+v, want product 3262", choice)
	}
	for _, want := range []string{
		"1. Windows 11 25H2 (26200.6584)",
		"2. Windows 10 22H2 v1 (19045.2965)",
		"6. Windows Server 2025",
		"7. Windows 11 Enterprise",
		`Commands: --list (full catalog) · --help`,
		`Choice [1-7], search by name, or "list":`,
	} {
		if !strings.Contains(output, want) {
			t.Errorf("output missing %q, got:\n%s", want, output)
		}
	}
}

func TestShowHomepage_updateAvailableFooter(t *testing.T) {
	oldVersion := Version
	Version = "0.3.3"
	defer func() { Version = oldVersion }()

	_, output := captureHomepage(t, "1\n", "0.3.4")
	if !strings.Contains(output, "Version 0.3.3  (Update Available  0.3.4)") {
		t.Errorf("output missing update-available footer, got:\n%s", output)
	}
}

func TestShowHomepage_noUpdateFooter(t *testing.T) {
	oldVersion := Version
	Version = "0.3.4"
	defer func() { Version = oldVersion }()

	_, output := captureHomepage(t, "1\n", "0.3.4")
	if strings.Contains(output, "Update Available") {
		t.Errorf("footer should not mention an update when already current, got:\n%s", output)
	}
	if !strings.Contains(output, "Version 0.3.4  - a TechLatest Open-source Contribution") {
		t.Errorf("output missing plain version footer, got:\n%s", output)
	}
}
