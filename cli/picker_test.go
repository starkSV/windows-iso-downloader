package main

import (
	"strings"
	"testing"
)

func TestParseChoice_valid(t *testing.T) {
	cases := []struct {
		input string
		max   int
		want  int
	}{
		{"1\n", 3, 1},
		{"3\n", 3, 3},
		{"  2  \n", 5, 2},
	}
	for _, c := range cases {
		got, err := parseChoice(strings.NewReader(c.input), c.max)
		if err != nil {
			t.Errorf("input %q: unexpected error: %v", c.input, err)
			continue
		}
		if got != c.want {
			t.Errorf("input %q: got %d, want %d", c.input, got, c.want)
		}
	}
}

func TestParseChoice_outOfRange(t *testing.T) {
	cases := []string{"0\n", "4\n"}
	for _, input := range cases {
		_, err := parseChoice(strings.NewReader(input), 3)
		if err == nil {
			t.Errorf("input %q: expected error for out-of-range, got nil", input)
		}
	}
}

func TestParseChoice_nonNumeric(t *testing.T) {
	_, err := parseChoice(strings.NewReader("abc\n"), 3)
	if err == nil {
		t.Error("expected error for non-numeric input, got nil")
	}
}

func TestParseChoice_emptyInput(t *testing.T) {
	_, err := parseChoice(strings.NewReader("\n"), 3)
	if err == nil {
		t.Error("expected error for empty input, got nil")
	}
}
