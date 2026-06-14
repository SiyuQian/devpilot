#!/usr/bin/env python3
"""Fetch a URL and extract its main article text VERBATIM.

This exists because WebFetch returns model-summarized content, not the source's
actual words — so `.orig` blocks built on WebFetch are compressed before the skill
ever runs. This script returns the real text instead.

Usage:
    python3 fetch_source.py <url>

Output:
    stdout  — extracted verbatim text (title line, blank line, then body).
    stderr  — one JSON diagnostic line: {"method","word_count","thin","title","final_url"}.

Contract for the caller (workflow.md):
    - Exit 0 with "thin": false  → trust stdout as the verbatim source text.
    - Exit 0 with "thin": true   → extraction was weak; fall back to WebFetch and
                                   TAG the artifact "based on a summarized fetch,
                                   not verbatim".
    - Non-zero exit              → fetch failed entirely; fall back to WebFetch (tagged).

No third-party services and no network egress beyond the target URL. Standard library
plus BeautifulSoup (bs4) when available; degrades to a stdlib-only parser otherwise.
"""

import gzip
import io
import json
import sys
import urllib.request

# A page that yields fewer words than this is treated as a failed/thin extraction,
# so the caller degrades honestly instead of close-reading a stub.
THIN_WORD_THRESHOLD = 120

UA = ("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) "
      "AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36")

# Containers that never hold article prose; dropped before extraction.
BOILERPLATE_TAGS = ("script", "style", "noscript", "nav", "header", "footer",
                    "aside", "form", "figure", "button", "svg", "iframe")

# Class/id tokens for sidebars, nav boxes, info boxes, comments, share/promo
# widgets. Matched as whole tokens (so "header" won't catch "headline") and dropped
# before extraction. Common across MediaWiki, WordPress, and most news CMSes.
NOISE_CLASSES = frozenset((
    "navbox", "sidebar", "vertical-navbox", "infobox", "hatnote", "mw-editsection",
    "reference", "reflist", "toc", "mw-jump-link", "noprint", "navigation",
    "breadcrumb", "breadcrumbs", "menu", "related", "related-articles", "comments",
    "comment", "share", "sharing", "social", "social-share", "cookie",
    "cookie-banner", "newsletter", "subscribe", "promo", "advertisement", "ads",
    "sponsor",
))
BLOCK_TAGS = ("p", "h1", "h2", "h3", "h4", "h5", "h6", "li", "blockquote", "pre")


def fetch_html(url):
    req = urllib.request.Request(url, headers={"User-Agent": UA,
                                               "Accept-Language": "en,zh;q=0.9"})
    with urllib.request.urlopen(req, timeout=30) as resp:
        final_url = resp.geturl()
        raw = resp.read()
        if resp.headers.get("Content-Encoding", "").lower() == "gzip":
            raw = gzip.GzipFile(fileobj=io.BytesIO(raw)).read()
        charset = resp.headers.get_content_charset() or "utf-8"
    return raw.decode(charset, errors="replace"), final_url


def extract_with_bs4(html):
    from bs4 import BeautifulSoup
    soup = BeautifulSoup(html, "html.parser")
    title = soup.title.get_text(strip=True) if soup.title else ""
    for tag in soup(list(BOILERPLATE_TAGS)):
        tag.decompose()
    for el in soup.find_all(attrs={"class": True}):
        try:
            if set(el.get("class") or []) & NOISE_CLASSES:
                el.decompose()
        except Exception:  # noqa: BLE001 — already detached by a parent's decompose
            pass
    for el in soup.select("[role=navigation], [role=note], [role=complementary]"):
        el.decompose()
    for br in soup.find_all("br"):       # legacy pages delimit paragraphs with <br>
        br.replace_with("\n")

    # Prefer the real article container. Semantic tags first, then the content-wrapper
    # class/id conventions most CMSes and wikis use; fall back to the body only when
    # none match (e.g. minimalist legacy pages).
    root = None
    for sel in ("article", "main", "[role=main]", "[itemprop=articleBody]",
                "#mw-content-text", ".mw-parser-output", ".post-content",
                ".entry-content", ".article-body", ".article__body", ".story-body"):
        node = soup.select_one(sel)
        if node and len(node.get_text(strip=True)) > 200:
            root = node
            break
    if root is None:
        root = soup.body or soup

    # Primary path: structured block extraction. Keeps headings (## markers, so the
    # skill can segment) and lists. Works for modern, well-marked-up HTML.
    lines = []
    for b in root.find_all(BLOCK_TAGS):
        txt = b.get_text(" ", strip=True)
        if not txt:
            continue
        if b.name in ("h1", "h2", "h3", "h4", "h5", "h6"):
            lines.append("\n## " + txt)
        elif b.name == "li":
            lines.append("- " + txt)
        else:
            lines.append(txt)
    body = "\n\n".join(lines)

    # Fallback path: legacy pages (e.g. <font> + <br>) where prose isn't in block
    # tags at all. Take the container's full text and split into paragraphs on the
    # newlines we inserted for <br> and the block boundaries.
    if len(body.split()) < THIN_WORD_THRESHOLD:
        raw = root.get_text()
        paras = [ln.strip() for ln in raw.split("\n") if len(ln.strip()) > 1]
        body = "\n\n".join(paras)
    return title, body, "bs4"


def extract_with_stdlib(html):
    """Fallback when bs4 is unavailable: crude tag strip via HTMLParser."""
    from html.parser import HTMLParser

    class Stripper(HTMLParser):
        def __init__(self):
            super().__init__()
            self.skip = 0
            self.title_mode = False
            self.title = ""
            self.parts = []

        def handle_starttag(self, tag, attrs):
            if tag in BOILERPLATE_TAGS:
                self.skip += 1
            if tag == "title":
                self.title_mode = True
            if tag in ("p", "br", "div", "li") + ("h1", "h2", "h3", "h4", "h5", "h6"):
                self.parts.append("\n")

        def handle_endtag(self, tag):
            if tag in BOILERPLATE_TAGS and self.skip:
                self.skip -= 1
            if tag == "title":
                self.title_mode = False

        def handle_data(self, data):
            if self.title_mode:
                self.title += data
            elif not self.skip and data.strip():
                self.parts.append(data.strip())

    p = Stripper()
    p.feed(html)
    text = " ".join(part for part in p.parts).replace(" \n ", "\n").strip()
    paras = [ln.strip() for ln in text.split("\n") if len(ln.strip()) > 1]
    return p.title.strip(), "\n\n".join(paras), "stdlib"


def main():
    if len(sys.argv) != 2:
        sys.stderr.write('{"error": "usage: fetch_source.py <url>"}\n')
        return 2
    url = sys.argv[1]
    try:
        html, final_url = fetch_html(url)
    except Exception as e:  # noqa: BLE001 — any fetch failure means "fall back".
        sys.stderr.write(json.dumps({"error": "fetch failed: %s" % e}) + "\n")
        return 1

    try:
        title, body, method = extract_with_bs4(html)
    except ImportError:
        title, body, method = extract_with_stdlib(html)
    except Exception as e:  # noqa: BLE001
        sys.stderr.write(json.dumps({"error": "extract failed: %s" % e}) + "\n")
        return 1

    word_count = len(body.split())
    thin = word_count < THIN_WORD_THRESHOLD
    sys.stderr.write(json.dumps({
        "method": method, "word_count": word_count, "thin": thin,
        "title": title, "final_url": final_url,
    }) + "\n")

    if title:
        sys.stdout.write(title + "\n\n")
    sys.stdout.write(body + "\n")
    return 0


if __name__ == "__main__":
    sys.exit(main())
