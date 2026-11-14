// Command gallery browses the flags: a search over every flag's name, code and capital, a grid of the ones that match, and beside it the chosen flag and the code that draws it.
package main

import (
	"fmt"
	"go/format"
	"slices"
	"strings"

	"github.com/nsgo-dev/nsgo"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/desktop"
	desklocales "github.com/nsgo-dev/nsgo/desktop/locales"
	"github.com/nsgo-dev/nsgo/ui"

	"github.com/nsgo-dev/nsgo-icons-flags/catalog"
)

// everyFlag is the catalog by English name.
var everyFlag = catalog.All()

// state is what the gallery's parts share.
type state struct {
	query    *core.Signal[string]
	square   *core.Signal[bool]
	chosen   *ui.Selection
	matching func() []catalog.Flag
}

// selected is the chosen flag, Japan's when none is.
func (s *state) selected() catalog.Flag {
	if keys := s.chosen.Keys(); len(keys) > 0 {
		if f, ok := catalog.Find(keys[0].(string)); ok {
			return f
		}
	}
	f, _ := catalog.Find("jp")
	return f
}

// at is the matching flag at i, which the list asks for as the matches change under it; ok is false past their end.
func (s *state) at(i int) (catalog.Flag, bool) {
	list := s.matching()
	if i < 0 || i >= len(list) {
		return catalog.Flag{}, false
	}
	return list[i], true
}

// search returns the flags whose code starts with query or whose name or capital holds it, ignoring case. The flag query is the code of comes first, then names that start with it, then the rest, each by name.
func search(query string) []catalog.Flag {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return everyFlag
	}
	rank := func(f catalog.Flag) int {
		switch {
		case f.Code == q:
			return 0
		case strings.HasPrefix(strings.ToLower(f.Name), q):
			return 1
		}
		return 2
	}
	var found []catalog.Flag
	for _, f := range everyFlag {
		if strings.HasPrefix(f.Code, q) || strings.Contains(strings.ToLower(f.Name), q) || strings.Contains(strings.ToLower(f.Capital), q) {
			found = append(found, f)
		}
	}
	slices.SortStableFunc(found, func(a, b catalog.Flag) int { return rank(a) - rank(b) })
	return found
}

// accessor is the name of the function package flags draws f with, by the rule its generator names them: jp is JP and gb-sct is GBSct.
func accessor(f catalog.Flag, square bool) string {
	parts := strings.Split(f.Code, "-")
	parts[0] = strings.ToUpper(parts[0])
	for i := 1; i < len(parts); i++ {
		parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
	}
	name := strings.Join(parts, "")
	if square {
		name += "Square"
	}
	return name
}

// usage is the Go that draws f.
func usage(f catalog.Flag, square bool) string {
	src := fmt.Sprintf("package p\n\nvar _ = core.Icon{\nSVG: flags.%s(),\nStyle: core.Style{FontSize: 24},\nLabel: %q,\n}\n", accessor(f, square), f.Name)
	// gofmt aligns the fields as an editor would.
	out, err := format.Source([]byte(src))
	if err != nil {
		return src
	}
	return strings.TrimSuffix(strings.TrimPrefix(string(out), "package p\n\nvar _ = "), "\n")
}

// flagView is f's flag at a height, with a hairline so white edges show on a white background.
func flagView(cx *core.Scope, f catalog.Flag, square bool, height core.Points, testID string) core.Node {
	colors := ui.ColorsOf(cx)
	svg, width := f.SVG(), height*4/3
	if square {
		svg, width = f.Square(), height
	}
	return core.View{Style: core.Style{Width: core.Pt(width), Height: core.Pt(height), BorderWidth: 1},
		StyleFunc: func() core.Style { return core.Style{BorderColor: colors.Line()} },
		Children: []core.Node{core.Icon{SVG: svg, Label: f.Name, TestID: testID,
			Style: core.Style{Width: core.Pct(100), Height: core.Pct(100)}}}}
}

// gutter is the grid's padding at its sides and end.
const gutter = 12

// cellWidth is a grid cell's width.
const cellWidth core.Points = 112

