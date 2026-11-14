// Command flaggen imports a pinned, clean flag-icons checkout and writes the flags package: two accessors per flag, the catalog that finds them by code, their provenance and upstream's license.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"go/format"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/nsgo-dev/nsgo/core"
)

const repository = "https://github.com/lipis/flag-icons"

// country is one entry of upstream's country.json.
type country struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	Capital   string `json:"capital"`
	Continent string `json:"continent"`
	ISO       bool   `json:"iso"`
	Flag4x3   string `json:"flag_4x3"`
	Flag1x1   string `json:"flag_1x1"`
}

// entry is a flag ready to write: its metadata, its accessors' name and both documents.
type entry struct {
	country
	symbol       string
	wide, square string
}

type provenance struct {
	Repository string `json:"repository"`
	Commit     string `json:"commit"`
	SHA256     string `json:"inputs_sha256"`
	Count      int    `json:"flag_count"`
}

// shapes are the two ratios upstream draws, with the size each document must have.
var shapes = []struct {
	dir  string
	w, h core.Points
}{{"4x3", 640, 480}, {"1x1", 512, 512}}

func main() {
	upstream := flag.String("upstream", "", "path to a clean flag-icons checkout")
	commit := flag.String("commit", "", "required full upstream commit hash")
	out := flag.String("out", ".", "module root to write the generated files to")
	check := flag.Bool("check", false, "verify the generated files without writing")
	flag.Parse()
	if err := run(*upstream, *commit, *out, *check); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func git(dir string, args ...string) (string, error) {
	b, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git: %w: %s", err, b)
	}
	return strings.TrimSpace(string(b)), nil
}

