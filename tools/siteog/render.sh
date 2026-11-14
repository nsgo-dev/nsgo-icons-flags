#!/bin/sh
# Renders tools/siteog/og.html into website/assets/og.png, the image link previews show, with a headless Chromium. Run it from the repository root after changing the page or the gallery screenshot.
#
# CHROME defaults to Google Chrome on macOS; point it at any Chromium binary, such as Playwright's chrome-headless-shell.
set -eu
chrome=${CHROME:-"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"}
page="file://$(pwd)/tools/siteog/og.html"
"$chrome" --headless --no-sandbox --hide-scrollbars --allow-file-access-from-files --force-device-scale-factor=1 --window-size=1200,630 --screenshot="$(pwd)/website/assets/og.png" "$page"
