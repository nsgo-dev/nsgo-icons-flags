# NSGo Flag Icons architecture

The repository is one Go module. Package `flags` at its root draws the [flag-icons](https://github.com/lipis/flag-icons) flags through NSGo's public `core` API. `tools/flaggen` generates it from a pinned upstream commit.

## Code map

| Path | Responsibility |
| --- | --- |
| `doc.go` | The package documentation and the lazy cache the accessors share |
| `flags_gen.go`, `flags_gen_test.go` | Generated: two accessors per flag, and the list tests walk |
| `catalog/catalog.go` | `Find` and `All`: flags by a code known at run time |
| `catalog/catalog_gen.go` | Generated: every flag's metadata and accessors, by English name |
| `upstream.json`, `LICENSE-FLAGS` | Generated: the upstream commit and a digest of its inputs, and upstream's license |
| `tools/flaggen` | The generator |
| `examples/gallery` | The flag finder: search, a virtualized grid, and the chosen flag's code |
| `website/` | The static site GitHub Pages serves, with no build step; `.github/workflows/website.yml` deploys it when it changes on `main` |
| `tools/sitehl` | Highlights the site's code blocks with Pygments when a page is written: write a changed block as a plain `<pre>`, then run `python3 tools/sitehl/highlight.py website/index.html website/zh/index.html` |

## Boundaries

- Use only NSGo's public packages. What the flags need that NSGo lacks goes into NSGo.
- Keep package `flags` free of a runtime registry, so an app links only the flags it calls. Only `catalog` and the gallery list every flag.
- Never edit generated files; change `tools/flaggen` and regenerate. Fidelity to upstream is the point.

## Generating

`flaggen` requires a clean checkout at exactly the commit it is given. It reads `country.json` and both documents of every flag listed there, and fails on a document upstream ships but does not list.

Each document must hold only elements NSGo's SVG reader draws, must parse with `core.ParseSVG`, and must be 640 by 480 (4:3) or 512 by 512 (1:1). The documents are embedded as upstream wrote them, with one exception: NSGo does not draw markers ([nsgo#105](https://github.com/nsgo-dev/nsgo/issues/105)), and the stars of the United States and of the U.S. Minor Outlying Islands are a `marker-mid`. `flaggen` replaces such a path with a translated copy of the marker's content at each of its middle vertices. It does this only for a top-level unpainted path of lines and a top-level marker with default units, origin and orientation, and rejects every other use of markers. Once NSGo draws markers, remove the rewrite.

It writes `flags_gen.go`, `flags_gen_test.go`, `catalog/catalog_gen.go`, `LICENSE-FLAGS` and `upstream.json`, and reports the accessors added and removed; review removals before accepting an update, since they change the public API. `-check` verifies the files instead of writing them, which CI runs against the commit in `upstream.json`.

NSGo ignores `stroke-miterlimit`, which some flags set, so a few sharp joins may differ slightly from a browser's.

## Verification

- `go test ./...` parses and draws every flag in both ratios, checks that the marked flags keep their fifty stars, and tests the generator, the catalog and the gallery headless.
- `go tool nsgo test -target macos examples/gallery` screenshots the gallery. `website/assets/gallery.png` is its `gallery.png`, 1600 pixels wide.