// browser is the search field over the grid of the flags that match.
func browser(cx *core.Scope, s *state) core.Node {
	colors := ui.ColorsOf(cx)
	width := core.NewSignal(cx, core.Points(0))
	columns := core.NewMemo(cx, func() int { return max(1, int((width.Get()+8)/(cellWidth+8))) })
	field := ui.Input{Value: s.query.Get, OnChange: s.query.Set, TestID: "search",
		Placeholder: fmt.Sprintf("Search %d flags by name, code or capital", len(everyFlag)), Style: core.Style{FlexGrow: 1}}
	count := core.Text{ContentFunc: func() string { return countText(len(s.matching())) },
		Style: ui.TypeCaption, StyleFunc: func() core.Style { return core.Style{Color: colors.Muted()} }}
	grid := core.Dynamic(func(*core.Scope) core.Node {
		n, square := columns.Get(), s.square.Get()
		return ui.List{Count: func() int { return len(s.matching()) }, Columns: n, Gap: 8, TapSelects: true,
			Key:       func(i int) any { f, _ := s.at(i); return f.Code },
			RowLabel:  func(i int) string { f, _ := s.at(i); return f.Name },
			Selection: s.chosen, EstimatedSize: 96, Label: "Flags", TestID: "flags",
			Empty: core.Text{Content: "No flags match", Style: ui.TypeBody.Merge(core.Style{AlignSelf: core.AlignCenter, Margin: core.All(32)}),
				StyleFunc: func() core.Style { return core.Style{Color: colors.Muted()} }},
			Row: func(cx *core.Scope, row core.ListRow) core.Node {
				return core.Dynamic(func(cx *core.Scope) core.Node {
					f, ok := s.at(row.Index())
					if !ok {
						return core.View{}
					}
					return core.View{Style: core.Style{AlignItems: core.AlignCenter, Gap: 8, Width: core.Pct(100)}, Children: []core.Node{
						flagView(cx, f, square, 42, "flag-"+f.Code),
						core.Text{Content: f.Name, Style: ui.TypeCaption.Merge(core.Style{NumberOfLines: 1, MaxWidth: core.Pct(100)}),
							StyleFunc: func() core.Style { return core.Style{Color: colors.Muted()} }},
					}}
				})
			},
			RowStyle: core.Style{Direction: core.Column, JustifyContent: core.JustifyCenter,
				Width: core.Pt(cellWidth), Height: core.Pt(96), Padding: core.All(8)},
			Style: ui.FillRest, ListStyle: core.Style{Padding: core.Edges{Left: gutter, Right: gutter, Bottom: gutter}}}
	})
	return core.View{Style: ui.FillRest, Children: []core.Node{
		core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 12, Padding: core.Symmetric(12, gutter)},
			Children: []core.Node{field, count}},
		core.View{Style: ui.FillRest, OnLayout: func(b core.Bounds) { width.Set(b.Size.Width - 2*gutter) }, Children: []core.Node{grid}},
	}}
}

func countText(n int) string {
	if n == 1 {
		return "1 flag"
	}
	return fmt.Sprintf("%d flags", n)
}

// panelWidth is the panel's width beside the grid.
const panelWidth = 340

