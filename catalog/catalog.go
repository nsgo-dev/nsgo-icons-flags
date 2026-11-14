// Package catalog finds a flag by a code known only at run time, such as a user's country, and lists every flag with its English name.
//
//	if f, ok := catalog.Find(region); ok {
//		core.Icon{SVG: f.SVG(), Style: core.Style{FontSize: 24}, Label: f.Name}
//	}
//
// Importing it links every flag in both ratios into the binary, about 4 MB; an app that knows its flags at build time calls package flags' accessors instead, and carries only those.
package catalog

import (
	"strings"
	"sync"

	"github.com/nsgo-dev/nsgo/core"
)

// Flag is one flag and what upstream records about it.
type Flag struct {
	// Code is upstream's lowercase code: an ISO 3166-1 alpha-2 code such as "jp", a subdivision such as "gb-sct", or an organization such as "eu".
	Code string
	// Name is the English name, such as "Japan".
	Name string
	// Capital and Continent are empty where upstream records none.
	Capital, Continent string
	// ISO reports whether Code is an ISO 3166-1 alpha-2 code.
	ISO bool

	wide, square func() *core.SVG
}

// SVG returns the 4:3 flag, parsed once and shared. It is nil for the zero Flag.
func (f Flag) SVG() *core.SVG {
	if f.wide == nil {
		return nil
	}
	return f.wide()
}

// Square returns the 1:1 flag, parsed once and shared. It is nil for the zero Flag.
func (f Flag) Square() *core.SVG {
	if f.square == nil {
		return nil
	}
	return f.square()
}

var byCode = sync.OnceValue(func() map[string]Flag {
	m := make(map[string]Flag, len(all))
	for _, f := range all {
		m[f.Code] = f
	}
	return m
})

// Find returns the flag for code, ignoring case and surrounding space, so "JP" and "gb-SCT" are found. A region from a locale, such as the "GB" of "en-GB", is a code it takes.
func Find(code string) (Flag, bool) {
	f, ok := byCode()[strings.ToLower(strings.TrimSpace(code))]
	return f, ok
}

// All returns every flag, sorted by English name. The slice is the caller's.
func All() []Flag {
	out := make([]Flag, len(all))
	copy(out, all)
	return out
}
