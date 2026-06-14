# Workflow

Read `output-contract.md` first. This file is the step-by-step execution flow for
producing the close-reading artifact.

## 1. Identify and fetch the source

The user provides one of:

- **URL** — use `WebFetch`. If the page is paywalled, mostly navigation, or too thin,
  tell the user the source is weak and continue only with what is actually accessible.
- **PDF** (`.pdf`) — use the `Read` tool with the file path. **Read the whole document**,
  not just the first pages. You cannot close-read text you never loaded, so cover every
  section before writing.
- **Word doc** (`.docx`) — convert with `textutil -convert txt <file> -stdout` (macOS) or
  `pandoc <file> -t plain` as fallback.
- **Text / Markdown** (`.txt`, `.md`) or **pasted text** — read directly.

For long sources, read in passes if needed, but do not start writing until you have seen
the full source end to end.

## 2. Normalize metadata

Identify, from the source itself:

- **Title** — the source's real title.
- **Content Type / 内容类型** — pick from: `Article / 文章`, `Report / 报告`,
  `Paper / 论文`, `PDF Document / PDF 文档`, `Web Page / 网页`,
  `Transcript / 访谈实录`, or `Document / 文档` when uncertain.
- **Source / 来源** — URL or filename (clickable in the meta block).
- **Author / 作者** and **Published / 发布时间** — only if the source states them.

## 3. Segment the source

Split the source into its natural sections, following its own chapter/heading/section
order. Each section becomes one `<h2>` block in the artifact. For an unstructured source,
segment by topic shift into a handful of coherent chunks. Do not flatten everything into
one undifferentiated block, and do not reorder the source.

Within each section, note:

- which paragraphs are substantive (these become `.orig` blocks — keep them verbatim)
- which terms or phrases genuinely need a `.note` gloss
- what a fair `节后小测` question would test, answerable from that section alone

## 4. Build the artifact

Load `skeleton.md` and follow its layout. For each section, in source order:

1. Reproduce the section's substantive paragraphs **verbatim** in `.orig` blocks. Keep
   large portions of the original — this is the heart of the artifact. Do not paraphrase
   or compress them into a single line.
2. Put a faithful Chinese translation in a `.zh` block directly below each `.orig`.
3. Add `.note` glosses sparingly, only where a term or phrase needs explanation.
4. End the section with a `节后小测 / Section Quiz` containing 1–3 questions, each answer
   hidden in a `<details>` element. Make the quiz **bilingual**, mirroring the
   `.orig` → `.zh` rhythm: write each question in the source's original language with a
   faithful Chinese translation directly beneath it, give the answer and explanation in
   both languages, and use the bilingual labels (`查看答案 / Show answer`, `答案/Answer`,
   `解析/Explanation`). If the source is already Chinese, the question stays Chinese-only —
   same as a Chinese passage needs no translation.

Then close with a `总测 / Final Test` spanning the whole source, same bilingual questions
and `<details>` answer pattern.

Keep the meta block at the top and a `目录` table of contents when the source has several
sections (drop it for short single-section sources).

### Guard against over-compression

The recurring failure mode is summarizing instead of close-reading. Before saving, sanity
-check: a reader should be able to follow the source's full argument from your `.orig`
blocks alone, in order. If whole sections of the source are missing, or your `.orig`
blocks are one-line snippets where the source had full paragraphs, you have summarized —
go back and restore the original text. The artifact is normally **larger** than the source,
not smaller.

## 5. Save and present

Save with the `Write` tool to the current working directory unless the user says otherwise.

Naming convention:

- For files: `study-guide-[original-filename].html`
- For URLs: `study-guide-[slugified-domain-or-title].html`

Tell the user the saved file path.

## Edge cases

- **Paywalled / weak content**: keep the close-reading structure over whatever text is
  accessible, and say the source was thin. Don't pad with invented content.
- **Very long sources**: still cover every section. Length is expected; do not collapse
  into an abstract.
- **Multiple sources in one request**: produce one artifact per source.
- **Non-prose inputs** (slides, tables, forms): keep the structure honestly, reproduce
  what text exists, and note when the source isn't argument-driven.
- **Very short source**: drop the `目录`, keep at least one `.orig`/`.zh` passage and one
  quiz; still close-read rather than summarize.
