package main

import (
	"slices"
	"testing"
	"time"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/nsgotest"

	"github.com/nsgo-dev/nsgo-icons-flags/catalog"
)

// TestMain gives the platform the main thread under nsgo test -target.
func TestMain(m *testing.M) { nsgotest.Main(m) }

func start(t testing.TB) *nsgotest.App {
	return nsgotest.Start(t, Root, nsgotest.Size(1100, 760), nsgotest.Environment(core.Environment{Locale: "en"}))
}

// TestBrowse lays out the grid within the window beside the panel, mounting only the flags near the visible ones.
func TestBrowse(t *testing.T) {
	app := start(t)
	grid := app.Find("flags").Frame
	if grid.Size.Height > 760 {
		t.Fatalf("grid is %v tall, past the window", grid.Size.Height)
	}
	if grid.Origin.X != 0 || grid.Origin.X+grid.Size.Width != 1100-panelWidth {
		t.Fatalf("grid spans %v, not edge to edge beside the panel", grid)
	}
	mounted := 0
	for _, f := range everyFlag {
		if app.Exists("flag-" + f.Code) {
			mounted++
		}
	}
	if mounted == 0 || mounted >= len(everyFlag)/2 {
		t.Fatalf("%d of %d flags mounted", mounted, len(everyFlag))
	}
	if got := app.Find("preview-name").Text; got != "Japan" {
		t.Fatalf("preview shows %q, want Japan", got)
	}
	app.Screenshot("gallery")
}

// TestSearchAndCopy narrows the grid, chooses a flag and copies its code, and shows it square.
func TestSearchAndCopy(t *testing.T) {
	app := start(t)
	app.Type("search", "united")
	app.WaitFor("text:5 flags", 2*time.Second)
	app.Press("flag-us")
	if got := app.Find("preview-name").Text; got != "United States of America" {
		t.Fatalf("preview shows %q", got)
	}
	if got := app.Find("preview-facts").Text; got != "US · Washington, D.C. · North America" {
		t.Fatalf("facts are %q", got)
	}
	want := "core.Icon{\n\tSVG:   flags.US(),\n\tStyle: core.Style{FontSize: 24},\n\tLabel: \"United States of America\",\n}"
	if got := app.Find("usage").Text; got != want {
		t.Fatalf("usage is\n%s\nwant\n%s", got, want)
	}
	app.Press("copy")
	app.Find("text:Copied")
	app.Screenshot("gallery-search")
	app.Press("ratio-square")
	if got := app.Find("usage").Text; got != "core.Icon{\n\tSVG:   flags.USSquare(),\n\tStyle: core.Style{FontSize: 24},\n\tLabel: \"United States of America\",\n}" {
		t.Fatalf("square usage is\n%s", got)
	}
	app.Screenshot("gallery-square")
	app.Clear("search")
	app.Type("search", "no such flag")
	app.Find("text:No flags match")
}

func TestSearch(t *testing.T) {
	codes := func(query string) []string {
		var out []string
		for _, f := range search(query) {
			out = append(out, f.Code)
		}
		return out
	}
	if got := codes("JP"); !slices.Equal(got, []string{"jp"}) {
		t.Errorf("JP found %v", got)
	}
	if got := codes("tokyo"); !slices.Equal(got, []string{"jp"}) {
		t.Errorf("tokyo found %v", got)
	}
	if got := codes("gb-"); !slices.Equal(got, []string{"gb-eng", "gb-nir", "gb-sct", "gb-wls"}) {
		t.Errorf("gb- found %v", got)
	}
	// A code match comes before names that merely contain the query.
	if got := codes("in"); len(got) < 2 || got[0] != "in" {
		t.Errorf("in found %v", got)
	}
	if got := len(search(" ")); got != len(everyFlag) {
		t.Errorf("an empty search found %d", got)
	}
	if got := codes("zzz"); got != nil {
		t.Errorf("zzz found %v", got)
	}
}

func TestAccessor(t *testing.T) {
	for code, want := range map[string]string{"jp": "JP", "gb-sct": "GBSct", "asean": "ASEAN"} {
		f, _ := catalog.Find(code)
		if got := accessor(f, false); got != want {
			t.Errorf("%s: %s, want %s", code, got, want)
		}
		if got := accessor(f, true); got != want+"Square" {
			t.Errorf("%s square: %s", code, got)
		}
	}
}
