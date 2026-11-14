# NSGo Flag Icons

NSGo Flag Icons is flag-icons for NSGo as a library of its own. Read [ARCHITECTURE.md](ARCHITECTURE.md) before changing code.

## Principles

- NSGo comes from the commit `go.mod` requires. When the flags need a change in NSGo, land it on NSGo's `main` first, following that repository's own instructions, then run `go get github.com/nsgo-dev/nsgo@<commit>`.
- Keep the [boundaries](ARCHITECTURE.md#boundaries): NSGo's public packages only, no runtime registry in package `flags`, generated files only from `tools/flaggen`.
- Keep `LICENSE` for this repository's code and `LICENSE-FLAGS` for upstream's flags.

## Verification

Run the checks in [Verification](ARCHITECTURE.md#verification) from the repository root. After an upstream update or a change to the gallery's look, look at the macOS screenshots, and refresh `assets/gallery.png` when the gallery changed.

## Documentation

- Record completed user-visible outcomes as one-clause `Unreleased` bullets in [CHANGELOG.md](CHANGELOG.md).
- Keep `README.md` and `README.zh-CN.md` equivalent; everything else, including code comments, is English.
