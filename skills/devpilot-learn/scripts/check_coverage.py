#!/usr/bin/env python3
"""Section-level coverage gate: did the artifact keep every section, or drop some?

The skill's standing failure mode is *coverage compression* — the model keeps the
`.orig` blocks it writes verbatim, but silently skips whole sections of the source.
Instructions alone don't stop this. This gate measures coverage per source section and
names the paragraphs that went missing, so they can be restored before saving.

It relies on the source text carrying `## heading` markers (as emitted by extract_pdf.py
and fetch_source.py). End-matter — References / Bibliography / Acknowledgments and
everything after — is excluded, since the skill legitimately drops it.

Usage:
    python3 check_coverage.py <source.txt> <artifact.html> [--threshold 0.85]

Output (stdout): a per-section coverage table, then the source paragraphs absent from
every `.orig` block (the ones to restore).
Exit: 0 if overall coverage >= threshold AND no substantive section was fully dropped;
otherwise 1.
"""

import argparse
import re
import sys

# Paragraphs shorter than this (words) are captions/noise — not held to the coverage bar.
MIN_PARA_WORDS = 25
# A source paragraph is "covered" when a run of this many of its words appears in .orig.
WINDOW = 12
# Headings that begin end-matter the skill legitimately drops.
ENDMATTER = re.compile(
    r"^\s*(references|bibliography|works cited|参考文献|acknowledge?ments?|致谢)\b", re.I)


def normalize(text):
    text = re.sub(r"\s+", " ", text.lower())
    return re.sub(r"[^\w\s]", "", text)


_CAPTION = re.compile(r"^\s*(figure|fig\.?|table|algorithm|listing)\s*\d", re.I)
_CITE = re.compile(r"\([^)]*\b(?:19|20)\d{2}[^)]*\)")


def is_noncontent(para):
    """Figure/table captions, affiliations, and citation/taxonomy enumerations —
    not prose to close-read, so excluded from the coverage denominator."""
    words = para.split()
    if not words:
        return True
    if _CAPTION.match(para) or "@" in para:        # caption / email-affiliation line
        return True
    digits = sum(1 for w in words if re.fullmatch(r"[\d.,%§()/+-]+", w))
    if digits / len(words) > 0.25:                 # table dump
        return True
    cites = len(_CITE.findall(para))               # "Name (Author et al., 2022)" lists
    if cites >= 4 and len(words) / cites < 14:      # citation-dense enumeration, not prose
        return True
    return False


def parse_sections(path):
    """Return [(title, [paragraph, ...]), ...], stopping at end-matter."""
    raw = open(path, encoding="utf-8", errors="replace").read()
    sections, title, paras = [], "(preamble)", []
    for block in re.split(r"\n\s*\n", raw):
        block = block.strip()
        if not block:
            continue
        if block.startswith("## "):
            head = block[3:].strip()
            if ENDMATTER.match(head):
                if paras:
                    sections.append((title, paras))
                return sections  # cut end-matter and everything after
            if paras:
                sections.append((title, paras))
            title, paras = head, []
        else:
            paras.append(block)
    if paras:
        sections.append((title, paras))
    return sections


def artifact_orig_text(path):
    try:
        from bs4 import BeautifulSoup
        soup = BeautifulSoup(open(path, encoding="utf-8", errors="replace").read(),
                             "html.parser")
        blocks = [e.get_text(" ", strip=True) for e in soup.select(".orig")]
        if blocks:
            return " ".join(blocks)
    except ImportError:
        pass
    html = open(path, encoding="utf-8", errors="replace").read()
    blocks = re.findall(r'class=["\']orig["\'][^>]*>(.*?)</', html, re.S)
    return " ".join(re.sub(r"<[^>]+>", " ", b) for b in blocks)


def is_covered(para, orig_norm):
    words = normalize(para).split()
    if len(words) < WINDOW:
        return normalize(para) in orig_norm
    for start in (0, max(0, len(words) // 2 - WINDOW // 2), len(words) - WINDOW):
        if " ".join(words[start:start + WINDOW]) in orig_norm:
            return True
    return False


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("source")
    ap.add_argument("artifact")
    ap.add_argument("--threshold", type=float, default=0.85)
    args = ap.parse_args()

    sections = parse_sections(args.source)
    orig_norm = normalize(artifact_orig_text(args.artifact))

    total = covered = 0
    dropped_sections, missing = [], []
    rows = []
    for title, paras in sections:
        gradable = [p for p in paras
                    if len(p.split()) >= MIN_PARA_WORDS and not is_noncontent(p)]
        if not gradable:
            continue
        cov = [p for p in gradable if is_covered(p, orig_norm)]
        total += len(gradable)
        covered += len(cov)
        rows.append((title, len(cov), len(gradable)))
        if not cov:
            dropped_sections.append(title)
        missing.extend(p for p in gradable if p not in cov)

    if total == 0:
        print("No substantive prose sections found — check the source.txt path/format.")
        return 1
    ratio = covered / total

    print("Section coverage gate")
    for title, c, n in rows:
        flag = "  <-- DROPPED" if c == 0 else ("  (partial)" if c < n else "")
        print(f"  {c}/{n:<3} {title[:60]}{flag}")
    print(f"\n  overall: {covered}/{total} paragraphs = {ratio:.1%} "
          f"(threshold {args.threshold:.0%})")

    if missing:
        print("\nSource paragraphs NOT represented in any .orig block — restore these:")
        for i, p in enumerate(missing, 1):
            print(f"\n  [{i}] {' '.join(p.split())[:200]}...")

    failed = ratio < args.threshold or dropped_sections
    if failed:
        if dropped_sections:
            print("\nDropped sections: " + ", ".join(dropped_sections))
        print(f"\nFAIL: restore the missing paragraphs above, then re-run.")
        return 1
    print("\nPASS: coverage meets threshold and no section was dropped.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
