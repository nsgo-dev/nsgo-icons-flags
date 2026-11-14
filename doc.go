// Package flags provides the flag-icons country flags for NSGo: two accessors per flag, each an SVG that core.Icon draws in the flag's own colors.
//
//	core.Icon{SVG: flags.JP(), Style: core.Style{FontSize: 24}, Label: "Japan"}
//
// JP returns the 4:3 flag and JPSquare the 1:1 one. Names are upstream's codes with the first part in capitals and the rest capitalized: jp is JP, gb-sct is GBSct and eu is EU. Upstream's codes are ISO 3166-1 alpha-2 codes, a few subdivisions such as gb-sct, and a few organizations such as eu and un.
//
// Each accessor parses its SVG on first use and returns the same shared SVG afterwards. There is no runtime registry here, so a binary carries only the flags it calls; package catalog finds a flag by a code known only at run time, and carries them all.
//
// The generated flags are pinned in upstream.json and covered by upstream's MIT license in LICENSE-FLAGS.
package flags

import (
	"sync"

	"github.com/nsgo-dev/nsgo/core"
)

// Zero-value caches avoid package initializers rooting unused SVG resources.
type lazyFlag struct {
	once sync.Once
	svg  *core.SVG
}

func (f *lazyFlag) get(data string) *core.SVG {
	f.once.Do(func() { f.svg = core.MustParseSVG([]byte(data)) })
	return f.svg
}
