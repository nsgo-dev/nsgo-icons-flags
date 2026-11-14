package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"
)

// drawn are the elements NSGo's SVG reader draws or reads. A document with any other element would lose part of its drawing without an error, so prepare rejects it.
var drawn = map[string]bool{
	"svg": true, "g": true, "a": true, "switch": true, "use": true, "symbol": true, "defs": true,
	"path": true, "rect": true, "circle": true, "ellipse": true, "line": true, "polyline": true, "polygon": true,
	"clipPath": true, "linearGradient": true, "radialGradient": true, "stop": true,
	"title": true, "desc": true, "metadata": true,
}

// element is an element's place in the document: its byte span, its content's span, its depth and its attributes.
type element struct {
	name                   string
	attrs                  map[string]string
	depth                  int
	start, inner, end, out int64
}

// prepare returns the document NSGo draws as upstream does. NSGo's reader does not draw markers (https://github.com/nsgo-dev/nsgo/issues/105), and the United States' stars are markers, so prepare places a copy of each marker's content at every vertex it marks. Every other element must be one NSGo draws.
func prepare(doc string) (string, error) {
	elements, err := scan(doc)
	if err != nil {
		return "", err
	}
	markers := map[string]element{}
	var users []element
	for _, e := range elements {
		if e.name == "marker" {
			markers[e.attrs["id"]] = e
		}
		for name, value := range e.attrs {
			if name == "style" && strings.Contains(value, "marker") {
				for _, decl := range strings.Split(value, ";") {
					if k, v, _ := strings.Cut(decl, ":"); strings.HasPrefix(strings.TrimSpace(k), "marker") && strings.TrimSpace(v) != "none" {
						return "", fmt.Errorf("<%s> sets markers in its style", e.name)
					}
				}
			}
			if strings.HasPrefix(name, "marker-") || name == "marker" {
				users = append(users, e)
				break
			}
		}
	}
	if len(markers) == 0 && len(users) == 0 {
		return doc, checkDrawn(elements)
	}
	if root := elements[0]; !onlyAttrs(root.attrs, "xmlns", "xmlns:xlink", "id", "viewBox", "width", "height") {
		return "", fmt.Errorf("a marked document's root has presentation attributes, which markers would not inherit")
	}
	type edit struct {
		from, to int64
		text     string
	}
	var edits []edit
	for id, m := range markers {
		if m.depth != 1 || !onlyAttrs(m.attrs, "id", "markerWidth", "markerHeight") {
			return "", fmt.Errorf("marker %q: only a top-level marker with the default units, origin and orientation is supported", id)
		}
		if strings.Contains(doc[m.inner:m.end], "id=") {
			return "", fmt.Errorf("marker %q: copies of its content would repeat its ids", id)
		}
		edits = append(edits, edit{m.start, m.out, ""})
	}
	for _, u := range users {
		ref := u.attrs["marker-mid"]
		id, ok := strings.CutPrefix(ref, "url(#")
		id, closed := strings.CutSuffix(id, ")")
		m, found := markers[id]
		if u.name != "path" || u.depth != 1 || !ok || !closed || !found || !onlyAttrs(u.attrs, "d", "fill", "marker-mid") || u.attrs["fill"] != "none" {
			return "", fmt.Errorf("<%s marker-mid=%q>: only a top-level unpainted path with marker-mid is supported", u.name, ref)
		}
		vertices, err := pathVertices(u.attrs["d"])
		if err != nil {
			return "", err
		}
		content := strings.TrimSpace(doc[m.inner:m.end])
		var b strings.Builder
		for i, v := range vertices {
			if i == 0 || i == len(vertices)-1 {
				continue
			}
			fmt.Fprintf(&b, "<g transform=\"translate(%s %s)\">%s</g>", number(v[0]), number(v[1]), content)
		}
		edits = append(edits, edit{u.start, u.out, b.String()})
	}
	sort.Slice(edits, func(i, j int) bool { return edits[i].from > edits[j].from })
	for _, e := range edits {
		doc = doc[:e.from] + e.text + doc[e.to:]
	}
	again, err := scan(doc)
	if err != nil {
		return "", err
	}
	return doc, checkDrawn(again)
}

func checkDrawn(elements []element) error {
	for _, e := range elements {
		if !drawn[e.name] {
			return fmt.Errorf("NSGo does not draw <%s>", e.name)
		}
	}
	return nil
}

