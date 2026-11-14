"""Highlights the website's plain code blocks with Pygments, once, when a page is written, so the site runs no highlighting script.

Usage, from the repository root: python3 tools/sitehl/highlight.py website/index.html website/zh/index.html

It rewrites each <pre> without a class into <pre class="go"> or <pre class="bash"> holding Pygments' token spans, and leaves highlighted blocks alone, so write a new or changed block as plain <pre> text and run it again. A block that starts with "go get" or "git clone" is shell; the rest are Go.
"""

import html
import re
import subprocess
import sys

SHELL = ("go get", "git clone")


def highlight(code):
    lang = "bash" if code.startswith(SHELL) else "go"
    out = subprocess.run(["pygmentize", "-l", lang, "-f", "html", "-O", "nowrap"], input=code, capture_output=True, text=True, check=True, timeout=30).stdout.rstrip("\n")
    if lang == "go":
        # Pygments marks every Go identifier alike; an identifier before a call is a function.
        out = re.sub(r'<span class="nx">(\w+)</span><span class="p">\(', r'<span class="nf">\1</span><span class="p">(', out)
    else:
        # A command at the start of a line is drawn like the builtins Pygments knows, such as cd.
        out = re.sub(r'(?m)^(go|git)(?=<span class="w">)', r'<span class="nb">\1</span>', out)
    return lang, out


def main(paths):
    for path in paths:
        with open(path, encoding="utf-8") as f:
            page = f.read()

        def replace(match):
            lang, body = highlight(html.unescape(match.group(1)))
            return '<pre class="%s">%s</pre>' % (lang, body)

        page, count = re.subn(r"<pre>(.*?)</pre>", replace, page, flags=re.S)
        with open(path, "w", encoding="utf-8") as f:
            f.write(page)
        print("%s: %d blocks" % (path, count))


if __name__ == "__main__":
    main(sys.argv[1:])
