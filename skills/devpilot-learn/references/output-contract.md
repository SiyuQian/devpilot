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
3. **Quizzes** — short `节后小测` after each section and a `总测` at the end, every
   answer hidden inside a `<details>` element so it stays collapsed until opened.

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
- Quizzes must be answerable from the source alone — never test outside knowledge.
- If the source is thin, paywalled, or partly inaccessible, degrade honestly: keep the
  close-reading structure over whatever text you actually have, and say so. Don't pad.