// panel is the chosen flag, what upstream records about it, the code that draws it, and the ratio.
func panel(cx *core.Scope, s *state) core.Node {
	colors := ui.ColorsOf(cx)
	muted := func() core.Style { return core.Style{Color: colors.Muted()} }
	code := func() string { return usage(s.selected(), s.square.Get()) }
	copied := core.NewSignal(cx, "")
	preview := core.Dynamic(func(cx *core.Scope) core.Node {
		f := s.selected()
		facts := strings.ToUpper(f.Code)
		for _, fact := range []string{f.Capital, f.Continent} {
			if fact != "" {
				facts += " · " + fact
			}
		}
		return core.View{Style: core.Style{AlignItems: core.AlignCenter, Gap: 6}, Children: []core.Node{
			core.View{Style: core.Style{AlignSelf: core.AlignStretch, Height: core.Pt(200), AlignItems: core.AlignCenter,
				JustifyContent: core.JustifyCenter, CornerRadius: ui.RadiusControl},
				StyleFunc: func() core.Style { return core.Style{Background: colors.Inset()} },
				Children:  []core.Node{flagView(cx, f, s.square.Get(), 150, "preview")}},
			core.Text{Content: f.Name, Style: ui.TypeTitle.Merge(core.Style{Margin: core.Edges{Top: 6}}), TestID: "preview-name"},
			core.Text{Content: facts, Style: ui.TypeCaption, StyleFunc: muted, TestID: "preview-facts"},
		}}
	})
	copyButton := core.Dynamic(func(*core.Scope) core.Node {
		text := "Copy"
		if copied.Get() == code() {
			text = "Copied"
		}
		return ui.Button{Text: text, Variant: ui.ButtonSecondary, Size: ui.SizeSmall, TestID: "copy",
			OnPress: func() { c := code(); cx.SetClipboard(c); copied.Set(c) }}
	})
	ratio := ui.Tabs[bool]{Size: ui.SizeSmall, Equal: true, Label: "Ratio", TestID: "ratio", Value: s.square.Get, OnChange: s.square.Set,
		Style: core.Style{FlexGrow: 1}, Options: []ui.Option[bool]{
			{Value: false, Label: "4:3", TestID: "ratio-wide"},
			{Value: true, Label: "1:1", TestID: "ratio-square"},
		}}
	return core.View{Style: core.Style{Width: core.Pt(panelWidth), Height: core.Pct(100), FlexShrink: 0, Borders: core.Edges{Left: 1}},
		StyleFunc: func() core.Style { return core.Style{BorderColor: colors.Line(), Background: colors.Surface()} },
		Children: []core.Node{ui.ScrollView{Style: ui.FillRest, TestID: "panel", Children: []core.Node{
			core.View{Style: core.Style{Padding: core.All(16), Gap: 20}, Children: []core.Node{
				preview,
				core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 12}, Children: []core.Node{
					core.Text{Content: "Ratio", Style: ui.TypeBody.Merge(core.Style{Width: core.Pt(56)}), StyleFunc: muted}, ratio}},
				core.View{Style: core.Style{Gap: 8}, Children: []core.Node{
					core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, JustifyContent: core.JustifySpaceBetween},
						Children: []core.Node{
							core.Text{Content: "Usage", Style: ui.TypeCaption.Merge(core.Style{FontWeight: ui.Semibold}), StyleFunc: muted},
							copyButton,
						}},
					core.View{Style: core.Style{Padding: core.All(12), CornerRadius: ui.RadiusControl}, Selectable: true,
						StyleFunc: func() core.Style { return core.Style{Background: colors.Inset()} },
						Children:  []core.Node{core.Text{ContentFunc: code, Style: ui.TypeCode, TestID: "usage"}}},
				}},
			}},
		}}}}
}

// Root is the gallery's one screen: the search and the grid, and the panel beside them.
func Root(cx *core.Scope) core.Node {
	colors := ui.ColorsOf(cx)
	s := &state{query: core.NewSignal(cx, ""), square: core.NewSignal(cx, false), chosen: ui.NewSelection(cx, false)}
	if len(s.chosen.Keys()) == 0 {
		s.chosen.Set("jp")
	}
	s.matching = core.NewMemoFunc(cx, func() []catalog.Flag { return search(s.query.Get()) }, nil).Get
	desktop.SetTitle(cx, "Flags")
	cx.Effect(func() { desktop.SetApplicationMenu(cx, menu(cx)) })
	return core.View{Style: ui.FillRest.Merge(core.Style{Direction: core.Row}),
		StyleFunc: func() core.Style { return core.Style{Background: colors.Background()} },
		Children:  []core.Node{browser(cx, s), panel(cx, s)}}
}

// menu is the application menu: the app menu and Edit of roles, and Window.
func menu(cx *core.Scope) []desktop.MenuItem {
	tr := desklocales.For(cx)
	return []desktop.MenuItem{
		{Label: "Flags", Items: []desktop.MenuItem{
			{Role: desktop.MenuAbout}, {Separator: true},
			{Role: desktop.MenuHide}, {Role: desktop.MenuHideOthers}, {Role: desktop.MenuShowAll}, {Separator: true},
			{Role: desktop.MenuQuit},
		}},
		{Label: tr.Edit(), Items: []desktop.MenuItem{
			{Role: desktop.MenuUndo}, {Role: desktop.MenuRedo}, {Separator: true},
			{Role: desktop.MenuCut}, {Role: desktop.MenuCopy}, {Role: desktop.MenuPaste}, {Role: desktop.MenuSelectAll},
		}},
		{Label: tr.Window(), Items: []desktop.MenuItem{
			{Role: desktop.MenuMinimize}, {Role: desktop.MenuZoom}, {Role: desktop.MenuToggleFullScreen}, {Separator: true},
			{Role: desktop.MenuClose},
		}},
	}
}

func main() { nsgo.Run(Root, nsgo.WindowSize(1100, 760)) }
