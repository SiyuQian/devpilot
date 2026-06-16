---
name: devpilot-learn
description: >
  Turn a single article, document, or web page into a standalone HTML learning artifact:
  either a bilingual digest or a chaptered Chinese study guide with source summaries.
  MUST use this skill — not raw summarization — whenever the user asks to summarize,
  digest, get key points from, extract takeaways from, or turn a URL, PDF, .docx,
  .md, .txt, paper, report, or blog post into notes or study material. Trigger on:
  summarize this, key points, takeaways, digest, highlights, TL;DR, what does it say,
  overview, 学习资料, 学习笔记, 复习资料, 总结, 摘要, 要点, 重点, 提取, 整理, 讲了什么.
  NOT for: writing new content, translating, comparing two documents, building apps,
  multi-source news, code review, or data/CSV analysis.
---

# Learn

Generate a standalone HTML **close-reading study artifact** from a single source.

This skill has one job and one output shape. It does **not** summarize. It produces a
single-column, reading-oriented document that keeps the source's **actual words** and
helps the reader study them:

- **Verbatim original** — the source's substantive text, copied faithfully, is the
  primary prose and dominates the page.
- **Chinese translation** — a faithful translation sits directly *below* each original
  passage (never in a side-by-side column).
- **Quizzes** — a short `节后小测` after each section and a `总测` at the end, with every
  answer collapsed inside a `<details>` element.

The recurring failure mode this skill guards against is **over-compression**: turning the
source into a short digest and losing its substance. The artifact is normally *larger*
than the source, because it adds a translation and quizzes beneath text it has kept.

## Files in this skill

| File | When to load |
|---|---|
| `references/output-contract.md` | First — the output guarantees and hard constraints, especially the rule against over-compression. |
| `references/workflow.md` | During execution — source fetching, segmentation, generation flow, save rules, and edge cases. |
| `references/skeleton.md` | When ready to emit the HTML — the single-column verbatim-original → translation → quiz layout. |
| `scripts/fetch_source.py` | When the source is a URL — fetches the page's **verbatim** text. Use this instead of `WebFetch`, which returns a summary and silently compresses the source. |
| `scripts/extract_pdf.py` | When the source is a PDF — layout-aware **verbatim** extraction into real paragraphs + section headings (needs PyMuPDF). |
| `scripts/batch_sections.py` | When the source is large (≥ ~6k words or ≥ 25 sections) — groups it into ordered, word-budgeted batches so you can fan out one subagent per batch instead of emitting the whole artifact in one output-limited pass. |
| `scripts/check_coverage.py` | Before saving — the mechanical coverage gate; verifies every source section survives in the `.orig` blocks and names what to restore. |

## How to use this skill

1. Read `references/output-contract.md` before writing anything.
2. Follow `references/workflow.md` to fetch the source (verbatim, into `source.txt`),
   segment it, and generate the artifact.
3. Load `references/skeleton.md` for the layout when you are ready to emit the final HTML file.
4. Run `scripts/check_coverage.py source.txt <artifact.html>` before saving and restore
   anything it flags — passing the gate is required.

Keep `SKILL.md` as the entry point. Put detailed layout, workflow, and output spec
in the reference files above rather than re-expanding them inline.

**Helper dependencies (Python 3):** `fetch_source.py` uses BeautifulSoup (`bs4`) and
degrades to the standard library if it is absent; `extract_pdf.py` needs PyMuPDF
(`python3 -m pip install --user pymupdf`) and the workflow falls back to a plain `Read`
of the PDF if it is missing. The scripts never hard-block the skill.