func onlyAttrs(attrs map[string]string, allowed ...string) bool {
	for name := range attrs {
		found := false
		for _, a := range allowed {
			if name == a {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// scan lists the document's elements in order with their spans.
func scan(doc string) ([]element, error) {
	d := xml.NewDecoder(strings.NewReader(doc))
	var elements []element
	var open []int
	for {
		before := d.InputOffset()
		tok, err := d.RawToken()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			attrs := map[string]string{}
			for _, a := range t.Attr {
				name := a.Name.Local
				if a.Name.Space != "" {
					name = a.Name.Space + ":" + name
				}
				attrs[name] = a.Value
			}
			name := t.Name.Local
			if t.Name.Space != "" {
				name = t.Name.Space + ":" + name
			}
			elements = append(elements, element{name: name, attrs: attrs, depth: len(open), start: before, inner: d.InputOffset()})
			open = append(open, len(elements)-1)
		case xml.EndElement:
			if len(open) == 0 {
				return nil, fmt.Errorf("unbalanced </%s>", t.Name.Local)
			}
			e := &elements[open[len(open)-1]]
			open = open[:len(open)-1]
			// A self-closing element ends where it starts its content.
			e.end, e.out = before, d.InputOffset()
			if e.end < e.inner {
				e.end = e.inner
			}
		}
	}
	if len(elements) == 0 || elements[0].name != "svg" {
		return nil, fmt.Errorf("not an SVG document")
	}
	return elements, nil
}

// pathVertices returns the vertices of path data made of lines: every command's end point, closepath's included. Curves and arcs are rejected, since a marker on them would also need their direction.
func pathVertices(d string) ([][2]float64, error) {
	var out [][2]float64
	var x, y, sx, sy float64
	var cmd byte
	i := 0
	skip := func() {
		for i < len(d) && (d[i] == ' ' || d[i] == ',' || d[i] == '\n' || d[i] == '\t' || d[i] == '\r') {
			i++
		}
	}
	num := func() (float64, error) {
		skip()
		j := i
		if j < len(d) && (d[j] == '+' || d[j] == '-') {
			j++
		}
		digits := 0
		for j < len(d) && d[j] >= '0' && d[j] <= '9' {
			j++
			digits++
		}
		if j < len(d) && d[j] == '.' {
			j++
			for j < len(d) && d[j] >= '0' && d[j] <= '9' {
				j++
				digits++
			}
		}
		if digits == 0 {
			return 0, fmt.Errorf("path data: expected a number at %d", i)
		}
		if j < len(d) && (d[j] == 'e' || d[j] == 'E') {
			k := j + 1
			if k < len(d) && (d[k] == '+' || d[k] == '-') {
				k++
			}
			if k < len(d) && d[k] >= '0' && d[k] <= '9' {
				for k < len(d) && d[k] >= '0' && d[k] <= '9' {
					k++
				}
				j = k
			}
		}
		v, err := strconv.ParseFloat(d[i:j], 64)
		i = j
		return v, err
	}
	for {
		skip()
		if i >= len(d) {
			break
		}
		if c := d[i]; (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
			cmd = c
			i++
		} else if cmd == 0 || cmd == 'z' || cmd == 'Z' {
			return nil, fmt.Errorf("path data: expected a command at %d", i)
		}
		rel := cmd >= 'a'
		switch cmd {
		case 'M', 'm', 'L', 'l':
			a, err := num()
			if err != nil {
				return nil, err
			}
			b, err := num()
			if err != nil {
				return nil, err
			}
			if rel {
				a, b = x+a, y+b
			}
			x, y = a, b
			if cmd == 'M' || cmd == 'm' {
				sx, sy = x, y
				// Pairs after a moveto are lines.
				cmd = map[byte]byte{'M': 'L', 'm': 'l'}[cmd]
			}
		case 'H', 'h':
			a, err := num()
			if err != nil {
				return nil, err
			}
			if rel {
				a += x
			}
			x = a
		case 'V', 'v':
			a, err := num()
			if err != nil {
				return nil, err
			}
			if rel {
				a += y
			}
			y = a
		case 'Z', 'z':
			x, y = sx, sy
		default:
			return nil, fmt.Errorf("path data: marked paths may only use lines, not %q", cmd)
		}
		out = append(out, [2]float64{x, y})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("path data: empty")
	}
	return out, nil
}

// number writes a coordinate as SVG reads it, without the binary noise of the sums that made it.
func number(v float64) string {
	return strconv.FormatFloat(math.Round(v*1e4)/1e4, 'f', -1, 64)
}
