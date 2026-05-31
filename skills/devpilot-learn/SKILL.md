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

Generate a standalone HTML learning artifact from a single source.

The skill supports two valid output modes:

- **Bilingual digest mode** — a fixed `English | 中文` comparison document for general
  summaries
- **Study-guide mode** — a Chinese-first, chaptered learning handout for long or
  educational sources, with explicit `原文摘要 / Source Summary` blocks to help review

When the user asks for 学习资料 / notes / review material, or the source is chaptered,
legal, academic, technical, doctrinal, or exam-prep oriented, prefer **study-guide mode**.

## Files in this skill

| File | When to load |
|---|---|
| `references/output-contract.md` | First — mode selection, output guarantees, and the hard constraints that must never drift. |
| `references/workflow.md` | During execution — source handling, mode selection, generation flow, save rules, and edge cases. |
| `references/html-skeleton.md` | Use for bilingual digest mode — the responsive comparison-layout baseline. |
| `references/study-guide-skeleton.md` | Use for study-guide mode — the chaptered learning-material layout with `原文摘要`. |

## How to use this skill

1. Read `references/output-contract.md` before writing anything.
2. Follow `references/workflow.md` to choose the correct mode and generate the right structure.
3. Load only the skeleton file for the chosen mode when you are ready to emit the final HTML file.

Keep `SKILL.md` as the entry point. Put detailed layout, workflow, and output spec
in the reference files above rather than re-expanding them inline.
