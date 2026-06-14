#!/usr/bin/env python3
"""Group a verbatim source.txt into ordered, word-budgeted batches for fan-out.

When a source is large, generating the whole close-reading artifact in one pass
bumps the model's per-response output limit: the agent gets terser toward the end
and the run drags. This helper splits the *already-extracted* verbatim text into
consecutive batches small enough that one subagent can reproduce each batch in full
without hitting that limit. The orchestrator dispatches one subagent per batch,
stitches the fragments back in source order, and runs the coverage gate as usual.

Input: a source.txt whose sections are marked with `## heading` lines (this is what
extract_pdf.py emits; for other sources, add `## ` headings or segment by topic
first). Anything before the first `## ` heading is treated as preamble and attached
to the first batch.

Output: a single JSON object on stdout:
  {
    "max_words": 2500,
    "total_words": 12121,
    "num_sections": 67,
    "num_batches": 6,
    "batches": [
      {"index": 0, "headings": ["...", "..."], "words": 2310, "text": "## ...\n\n..."},
      ...
    ]
  }
Each batch's "text" is the verbatim slice (heading lines included) the subagent
must reproduce. Batches never split a section and never reorder the source.

Usage:
  python3 batch_sections.py source.txt [--max-words N]

--max-words defaults to 2500 source words per batch. A single section larger than
the budget still becomes its own batch (never split mid-section); its word count
will exceed --max-words and that is expected.
"""

import argparse
import json
import re
import sys

HEADING = re.compile(r"^##\s+(.*\S)\s*$")


def split_sections(text):
    """Return (preamble, [(heading, body_text), ...]) preserving order."""
    lines = text.splitlines(keepends=True)
    preamble = []
    sections = []  # list of [heading, [lines...]]
    current = None
    for line in lines:
        m = HEADING.match(line.rstrip("\n"))
        if m:
            current = [m.group(1), [line]]
            sections.append(current)
        elif current is None:
            preamble.append(line)
        else:
            current[1].append(line)
    return "".join(preamble), [(h, "".join(b)) for h, b in sections]


def word_count(s):
    return len(s.split())


def batch(preamble, sections, max_words):
    batches = []
    cur_headings, cur_text, cur_words = [], [], 0

    def flush():
        nonlocal cur_headings, cur_text, cur_words
        if cur_headings:
            batches.append(
                {
                    "index": len(batches),
                    "headings": cur_headings,
                    "words": cur_words,
                    "text": "".join(cur_text).rstrip("\n") + "\n",
                }
            )
        cur_headings, cur_text, cur_words = [], [], 0

    for heading, body in sections:
        w = word_count(body)
        # Start a new batch when the current one is non-empty and would overflow.
        if cur_headings and cur_words + w > max_words:
            flush()
        cur_headings.append(heading)
        cur_text.append(body)
        cur_words += w
    flush()

    # Attach preamble (title/intro before the first heading) to the first batch.
    if preamble.strip() and batches:
        batches[0]["text"] = preamble.rstrip("\n") + "\n\n" + batches[0]["text"]
        batches[0]["words"] += word_count(preamble)
    return batches


def main():
    ap = argparse.ArgumentParser(description="Batch a source.txt for fan-out.")
    ap.add_argument("source", help="path to the verbatim source.txt")
    ap.add_argument("--max-words", type=int, default=2500,
                    help="target max source words per batch (default 2500)")
    args = ap.parse_args()

    with open(args.source, encoding="utf-8") as f:
        text = f.read()

    preamble, sections = split_sections(text)
    if not sections:
        print(json.dumps({"error": "no '## ' section headings found; "
                          "add headings or segment by topic first"}),
              file=sys.stderr)
        return 2

    batches = batch(preamble, sections, args.max_words)
    out = {
        "max_words": args.max_words,
        "total_words": word_count(text),
        "num_sections": len(sections),
        "num_batches": len(batches),
        "batches": batches,
    }
    json.dump(out, sys.stdout, ensure_ascii=False)
    sys.stdout.write("\n")
    return 0


if __name__ == "__main__":
    sys.exit(main())
