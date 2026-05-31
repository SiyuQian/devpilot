---
name: devpilot-learn
description: >
  Summarize any single article, document, or web page into a standalone bilingual
  HTML digest with an English | 中文 comparison layout and a terminal terminology
  glossary. MUST use this skill — not raw summarization — whenever the user asks
  to summarize, digest, get key points from, or extract takeaways from a URL, PDF,
  .docx, .md, .txt, paper, report, or blog post. The skill produces a shareable
  HTML file you cannot generate without it. Trigger on: summarize this, key points,
  takeaways, digest, highlights, TL;DR, what does it say, overview, 总结, 摘要,
  要点, 重点, 提取, 整理, 讲了什么. NOT for: writing new content, translating,
  comparing two documents, building apps, multi-source news, code review, or data/CSV
  analysis.
---

# Learn

Generate a standalone bilingual HTML digest from a single source. The output is a
fixed-format comparison document: desktop uses an `English | 中文` two-column layout,
mobile stacks the same content in that order, and the document ends with a four-column
terminology glossary.

This skill does **not** switch to single-language output. Even if the user asks for
English-only or Chinese-only output, still produce the bilingual digest.

## Files in this skill

| File | When to load |
|---|---|
| `references/output-contract.md` | First — the fixed bilingual artifact contract and the hard constraints that must never drift. |
| `references/workflow.md` | During execution — source handling, normalization, bilingual writing flow, save rules, and edge cases. |
| `references/html-skeleton.md` | When generating the final artifact — the concrete responsive HTML structure and style baseline. |

## How to use this skill

1. Read `references/output-contract.md` before writing anything.
2. Follow `references/workflow.md` for source-specific handling and section-by-section generation.
3. Load `references/html-skeleton.md` only when you are ready to emit the final HTML file.

Keep `SKILL.md` as the entry point. Put detailed layout, workflow, and output spec
in the reference files above rather than re-expanding them inline.
