package main

import "testing"

func TestParseSkuInfo_success(t *testing.T) {
	raw := []byte(`{"Skus":[{"Id":"19675","Language":"English (United States)"},{"Id":"19676","Language":"French"}]}`)
	langs, err := parseSkuInfo(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(langs) != 2 {
		t.Fatalf("want 2 languages, got %d", len(langs))
	}
	if langs[0].ID != "19675" || langs[0].Language != "English (United States)" {
		t.Errorf("unexpected first lang: %+v", langs[0])
	}
}

func TestParseSkuInfo_rateLimitError(t *testing.T) {
	raw := []byte(`{"Errors":[{"Type":9,"Value":"715-123130 blocked"}]}`)
	_, err := parseSkuInfo(raw)
	if err == nil {
		t.Fatal("expected error for Errors array with Type 9")
	}
}

func TestParseSkuInfo_empty(t *testing.T) {
	_, err := parseSkuInfo([]byte(`{"Skus":[]}`))
	if err == nil {
		t.Fatal("expected error for empty Skus")
	}
}

func TestParseDownloadLinks_architectureString(t *testing.T) {
	raw := []byte(`{"ProductDownloadOptions":[{"Uri":"https://example.com/win11.iso?t=abc","Architecture":"x64"}]}`)
	links, err := parseDownloadLinks(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(links) != 1 {
		t.Fatalf("want 1 link, got %d", len(links))
	}
	if links[0].URI != "https://example.com/win11.iso?t=abc" || links[0].Architecture != "x64" {
		t.Errorf("unexpected link: %+v", links[0])
	}
}

func TestParseDownloadLinks_downloadTypeFallback(t *testing.T) {
	raw := []byte(`{"ProductDownloadOptions":[{"Uri":"https://example.com/a.iso","DownloadType":1}]}`)
	links, err := parseDownloadLinks(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if links[0].Architecture != "x64" {
		t.Errorf("want x64 from DownloadType 1, got %q", links[0].Architecture)
	}
}

func TestParseDownloadLinks_error(t *testing.T) {
	raw := []byte(`{"Errors":[{"Type":4,"Value":"no download links found for this SKU"}]}`)
	_, err := parseDownloadLinks(raw)
	if err == nil {
		t.Fatal("expected error for Errors array")
	}
}

func TestParseDownloadLinks_empty(t *testing.T) {
	_, err := parseDownloadLinks([]byte(`{"ProductDownloadOptions":[]}`))
	if err == nil {
		t.Fatal("expected error for empty options")
	}
}

func TestExtractFwlinks_dedupAndUnescape(t *testing.T) {
	html := `
		<a href="https://go.microsoft.com/fwlink/?linkid=111&amp;clcid=0x409">x64</a>
		<a href="https://go.microsoft.com/fwlink/?linkid=111&amp;clcid=0x409">dup</a>
		<a href="https://go.microsoft.com/fwlink/?linkid=222">arm</a>
	`
	links := extractFwlinks(html)
	if len(links) != 2 {
		t.Fatalf("want 2 unique fwlinks, got %d: %v", len(links), links)
	}
	if links[0] != "https://go.microsoft.com/fwlink/?linkid=111&clcid=0x409" {
		t.Errorf("expected unescaped &, got %q", links[0])
	}
}

func TestDetectArchLang(t *testing.T) {
	url := "https://software-static.download.prss.microsoft.com/.../server2025_arm64_en-us.iso"
	if got := detectArch(url); got != "ARM64" {
		t.Errorf("arch: want ARM64, got %s", got)
	}
	if got := detectLang(url); got != "en-us" {
		t.Errorf("lang: want en-us, got %s", got)
	}
}
