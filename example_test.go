package flags_test

import (
	"fmt"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/ui"

	flags "github.com/nsgo-dev/nsgo-icons-flags"
	"github.com/nsgo-dev/nsgo-icons-flags/catalog"
)

func Example() {
	row := core.View{Style: core.Style{Direction: core.Row, Gap: 8}, Children: []core.Node{
		core.Icon{SVG: flags.JP(), Style: core.Style{FontSize: 24}, Label: "Japan"},
		core.Icon{SVG: flags.GBSct(), Style: core.Style{FontSize: 24}, Label: "Scotland"},
		ui.IconButton{Icon: flags.EUSquare, Label: "Europe"},
	}}
	w, h := flags.JP().Size()
	fmt.Println(len(row.Children), w, h)
	// Output: 3 640 480
}

func Example_catalog() {
	if f, ok := catalog.Find("GB"); ok {
		_ = core.Icon{SVG: f.SVG(), Style: core.Style{FontSize: 24}, Label: f.Name}
		fmt.Println(f.Code, f.Name, f.Capital)
	}
	// Output: gb United Kingdom London
}
