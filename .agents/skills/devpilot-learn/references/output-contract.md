# Output Contract

The final artifact is always a single self-contained `.html` file with inline CSS
only — no external stylesheets, no JavaScript, no network dependencies. It must open
correctly by double-clicking the file.

## The one mode: close-reading study artifact

This skill produces **one** kind of artifact: a single-column, Chinese-annotated
*close reading* of the source. It is not a digest and not a summary. The whole point
is that the reader can study the source's **actual words**, with help, rather than
read your compressed retelling of them.

Three pillars, in this rhythm, repeated section by section:

1. **Verbatim original** (`.orig`) — the source's real wording, copied faithfully.
   This is the primary prose and should dominate the page by volume.
2. **Chinese translation** (`.zh`) — a faithful translation sitting directly *below*
   each original passage (never in a second column). This is where interpretation
   lives; keep it accurate to the passage above it.
3. **Quizzes** — short `节后小测 / Section Quiz` after each section and a
   `总测 / Final Test` at the end, every answer hidden inside a `<details>` element so
   it stays collapsed until opened. **Quizzes follow the same bilingual rhythm as the
   passages**: each question appears in the source's original language with a faithful
   Chinese translation directly beneath it (exactly as `.orig` → `.zh`), the answer and
   explanation are given in both languages, and the structural labels are bilingual
   (`查看答案 / Show answer`, `答案/Answer`, `解析/Explanation`). When the source is
   already Chinese, the question needs no second line — same as a Chinese passage needs
   no translation.

Optional, used sparingly: `.note` glosses for individual terms or hard phrases that
genuinely need explanation.

## What "don't over-compress" means concretely

The previous version of this skill summarized the source and lost its substance. That
is the failure mode this contract exists to prevent.

- **Keep the original text.** Reproduce the source's substantive paragraphs verbatim
  in `.orig` blocks. Do not paraphrase them, do not reduce a multi-sentence paragraph
  to a single quoted line, and do not replace a section with a summary of it.
- **Coverage over brevity.** Walk the source front to back. A reader should be able to
  follow the source's full argument from your `.orig` blocks alone, in the source's
  own order. Skipping whole sections to save space defeats the purpose.
- **The artifact is normally larger than the source**, because it adds a translation
  and quizzes beneath text it has kept. If your output is dramatically shorter than the
  source, you have summarized instead of close-read — go back and restore the original
  passages.
- The only text you may safely drop is genuine non-content: running headers/footers,
  page numbers, copyright boilerplate, navigation, and pure repetition.

## Non-negotiable rules

- Single-column flow only. Never a side-by-side / two-column comparison layout — that
  framing pushes generation toward digest behavior.
- `.zh` must faithfully translate the `.orig` directly above it; do not add claims that
  aren't in that passage and do not editorialize.
- Preserve source terminology. For statutes, cases, proper nouns, product/model/org
  names, keep the original term and add a common Chinese rendering only when it helps.
- Build `.orig` from the source's real text, fetched verbatim. For URLs that means the
  bundled `scripts/fetch_source.py`, for PDFs `scripts/extract_pdf.py` — **never**
  `WebFetch`, which returns a summary of the page, not its words, so `.orig` built on it
  is compressed before you begin. If extraction genuinely fails and you must fall back to
  a summarized fetch, say so in the meta block (`基于摘要式抓取，非逐字原文 / Based on a
  summarized fetch, not verbatim`); never pass summary text off as the verbatim original.
- Coverage is gated mechanically, not by eye. Before saving you must run
  `scripts/check_coverage.py source.txt <artifact.html>` and it must pass. It reports
  which source sections/paragraphs are missing from your `.orig` blocks; restore them and
  re-run until it passes. A failing gate means the artifact is not done.
- Quizzes must be answerable from the source alone — never test outside knowledge.
- Quizzes are bilingual, never single-language. A question in only one language (or
  bilingual answer content under Chinese-only labels like a bare `节后小测` / `查看答案`)
  is a defect: pair the source-language question with a Chinese translation and use the
  bilingual labels above. The lone exception is a source that is already Chinese, where
  the question is simply Chinese.
- If the source is thin, paywalled, or partly inaccessible, degrade honestly: keep the
  close-reading structure over whatever text you actually have, and say so. Don't pad.