func run(upstream, commit, out string, check bool) error {
	if upstream == "" || !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(commit) {
		return fmt.Errorf("provide -upstream and a full 40-character -commit")
	}
	head, err := git(upstream, "rev-parse", "HEAD")
	if err != nil {
		return err
	}
	if head != commit {
		return fmt.Errorf("checkout is %s, requested %s", head, commit)
	}
	status, err := git(upstream, "status", "--porcelain", "--untracked-files=all")
	if err != nil {
		return err
	}
	if status != "" {
		return fmt.Errorf("upstream checkout must be clean")
	}
	license, err := os.ReadFile(filepath.Join(upstream, "LICENSE"))
	if err != nil {
		return err
	}
	index, err := os.ReadFile(filepath.Join(upstream, "country.json"))
	if err != nil {
		return err
	}
	var countries []country
	if err := json.Unmarshal(index, &countries); err != nil {
		return fmt.Errorf("country.json: %w", err)
	}
	if len(countries) == 0 {
		return fmt.Errorf("no flags found")
	}
	hash := sha256.New()
	fmt.Fprintf(hash, "LICENSE\x00%s\x00country.json\x00%s\x00", license, index)
	entries := make([]entry, 0, len(countries))
	codes := map[string]bool{}
	symbols := map[string]string{}
	for _, c := range countries {
		if codes[c.Code] {
			return fmt.Errorf("country.json lists %q twice", c.Code)
		}
		codes[c.Code] = true
		if strings.TrimSpace(c.Name) == "" || strings.ContainsAny(c.Name+c.Capital+c.Continent, "\n\r") {
			return fmt.Errorf("%s: invalid metadata", c.Code)
		}
		symbol, err := identifier(c.Code)
		if err != nil {
			return err
		}
		for _, name := range []string{symbol, symbol + "Square"} {
			if previous, ok := symbols[name]; ok {
				return fmt.Errorf("identifier collision: %s and %s", previous, c.Code)
			}
			symbols[name] = c.Code
		}
		e := entry{country: c, symbol: symbol}
		for i, shape := range shapes {
			rel := fmt.Sprintf("flags/%s/%s.svg", shape.dir, c.Code)
			if listed := []string{c.Flag4x3, c.Flag1x1}[i]; listed != rel {
				return fmt.Errorf("%s: country.json points at %q, want %q", c.Code, listed, rel)
			}
			data, err := os.ReadFile(filepath.Join(upstream, filepath.FromSlash(rel)))
			if err != nil {
				return err
			}
			fmt.Fprintf(hash, "%s\x00%s\x00", rel, data)
			doc, err := prepare(string(data))
			if err != nil {
				return fmt.Errorf("%s: %w", rel, err)
			}
			parsed, err := core.ParseSVG([]byte(doc))
			if err != nil {
				return fmt.Errorf("%s: %w", rel, err)
			}
			if w, h := parsed.Size(); w != shape.w || h != shape.h {
				return fmt.Errorf("%s: unexpected size %gx%g", rel, w, h)
			}
			if i == 0 {
				e.wide = doc
			} else {
				e.square = doc
			}
		}
		entries = append(entries, e)
	}
	// Every document upstream ships must be listed, so a flag cannot be dropped silently.
	for _, shape := range shapes {
		paths, err := filepath.Glob(filepath.Join(upstream, "flags", shape.dir, "*.svg"))
		if err != nil {
			return err
		}
		for _, path := range paths {
			if code := strings.TrimSuffix(filepath.Base(path), ".svg"); !codes[code] {
				return fmt.Errorf("flags/%s/%s.svg is not in country.json", shape.dir, code)
			}
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Code < entries[j].Code })
	code, tests, catalog, err := render(entries)
	if err != nil {
		return err
	}
	meta, err := json.MarshalIndent(provenance{repository, commit, hex.EncodeToString(hash.Sum(nil)), len(entries)}, "", "  ")
	if err != nil {
		return err
	}
	files := []struct {
		name string
		data []byte
	}{
		{filepath.Join(out, "flags_gen.go"), code},
		{filepath.Join(out, "flags_gen_test.go"), tests},
		{filepath.Join(out, "catalog", "catalog_gen.go"), catalog},
		{filepath.Join(out, "LICENSE-FLAGS"), license},
		{filepath.Join(out, "upstream.json"), append(meta, '\n')},
	}
	old, err := os.ReadFile(filepath.Join(out, "flags_gen.go"))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	reportChanges(old, code)
	for _, file := range files {
		if check {
			existing, err := os.ReadFile(file.name)
			if err != nil {
				return err
			}
			if !bytes.Equal(existing, file.data) {
				return fmt.Errorf("%s is not up to date", file.name)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(file.name), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(file.name, file.data, 0644); err != nil {
			return err
		}
	}
	fmt.Printf("Validated %d flags at %s\n", len(entries), commit)
	return nil
}

// identifier turns an upstream code into an exported Go name: the first part in capitals and the rest capitalized, so jp is JP and gb-sct is GBSct.
func identifier(code string) (string, error) {
	if !regexp.MustCompile(`^[a-z]+(-[a-z]+)*$`).MatchString(code) {
		return "", fmt.Errorf("invalid flag code %q", code)
	}
	parts := strings.Split(code, "-")
	parts[0] = strings.ToUpper(parts[0])
	for i := 1; i < len(parts); i++ {
		parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
	}
	s := strings.Join(parts, "")
	if !token.IsIdentifier(s) {
		return "", fmt.Errorf("invalid identifier %q", s)
	}
	return s, nil
}

func render(entries []entry) (code, tests, catalog []byte, err error) {
	var c, t, k bytes.Buffer
	c.WriteString("// Code generated by flaggen; DO NOT EDIT.\n\npackage flags\n\nimport \"github.com/nsgo-dev/nsgo/core\"\n\n")
	t.WriteString("// Code generated by flaggen; DO NOT EDIT.\n\npackage flags\n\nimport \"github.com/nsgo-dev/nsgo/core\"\n\nvar allFlags = []struct{ code string; wide, square func() *core.SVG }{\n")
	k.WriteString("// Code generated by flaggen; DO NOT EDIT.\n\npackage catalog\n\nimport flags \"github.com/nsgo-dev/nsgo-icons-flags\"\n\n// all lists every flag by its English name.\nvar all = []Flag{\n")
	for _, e := range entries {
		fmt.Fprintf(&c, "// %s returns the 4:3 flag of %s (%s).\n// It parses once on first use and returns the same shared SVG afterwards.\nfunc %s() *core.SVG { return cache%s.get(%q) }\n\nvar cache%s lazyFlag\n\n",
			e.symbol, e.Name, e.Code, e.symbol, e.symbol, e.wide, e.symbol)
		fmt.Fprintf(&c, "// %sSquare returns the 1:1 flag of %s (%s).\n// It parses once on first use and returns the same shared SVG afterwards.\nfunc %sSquare() *core.SVG { return cache%sSquare.get(%q) }\n\nvar cache%sSquare lazyFlag\n\n",
			e.symbol, e.Name, e.Code, e.symbol, e.symbol, e.square, e.symbol)
		fmt.Fprintf(&t, "{%q, %s, %sSquare},\n", e.Code, e.symbol, e.symbol)
	}
	byName := slices.Clone(entries)
	slices.SortStableFunc(byName, func(a, b entry) int {
		if n := strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)); n != 0 {
			return n
		}
		return strings.Compare(a.Code, b.Code)
	})
	for _, e := range byName {
		fmt.Fprintf(&k, "{Code: %q, Name: %q, Capital: %q, Continent: %q, ISO: %t, wide: flags.%s, square: flags.%sSquare},\n",
			e.Code, e.Name, e.Capital, e.Continent, e.ISO, e.symbol, e.symbol)
	}
	t.WriteString("}\n")
	k.WriteString("}\n")
	if code, err = format.Source(c.Bytes()); err != nil {
		return
	}
	if tests, err = format.Source(t.Bytes()); err != nil {
		return
	}
	catalog, err = format.Source(k.Bytes())
	return
}

func reportChanges(old, current []byte) {
	pattern := regexp.MustCompile(`(?m)^func ([A-Za-z0-9]+)\(\) \*core\.SVG`)
	names := func(data []byte) map[string]bool {
		m := map[string]bool{}
		for _, match := range pattern.FindAllSubmatch(data, -1) {
			m[string(match[1])] = true
		}
		return m
	}
	before, after := names(old), names(current)
	var added, removed []string
	for name := range after {
		if !before[name] {
			added = append(added, name)
		}
	}
	for name := range before {
		if !after[name] {
			removed = append(removed, name)
		}
	}
	sort.Strings(added)
	sort.Strings(removed)
	fmt.Printf("API: %d added, %d removed\n", len(added), len(removed))
	if len(removed) > 0 {
		fmt.Printf("Review removals/renames before accepting: %s\n", strings.Join(removed, ", "))
	}
}
