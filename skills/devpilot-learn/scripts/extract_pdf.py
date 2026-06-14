#!/usr/bin/env python3
"""Extract a PDF's text VERBATIM with layout awareness — real paragraphs and headings.

Plain text extraction (e.g. pypdf) of a multi-column or academic PDF returns page-blobs,
not paragraphs, so the model can't reliably copy `.orig` from it and a coverage gate
can't tell which sections were dropped. This uses PyMuPDF's font/coordinate data to
reconstruct paragraphs and detect section headings (emitted as `## heading`), giving the
skill clean, segmentable verbatim text.

Usage:
    python3 extract_pdf.py <file.pdf>

Output:
    stdout — title line, blank line, then body: `## heading` lines and blank-line-
             separated verbatim paragraphs, in reading order (column-aware).
    stderr — one JSON diagnostic line: {"method","word_count","pages","sections","thin","title"}.

Exit: 0 normal; 1 on read/parse failure; 3 if PyMuPDF is not installed (with a pip hint).
The caller (workflow.md) falls back to a plain `Read` of the PDF on a non-zero exit, and
tags the artifact if the fallback text is too thin to close-read.
"""

import json
import re
import sys
from collections import Counter

THIN_WORD_THRESHOLD = 120
# A line is a heading candidate when its font is this much larger than body text.
HEADING_SIZE_RATIO = 1.12
# Headings are short; longer lines at heading size are usually emphasized prose.
HEADING_MAX_WORDS = 14


def body_font_size(doc):
    """Most common span size, weighted by character count — i.e. the body text size."""
    sizes = Counter()
    for page in doc:
        for block in page.get_text("dict").get("blocks", []):
            for line in block.get("lines", []):
                for span in line.get("spans", []):
                    sizes[round(span["size"], 1)] += len(span["text"])
    return sizes.most_common(1)[0][0] if sizes else 10.0


def ordered_blocks(page):
    """Text blocks in reading order, column-aware for 2-column layouts."""
    blocks = [b for b in page.get_text("dict").get("blocks", []) if b.get("lines")]
    if not blocks:
        return []
    page_mid = page.rect.width / 2
    centers = [(b["bbox"][0] + b["bbox"][2]) / 2 for b in blocks]
    # Two columns if blocks sit clearly on both sides of the page midline.
    two_col = any(c < page_mid * 0.9 for c in centers) and any(c > page_mid * 1.1 for c in centers)
    def key(b):
        cx = (b["bbox"][0] + b["bbox"][2]) / 2
        col = 0 if (not two_col or cx < page_mid) else 1
        return (col, round(b["bbox"][1]))
    return sorted(blocks, key=key)


def block_text_and_size(block):
    lines, max_size = [], 0.0
    for line in block["lines"]:
        spans = line.get("spans", [])
        if spans:
            max_size = max(max_size, max(s["size"] for s in spans))
        lines.append("".join(s["text"] for s in spans))
    text = "\n".join(lines)
    text = re.sub(r"-\n(?=\w)", "", text)   # join hyphenated line breaks
    text = re.sub(r"\s*\n\s*", " ", text)    # collapse intra-paragraph line breaks
    return text.strip(), max_size


def main():
    if len(sys.argv) != 2:
        sys.stderr.write('{"error": "usage: extract_pdf.py <file.pdf>"}\n')
        return 1
    path = sys.argv[1]
    try:
        import fitz  # PyMuPDF
    except ImportError:
        sys.stderr.write(json.dumps({
            "error": "PyMuPDF not installed",
            "hint": "python3 -m pip install --user pymupdf",
        }) + "\n")
        return 3
    try:
        doc = fitz.open(path)
    except Exception as e:  # noqa: BLE001
        sys.stderr.write(json.dumps({"error": "open failed: %s" % e}) + "\n")
        return 1

    body = body_font_size(doc)
    title = (doc.metadata or {}).get("title") or ""
    page_count = len(doc)
    out, sections = [], 0
    for page in doc:
        for block in ordered_blocks(page):
            text, size = block_text_and_size(block)
            if not text:
                continue
            is_heading = (size >= body * HEADING_SIZE_RATIO
                          and len(text.split()) <= HEADING_MAX_WORDS)
            if is_heading:
                out.append("## " + text)
                sections += 1
            else:
                out.append(text)
    doc.close()

    body_text = "\n\n".join(out)
    word_count = len(body_text.split())
    sys.stderr.write(json.dumps({
        "method": "pymupdf", "word_count": word_count, "pages": page_count,
        "sections": sections, "thin": word_count < THIN_WORD_THRESHOLD, "title": title,
    }) + "\n")
    if title:
        sys.stdout.write(title + "\n\n")
    sys.stdout.write(body_text + "\n")
    return 0


if __name__ == "__main__":
    sys.exit(main())
