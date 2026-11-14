package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestIdentifier(t *testing.T) {
	for code, want := range map[string]string{"jp": "JP", "gb-sct": "GBSct", "eu": "EU", "asean": "ASEAN", "sh-ac": "SHAc"} {
		got, err := identifier(code)
		if err != nil || got != want {
			t.Errorf("%s: %q, %v; want %q", code, got, err, want)
		}
	}
	for _, code := range []string{"", "-x", "a--b", "JP", "a_b", "../x", "1x"} {
		if _, err := identifier(code); err == nil {
			t.Errorf("accepted %q", code)
		}
	}
}

func TestPathVertices(t *testing.T) {
	got, err := pathVertices("m0 0 16 11h61 61L47 37h61v2.5-1e1z")
	if err != nil {
		t.Fatal(err)
	}
	want := [][2]float64{{0, 0}, {16, 11}, {77, 11}, {138, 11}, {47, 37}, {108, 37}, {108, 39.5}, {108, 29.5}, {0, 0}}
	if !slices.Equal(got, want) {
		t.Fatalf("vertices %v, want %v", got, want)
	}
	for _, d := range []string{"", "M0 0C1 1 2 2 3 3", "M0 0a1 1 0 0 1 2 2", "5 5", "M0"} {
		if _, err := pathVertices(d); err == nil {
			t.Errorf("accepted %q", d)
		}
	}
}

func TestPrepare(t *testing.T) {
	plain := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 640 480"><path fill="red" d="M0 0h640v480H0z" style="marker:none"/></svg>`
	if got, err := prepare(plain); err != nil || got != plain {
		t.Fatalf("an unmarked document changed: %q, %v", got, err)
	}
	marked := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 20">
  <marker id="s" markerHeight="10" markerWidth="10"><circle cx="5" cy="5" r="4" fill="#fff"/></marker>
  <path fill="none" marker-mid="url(#s)" d="m0 5 30 0h30.5 30"/>
</svg>`
	got, err := prepare(marked)
	if err != nil {
		t.Fatal(err)
	}
	star := `<circle cx="5" cy="5" r="4" fill="#fff"/>`
	want := "<svg xmlns=\"http://www.w3.org/2000/svg\" viewBox=\"0 0 100 20\">\n  \n  " +
		`<g transform="translate(30 5)">` + star + `</g><g transform="translate(60.5 5)">` + star + "</g>\n</svg>"
	if got != want {
		t.Fatalf("expanded\n%s\nwant\n%s", got, want)
	}
	for name, doc := range map[string]string{
		"unsupported element": `<svg viewBox="0 0 1 1"><text>x</text></svg>`,
		"style element":       `<svg viewBox="0 0 1 1"><style>path{fill:red}</style></svg>`,
		"marker-end":          strings.Replace(marked, "marker-mid", "marker-end", 1),
		"oriented marker":     strings.Replace(marked, `markerWidth="10"`, `markerWidth="10" orient="auto"`, 1),
		"painted root":        strings.Replace(marked, `viewBox="0 0 100 20"`, `viewBox="0 0 100 20" stroke-width="2"`, 1),
		"curved path":         strings.Replace(marked, "h30.5 30", "c1 1 2 2 3 3", 1),
		"stroked path":        strings.Replace(marked, `fill="none" marker-mid`, `fill="none" stroke="red" marker-mid`, 1),
		"marker in style":     strings.Replace(plain, "marker:none", "marker-mid:url(#s)", 1),
		"id in marker":        strings.Replace(marked, "<circle ", `<circle id="c" `, 1),
		"missing marker":      strings.Replace(marked, "url(#s)", "url(#t)", 1),
	} {
		if _, err := prepare(doc); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestPinnedGeneration(t *testing.T) {
	upstream := t.TempDir()
	out := filepath.Join(t.TempDir(), "generated")
	mustWrite := func(path, data string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(data), 0644); err != nil {
			t.Fatal(err)
		}
	}
	mustGit := func(args ...string) string {
		t.Helper()
		result, err := git(upstream, args...)
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	wide := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 640 480"><path fill="#bc002d" d="M0 0h640v480H0z"/></svg>`
	square := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512"><path fill="#bc002d" d="M0 0h512v512H0z"/></svg>`
	mustWrite(filepath.Join(upstream, "LICENSE"), "fixture license\n")
	mustWrite(filepath.Join(upstream, "country.json"), `[
  {"capital": "Tokyo", "code": "jp", "continent": "Asia", "flag_1x1": "flags/1x1/jp.svg", "flag_4x3": "flags/4x3/jp.svg", "iso": true, "name": "Japan"},
  {"capital": "", "code": "gb-sct", "continent": "", "flag_1x1": "flags/1x1/gb-sct.svg", "flag_4x3": "flags/4x3/gb-sct.svg", "iso": false, "name": "Scotland"}
]`)
	for _, code := range []string{"jp", "gb-sct"} {
		mustWrite(filepath.Join(upstream, "flags", "4x3", code+".svg"), wide)
		mustWrite(filepath.Join(upstream, "flags", "1x1", code+".svg"), square)
	}
	mustGit("init")
	mustGit("add", ".")
	mustGit("-c", "user.name=Test", "-c", "user.email=test@example.invalid", "-c", "commit.gpgsign=false", "commit", "-m", "fixture")
	commit := mustGit("rev-parse", "HEAD")
	if err := run(upstream, commit, out, false); err != nil {
		t.Fatal(err)
	}
	if err := run(upstream, commit, out, true); err != nil {
		t.Fatal(err)
	}
	code, err := os.ReadFile(filepath.Join(out, "flags_gen.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"func JP() *core.SVG", "func JPSquare() *core.SVG", "func GBSct() *core.SVG", "// JP returns the 4:3 flag of Japan (jp)."} {
		if !strings.Contains(string(code), want) {
			t.Errorf("flags_gen.go lacks %q", want)
		}
	}
	catalog, err := os.ReadFile(filepath.Join(out, "catalog", "catalog_gen.go"))
	if err != nil {
		t.Fatal(err)
	}
	japan, scotland := strings.Index(string(catalog), `Name: "Japan"`), strings.Index(string(catalog), `Name: "Scotland"`)
	if japan < 0 || scotland < 0 || japan > scotland {
		t.Fatalf("the catalog is not sorted by name: %s", catalog)
	}
	generated := filepath.Join(out, "flags_gen.go")
	mustWrite(generated, "outdated")
	if err := run(upstream, commit, out, true); err == nil || !strings.Contains(err.Error(), generated+" is not up to date") {
		t.Fatalf("drift: %v", err)
	}
	if unchanged, err := os.ReadFile(generated); err != nil || string(unchanged) != "outdated" {
		t.Fatal("check mode wrote the package")
	}
	if err := run(upstream, strings.Repeat("0", 40), out, false); err == nil {
		t.Fatal("accepted wrong revision")
	}
	mustWrite(filepath.Join(upstream, "flags", "4x3", "xx.svg"), wide)
	if err := run(upstream, commit, out, false); err == nil || !strings.Contains(err.Error(), "clean") {
		t.Fatalf("dirty checkout: %v", err)
	}
	mustGit("add", ".")
	mustGit("-c", "user.name=Test", "-c", "user.email=test@example.invalid", "-c", "commit.gpgsign=false", "commit", "-m", "unlisted flag")
	if err := run(upstream, mustGit("rev-parse", "HEAD"), out, false); err == nil || !strings.Contains(err.Error(), "not in country.json") {
		t.Fatalf("unlisted flag: %v", err)
	}
}
