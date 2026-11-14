package catalog

import (
	"slices"
	"strings"
	"testing"

	flags "github.com/nsgo-dev/nsgo-icons-flags"
)

func TestFind(t *testing.T) {
	for code, want := range map[string]string{"jp": "Japan", "JP": "Japan", " gb-SCT ": "Scotland", "eu": "Europe"} {
		f, ok := Find(code)
		if !ok || f.Name != want {
			t.Errorf("Find(%q) = %q, %t; want %q", code, f.Name, ok, want)
		}
	}
	for _, code := range []string{"", "zz", "jpn", "gb_sct"} {
		if _, ok := Find(code); ok {
			t.Errorf("found %q", code)
		}
	}
	jp, _ := Find("jp")
	if jp.SVG() != flags.JP() || jp.Square() != flags.JPSquare() {
		t.Fatal("a found flag does not share the accessors' SVGs")
	}
	if !jp.ISO || jp.Capital != "Tokyo" || jp.Continent != "Asia" {
		t.Fatalf("jp metadata: %+v", jp)
	}
	if eu, _ := Find("eu"); eu.ISO {
		t.Fatal("eu is not an ISO 3166-1 code")
	}
}

func TestAll(t *testing.T) {
	list := All()
	if len(list) != len(all) || len(list) < 250 {
		t.Fatalf("%d flags", len(list))
	}
	if !slices.IsSortedFunc(list, func(a, b Flag) int { return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)) }) {
		t.Fatal("not sorted by name")
	}
	list[0] = Flag{}
	if All()[0].Code == "" {
		t.Fatal("All returned the catalog itself")
	}
	codes := map[string]bool{}
	for _, f := range all {
		if codes[f.Code] {
			t.Fatalf("%s listed twice", f.Code)
		}
		codes[f.Code] = true
		if f.SVG() == nil || f.Square() == nil {
			t.Fatalf("%s has no SVG", f.Code)
		}
	}
}

func TestZeroFlag(t *testing.T) {
	var f Flag
	if f.SVG() != nil || f.Square() != nil {
		t.Fatal("the zero Flag has a flag")
	}
}
