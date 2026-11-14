package flags

import (
	"sync"
	"testing"

	"github.com/nsgo-dev/nsgo/core"
)

func TestAllFlags(t *testing.T) {
	if len(allFlags) == 0 {
		t.Fatal("empty flag collection")
	}
	seen := map[string]bool{}
	for _, flag := range allFlags {
		if seen[flag.code] {
			t.Fatalf("%s listed twice", flag.code)
		}
		seen[flag.code] = true
		for _, shape := range []struct {
			name string
			get  func() *core.SVG
			w, h core.Points
		}{{"4x3", flag.wide, 640, 480}, {"1x1", flag.square, 512, 512}} {
			t.Run(flag.code+"/"+shape.name, func(t *testing.T) {
				t.Parallel()
				results := make([]*core.SVG, 8)
				var wg sync.WaitGroup
				for i := range results {
					wg.Go(func() { results[i] = shape.get() })
				}
				wg.Wait()
				first := results[0]
				if first == nil {
					t.Fatal("nil SVG")
				}
				for _, result := range results {
					if result != first {
						t.Fatal("accessor did not reuse its SVG")
					}
				}
				if w, h := first.Size(); w != shape.w || h != shape.h {
					t.Fatalf("size = %gx%g", w, h)
				}
				var d core.Drawing
				first.Draw(&d, core.Size{Width: shape.w, Height: shape.h}, 0)
				if len(d.Ops) == 0 {
					t.Fatal("draws nothing")
				}
			})
		}
	}
}

// TestMarkedStars checks that the flags whose stars upstream draws with an SVG marker, which NSGo does not draw, still have all fifty.
func TestMarkedStars(t *testing.T) {
	for name, get := range map[string]func() *core.SVG{"us": US, "us-square": USSquare, "um": UM, "um-square": UMSquare} {
		var d core.Drawing
		w, h := get().Size()
		get().Draw(&d, core.Size{Width: w, Height: h}, 0)
		white := 0
		for _, op := range d.Ops {
			if op.Kind == core.DrawFillColor && op.Color == core.RGB(0xffffff) {
				white++
			}
		}
		if white != 50 {
			t.Errorf("%s fills %d white shapes, want 50 stars", name, white)
		}
	}
}

func TestLazyFlag(t *testing.T) {
	var cache lazyFlag
	if cache.svg != nil {
		t.Fatal("initialized before use")
	}
	first := cache.get(`<svg viewBox="0 0 640 480"><rect width="640" height="480" fill="red"/></svg>`)
	if cache.get("not parsed again") != first {
		t.Fatal("not cached")
	}
}
