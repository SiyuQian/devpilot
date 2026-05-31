# Output Contract

The final artifact must always be a single self-contained `.html` file with inline
CSS only.

The skill supports two output modes.

## Mode 1: Bilingual digest mode

Use this for general article/document summarization when the user wants a concise
digest rather than study material.

Required shape:

- A fixed bilingual digest with `English | 中文` content blocks
- Desktop: side-by-side columns
- Mobile: stacked blocks, `English` first and `中文` second
- A document with the sections below, in this exact order:
  1. Bilingual Title
  2. Metadata
  3. `At a Glance / 一眼速览`
  4. `Summary / 摘要`
  5. `Key Points / 要点`
  6. `Evidence & Data / 证据与数据`
  7. `Visuals / 图表与视觉要素`
  8. `Terminology Glossary / 术语对照表`

## Mode 2: Study-guide mode

Use this when the user asks for 学习资料 / notes / review material, or the source is
chaptered, legal, academic, technical, doctrinal, or exam-prep oriented.

Required shape:

- A Chinese-first, single-column study guide
- A visible source metadata block near the top
- A table of contents for major sections
- Major sections should follow the source's chapter or section order
- Every major section must begin with an explicit `原文摘要 / Source Summary` block
  that faithfully summarizes that source section
- Study-oriented callouts such as `重点`, `术语`, `法条/案例`, examples, or review notes
  should be used when the source supports them
- A closing `总结复习` section is required
- Short review questions are recommended when the source is substantial enough to support
  them, but they are not mandatory for every source
- The artifact should help the user study the original source, not merely skim a digest

## Non-negotiable rules

- Pick one mode deliberately; do not blend them into an incoherent hybrid
- Stay within what the source explicitly states or directly supports; do not add speculative
  interpretation
- If the source quality is poor or incomplete, degrade honestly while preserving the
  chosen mode's overall structure
- Preserve important source terminology faithfully; prefer common Chinese renderings
  when they preserve meaning
- For proper nouns such as company names, product names, model names, and organizations,
  keep the original term and add a common Chinese rendering only when useful
- In study-guide mode, `原文摘要 / Source Summary` must summarize the source section itself,
  not your downstream study advice
