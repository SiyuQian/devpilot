# Workflow

Read `output-contract.md` first. This file is the step-by-step execution flow for
producing the close-reading artifact.

## 1. Identify and fetch the source

However the source arrives, capture its **verbatim** text into a working file
`source.txt` in the output directory. You close-read from that file, and the coverage
gate in step 5 checks your artifact against it — so the bundled fetch/extract helpers
write the text you must quote in `.orig`.

The user provides one of:

- **URL** — fetch the page's **verbatim** text with this skill's bundled helper, **not**
  with `WebFetch`. Run it by absolute path from the skill's base directory and save it:

  ```
  python3 <skill-dir>/scripts/fetch_source.py "<url>" > source.txt
  ```

  It writes the article's actual words to `source.txt` (this is the text your `.orig`
  blocks must quote) and a one-line JSON diagnostic to stderr.

  **Why not `WebFetch`:** `WebFetch` returns a model-generated *summary* of the page, not
  its words — for long articles it drops or paraphrases almost everything, so any `.orig`
  built on it is already compressed before you start. The helper exists to hand you the
  real text instead.

  Read the diagnostic and act on it:
  - `"thin": false` → trust stdout as the source text and close-read it.
  - `"thin": true`, a non-zero exit, or an `"error"` → extraction failed or the page was
    too sparse (paywall, JS-only rendering, bot wall). Only then fall back to `WebFetch`
    for whatever text is accessible, **and tag the artifact honestly**: add to the meta
    block `本资料基于摘要式抓取，非逐字原文 / Based on a summarized fetch, not verbatim`.
    Never present summarized text as if it were the verbatim original.
- **PDF** (`.pdf`) — extract the **verbatim** text with the bundled layout-aware helper,
  which reconstructs real paragraphs and marks section headings (`## heading`) so the
  coverage gate can see your section structure:

  ```
  python3 <skill-dir>/scripts/extract_pdf.py "<file.pdf>" > source.txt
  ```

  It needs PyMuPDF (`python3 -m pip install --user pymupdf`). Read the JSON diagnostic on
  stderr. If it exits non-zero (e.g. exit `3` = PyMuPDF missing) or reports `"thin": true`,
  fall back to reading the PDF with the `Read` tool — **read the whole document**, every
  section, and write that text to `source.txt` yourself. Either way you must end up with
  the full verbatim text on disk before writing.
- **Word doc** (`.docx`) — `textutil -convert txt <file> -stdout > source.txt` (macOS) or
  `pandoc <file> -t plain -o source.txt` as fallback.
- **Text / Markdown** (`.txt`, `.md`) or **pasted text** — copy it verbatim into
  `source.txt`.

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

**First decide whether to fan out.** Large sources don't fail the coverage gate, but
generating the whole artifact in one pass bumps the model's per-response output limit:
the agent gets terser toward the end and the run drags (a 12k-word source took ~37 min
serially and only finished by switching to a fragile append loop). When `source.txt` is
roughly **≥ 6,000 words or ≥ 25 sections** (check the extract diagnostic's `word_count`
/ `sections`, or `grep -c '^## ' source.txt`), build it with the section-batch fan-out
in **"Large sources"** below instead of one pass. Otherwise build it in one pass:

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

The recurring failure mode is summarizing instead of close-reading. A reader should be
able to follow the source's full argument from your `.orig` blocks alone, in order. If
whole sections of the source are missing, or your `.orig` blocks are one-line snippets
where the source had full paragraphs, you have summarized — restore the original text. The
artifact is normally **larger** than the source, not smaller. Step 5's coverage gate
enforces this mechanically; passing it is required, not optional.

### Large sources: fan out by section batch

When step 4 routed you here, split the work so no single agent has to emit the whole
artifact in one response. You stay the **orchestrator**: you own the shell and the final
test, and you dispatch one subagent per batch to produce the section fragments verbatim.

1. **Batch the source.** Run the bundled helper against `source.txt`:

   ```
   python3 <skill-dir>/scripts/batch_sections.py source.txt --max-words 2500 > batches.json
   ```

   It groups consecutive `## ` sections into ordered, word-budgeted batches (never
   splitting or reordering a section) and prints JSON: `num_batches` and a `batches`
   array, each with `index`, `headings`, `words`, and the verbatim `text` to reproduce.
   (If `source.txt` has no `## ` headings — e.g. an unstructured page — segment it into
   `## ` sections yourself first, per step 3, then re-run.)

2. **Assign anchor ids.** Walk every section heading across all batches in order and give
   it a stable id `sec-1`, `sec-2`, … You own these ids: the `目录` you build must link to
   them, so each subagent must use the exact id you hand it for each of its headings.

3. **Dispatch one subagent per batch, in parallel** (one message, multiple Agent calls —
   wall-clock then tracks the slowest batch, not the sum). Give each subagent:
   - this skill's `references/output-contract.md` and `references/skeleton.md` to read,
   - **only its batch's `text`** (its verbatim slice — it must not see or invent other
     sections), plus the `heading → sec-N` id map for its headings,
   - instructions to emit **only the section fragments** for its headings, in order:
     for each heading a `<h2 id="sec-N">…</h2>` followed by the `.passage`
     (verbatim `.orig` → faithful `.zh`), sparse `.note` glosses, and a bilingual
     `节后小测` — exactly as the one-pass per-section rules above require. **No**
     `<head>`, `<h1>`, `.meta`, `.toc`, `总测`, or `<body>` wrapper — fragments only.
   - a reminder to keep `.orig` blocks verbatim and large; it is reproducing, not
     summarizing, and it has output budget to spare because it only owns a few sections.

4. **Stitch.** Concatenate the returned fragments into `<main>` strictly in batch-index
   then heading order — this preserves source order. Wrap them in the shell from
   `skeleton.md`: `<head>`/`<style>`, `<h1>`, the `.meta` block, and a `.toc` linking to
   every `sec-N`.

5. **Write the final test yourself.** The `总测` spans the whole source, so build it from
   `source.txt` (which you hold) — or dispatch one extra subagent given the full
   `source.txt` for just the `总测`. Append it after the stitched sections.

The coverage gate in step 5 still runs against the assembled file and is still required —
fan-out changes how the artifact is produced, not the bar it must clear.

## 5. Run the coverage gate, then save

Before you save, **verify coverage mechanically** — don't rely on eyeballing it. Write
the artifact to its intended path, then run the gate against `source.txt`:

```
python3 <skill-dir>/scripts/check_coverage.py source.txt <artifact.html>
```

It prints per-section coverage and lists any source paragraphs missing from your `.orig`
blocks. **If it exits non-zero** (a section was dropped, or overall coverage is below
threshold), restore the listed paragraphs as new `.orig` + `.zh` passages in the right
section, then re-run until it passes. The gate already excludes references and
figure/table dumps, so the paragraphs it names are real prose you skipped — put them back,
don't argue with it.

Then save with the `Write` tool to the current working directory unless the user says
otherwise.

Naming convention:

- For files: `study-guide-[original-filename].html`
- For URLs: `study-guide-[slugified-domain-or-title].html`

Tell the user the saved file path. (The `source.txt` working file can be left in place or
removed; it is not the deliverable.)

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
